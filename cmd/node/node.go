//go:build (windows && (amd64 || 386 || arm64)) || (linux && (amd64 || arm || armv7l || arm64 || ppc64le || s390x)) || (darwin && (amd64 || arm64))

package node

import (
	"encoding/json"
	"fmt"
	version2 "github.com/hashicorp/go-version"
	"github.com/spf13/cobra"
	"io"
	"jianggujin.com/lvs/internal/config"
	"jianggujin.com/lvs/internal/invoke"
	"jianggujin.com/lvs/internal/util"
	"net/http"
	"runtime"
	"sort"
	"strings"
	"time"
)

type Command struct {
	command *cobra.Command
}

var nodeCmd = &Command{
	command: &cobra.Command{
		Use:   "node",
		Short: "Node.js version management",
	},
}

func Init(rootCmd *cobra.Command) {
	rootCmd.AddCommand(nodeCmd.command)
}

func (c *Command) AddCommand(commands ...util.Command) {
	for _, cmd := range commands {
		util.AddCommand(c.command, cmd)
	}
}

func (c *Command) Version() string {
	str, err := invoke.GetInvoker().Command("node", "-v")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(str))
}

func (c *Command) NewHttpClient(opts ...util.HttpClientOption) *http.Client {
	ops := append([]util.HttpClientOption{util.WithProxyStr(config.GetStringWithDefault(config.KeyNodeProxy, config.GetString(config.KeyLvsProxy)))}, opts...)
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
	Version  string   `json:"version"`  //  Node.js 版本号，如 v20.9.0
	Date     string   `json:"date"`     // 版本的发布日期，如 2023-10-03
	Npm      string   `json:"npm"`      // 随附的 npm 版本，如 10.2.3
	V8       string   `json:"v8"`       // Node.js 内部使用的 V8 引擎版本，如 11.8.172.21
	Uv       string   `json:"uv"`       // libuv 库的版本（用于异步 I/O 处理），如 1.44.2
	Zlib     string   `json:"zlib"`     // 内部使用的 zlib 压缩库版本，如 1.3
	Openssl  string   `json:"openssl"`  // 使用的 OpenSSL 版本，如 3.0.12
	Modules  string   `json:"modules"`  // Node.js 版本的 process.versions.modules 值（用于与本机 C++ 模块 ABI 兼容），如 115
	Lts      any      `json:"lts"`      // 是否为 LTS（长期支持）版本，false 表示非 LTS 版本，或者是一个字符串（如 "Hydrogen"），表示 LTS 代号
	Security bool     `json:"security"` // 是否为安全修复版本，true 表示该版本包含安全更新
	Files    []string `json:"files"`    // 版本支持的操作系统和架构的二进制文件，如 darwin-arm64（Mac M1/M2）、linux-x64（Linux x64）等
	semver   *version2.Version
}

func (v *Version) Semver() (*version2.Version, error) {
	if v.semver == nil {
		semver, err := version2.NewVersion(v.Version)
		if err != nil {
			return nil, err
		}
		v.semver = semver
	}
	return v.semver, nil
}

func (c *Command) Semver(version string) (*version2.Version, error) {
	return version2.NewVersion(version)
}

func (c *Command) FixVersion(version string) string {
	if version[0] != 'v' {
		version = "v" + version
	}
	return version
}

