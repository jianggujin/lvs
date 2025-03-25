package gom

import (
	"fmt"
	version2 "github.com/hashicorp/go-version"
	"github.com/spf13/cobra"
	"io"
	"jianggujin.com/lvs/internal/config"
	"jianggujin.com/lvs/internal/invoke"
	"jianggujin.com/lvs/internal/util"
	"net/http"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"
)

type Command struct {
	command *cobra.Command
}

var goCmd = &Command{
	command: &cobra.Command{
		Use:   "go",
		Short: "Go version management",
	},
}

func Init(rootCmd *cobra.Command) {
	rootCmd.AddCommand(goCmd.command)
}

func (c *Command) AddCommand(commands ...util.Command) {
	for _, cmd := range commands {
		util.AddCommand(c.command, cmd)
	}
}

func (c *Command) Version() string {
	// go version go1.22.0 linux/amd64
	str, err := invoke.GetInvoker().Command("go", "version")
	if err != nil {
		return ""
	}
	re := regexp.MustCompile(`go(\d+\.\d+(\.\d+)?)`)
	match := re.FindStringSubmatch(string(str))
	if len(match) < 2 {
		return ""
	}
	return match[0]
}

func (c *Command) NewHttpClient(opts ...util.HttpClientOption) *http.Client {
	ops := append([]util.HttpClientOption{util.WithProxyStr(config.GetStringWithDefault(config.KeyGoProxy, config.GetString(config.KeyLvsProxy)))}, opts...)
	return util.NewHttpClient(ops...)
}

func (c *Command) Get(url string, opts ...util.HttpClientOption) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", fmt.Sprintf("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36 LVS/%s", config.BuildVersion))
	return c.NewHttpClient(opts...).Do(req)
}

type Version struct {
	Version string `json:"version"` // Go 版本号，例如 "go1.22.0"
	Size    string `json:"size"`
	Sha256  string `json:"sha256"`
	semver  *version2.Version
}

func (v *Version) Semver() (*version2.Version, error) {
	if v.semver == nil {
		version, _ := strings.CutPrefix(v.Version, "go")
		semver, err := version2.NewVersion(version)
		if err != nil {
			return nil, err
		}
		v.semver = semver
	}
	return v.semver, nil
}

func (c *Command) Semver(version string) (*version2.Version, error) {
	version, _ = strings.CutPrefix(version, "go")
	return version2.NewVersion(version)
}

func (c *Command) FixVersion(version string) string {
	if len(version) > 2 && version[:2] != "go" {
		version = "go" + version
	}
	return version
}

func (c *Command) RawVersion(version *version2.Version) string {
	return "go" + version.Original()
}

type Collection []*Version

func (v Collection) Len() int {
	return len(v)
}

func (v Collection) Less(i, j int) bool {
	return v[i].semver.GreaterThan(v[j].semver)
}

func (v Collection) Swap(i, j int) {
	v[i], v[j] = v[j], v[i]
}

func (v Collection) Filter(filter func(*Version) (bool, error)) (Collection, error) {
	if filter == nil {
		return v, nil
	}
	result := make([]*Version, 0, len(v))
	for _, version := range v {
		ok, err := filter(version)
		if err != nil {
			return nil, err
		}
		if ok {
			result = append(result, version)
		}
	}
	return result, nil
}

func (v Collection) Find(filter func(*Version) (bool, error)) (*Version, error) {
	if filter == nil {
		return nil, nil
	}
	for _, version := range v {
		ok, err := filter(version)
		if err != nil {
			return nil, err
		}
		if ok {
			return version, nil
		}
	}
	return nil, nil
}

func (c *Command) ListVersions(filter func(*Version) (bool, error)) ([]*Version, error) {
	// 不使用?mode=json是因为返回数据不全，改为提取HTML信息
	fetchUrl := config.GetString(config.KeyGoMirror)
	resp, err := c.Get(fetchUrl, util.WithTimeout(30*time.Second))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	content, err := io.ReadAll(resp.Body)
	re := regexp.MustCompile(`<tr[^>]*>\s*<td[^>]*>\s*<a[^>]*>([^<]+)</a>\s*</td>\s*<td[^>]*>([^<]*)</td>\s*<td[^>]*>[^<]*</td>\s*<td[^>]*>[^<]*</td>\s*<td[^>]*>([^<]*)</td>\s*<td[^>]*>\s*<tt>([^<]*)</tt>\s*</td>\s*</tr>`)
	matches := re.FindAllStringSubmatch(string(content), -1)
	var versions []*Version
	ext := ".tar.gz"
	if "windows" == runtime.GOOS {
		ext = ".zip"
	}
	suffix := fmt.Sprintf(".%s-%s%s", runtime.GOOS, runtime.GOARCH, ext)
	for _, item := range matches {
		name := strings.TrimSpace(item[1])
		if !strings.HasSuffix(name, suffix) {
			continue
		}
		kind := strings.TrimSpace(strings.ToLower(item[2]))
		if kind != "archive" {
			continue
		}
		version := &Version{
			Version: name[0 : len(name)-len(suffix)],
			Size:    strings.TrimSpace(item[3]),
			Sha256:  strings.TrimSpace(item[4]),
		}
		if _, err = version.Semver(); err != nil {
			continue
		}
		versions = append(versions, version)
	}

	defaultFilter := func(nv *Version) (bool, error) {
		if filter != nil {
			return filter(nv)
		}
		return true, nil
	}
	sort.Sort(Collection(versions))
	return Collection(versions).Filter(defaultFilter)
}

type Download struct {
	Version  string
	BaseName string
	Ext      string
}

func (c *Command) ConvertDownload(version string) (*Download, error) {
	ext := "tar.gz"
	if runtime.GOOS == "windows" {
		ext = "zip"
	}
	return &Download{
		Version:  version,
		BaseName: fmt.Sprintf("%s.%s-%s", version, runtime.GOOS, runtime.GOARCH),
		Ext:      ext,
	}, nil
}