func (c *Command) RawVersion(version *version2.Version) string {
	return version.Original()
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
	var fileName string
	switch runtime.GOOS {
	case "windows":
		switch runtime.GOARCH {
		case "amd64":
			fileName = "win-x64-zip"
		case "386":
			fileName = "win-x86-zip"
		case "arm64":
			fileName = "win-arm64-zip"
		}
	case "darwin":
		switch runtime.GOARCH {
		case "amd64":
			fileName = "osx-x64-tar"
		case "arm64":
			fileName = "osx-arm64-tar"
		}
	case "linux":
		switch runtime.GOARCH {
		case "amd64":
			fileName = "linux-x64"
		case "arm", "armv7l":
			fileName = "linux-armv7l"
		case "arm64":
			fileName = "linux-arm64"
		case "ppc64le":
			fileName = "linux-ppc64le"
		case "s390x":
			fileName = "linux-s390x"
		}
	}
	if fileName == "" {
		return nil, fmt.Errorf("unsupported architecture: %s", runtime.GOARCH)
	}

	fetchUrl := config.GetString(config.KeyNodeMirror) + "index.json"
	resp, err := c.Get(fetchUrl, util.WithTimeout(30*time.Second))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	var versions []*Version
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(data, &versions); err != nil {
		return nil, err
	}
	defaultFilter := func(nv *Version) (bool, error) {
		if len(nv.Files) == 0 {
			return false, nil
		}
		has := false
		for _, fName := range nv.Files {
			if fName == fileName {
				has = true
				break
			}
		}
		if !has {
			return false, nil
		}
		if _, err = nv.Semver(); err != nil {
			return false, nil
		}
		if filter != nil {
			return filter(nv)
		}
		return true, nil
	}
	list, err := Collection(versions).Filter(defaultFilter)
	if err != nil {
		return nil, err
	}
	sort.Sort(list)
	return list, nil
}

type Download struct {
	Version  string
	BaseName string
	Ext      string
}

func (c *Command) ConvertDownload(version string) (*Download, error) {
	var name string
	var ext string
	switch runtime.GOOS {
	case "windows":
		ext = "zip"
		switch runtime.GOARCH {
		case "amd64":
			name = fmt.Sprintf("node-%s-win-x64", version) // https://nodejs.org/dist/v22.14.0/node-v22.14.0-win-x64.zip
		case "386":
			name = fmt.Sprintf("node-%s-win-x86", version) // https://nodejs.org/dist/v22.14.0/node-v22.14.0-win-x86.zip
		case "arm64":
			name = fmt.Sprintf("node-%s-win-arm64", version) // https://nodejs.org/dist/v22.14.0/node-v22.14.0-win-arm64.zip
		}
	case "darwin":
		ext = "tar.xz"
		//exts = []string{"tar.xz", "tar.gz"}
		switch runtime.GOARCH {
		case "amd64":
			name = fmt.Sprintf("node-%s-darwin-x64", version) // https://nodejs.org/dist/v22.14.0/node-v22.14.0-darwin-x64.tar.gz
		case "arm64":
			name = fmt.Sprintf("node-%s-darwin-arm64", version) // https://nodejs.org/dist/v22.14.0/node-v22.14.0-darwin-arm64.tar.gz
		}
	case "linux":
		ext = "tar.gz"
		// ext = []string{"tar.gz", "tar.xz"}
		// linux gz > xz
		switch runtime.GOARCH {
		case "amd64":
			name = fmt.Sprintf("node-%s-linux-x64", version) // https://nodejs.org/dist/v22.14.0/node-v22.14.0-linux-x64.tar.xz
		case "arm", "armv7l":
			name = fmt.Sprintf("node-%s-linux-armv7l", version) // https://nodejs.org/dist/v22.14.0/node-v22.14.0-linux-armv7l.tar.xz
		case "arm64":
			name = fmt.Sprintf("node-%s-linux-arm64", version) // https://nodejs.org/dist/v22.14.0/node-v22.14.0-linux-arm64.tar.xz
		case "ppc64le":
			name = fmt.Sprintf("node-%s-linux-ppc64le", version) // https://nodejs.org/dist/v22.14.0/node-v22.14.0-linux-ppc64le.tar.xz
		case "s390x":
			name = fmt.Sprintf("node-%s-linux-s390x", version) // https://nodejs.org/dist/v22.14.0/node-v22.14.0-linux-s390x.tar.xz
		}
	}
	if name == "" {
		return nil, fmt.Errorf("unsupported architecture: %s", runtime.GOARCH)
	}

	return &Download{
		Version:  version,
		BaseName: name,
		Ext:      ext,
	}, nil
}
