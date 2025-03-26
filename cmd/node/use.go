//go:build (windows && (amd64 || 386 || arm64)) || (linux && (amd64 || arm || armv7l || arm64 || ppc64le || s390x)) || (darwin && (amd64 || arm64))

package node

import (
	"fmt"
	version2 "github.com/hashicorp/go-version"
	"github.com/spf13/cobra"
	"jianggujin.com/lvs/internal/config"
	"jianggujin.com/lvs/internal/install"
	"jianggujin.com/lvs/internal/invoke"
	"jianggujin.com/lvs/internal/util"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

func init() {
	nodeCmd.AddCommand(&UseCommand{})
}

type UseCommand struct {
}

func (command *UseCommand) Init() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "use",
		Short:   "Activate the specified version of node.js",
		Aliases: []string{"u"},
		RunE:    command.RunE,
	}
	return cmd
}

func (command *UseCommand) RunE(_ *cobra.Command, versions []string) error {
	if len(versions) == 0 {
		version, err := config.GetWorkspaceUseVersion(config.ModuleNode)
		if err != nil && !os.IsNotExist(err) {
			return util.WrapError(err)
		}
		if version == "" {
			return nil
		}
		versions = []string{version}
	}

	installHome := config.GetPath(config.KeyNodeHome)

	version, err := command.convertVersion(installHome, versions[0])
	if err != nil {
		return err
	}
	version = nodeCmd.FixVersion(version)

	if _, err := nodeCmd.Semver(version); err != nil {
		return util.WrapErrorMsg("[%s] is not a valid version", version)
	}
	if version == nodeCmd.Version() {
		fmt.Printf("[%s] has been activated\n", version)
		return nil
	}
	name := "node"
	if runtime.GOOS == "windows" {
		name = "node.exe"
	}
	dir := filepath.Join(installHome, version)
	var path string
	if runtime.GOOS == "windows" {
		path = filepath.Join(dir, name)
	} else {
		path = filepath.Join(dir, "bin", name)
	}

	if !util.Exists(path) {
		return util.WrapErrorMsg("[%s] not found", version)
	}
	if err := util.ResetSymlink(config.GetPath(config.KeyNodeSymlink), dir, true); err != nil {
		return util.WrapErrorMsg("reset symlink error").SetErr(err)
	}

	m := config.Modules[config.ModuleNode]
	var installErr error
	// 需要安装
	if os.Getenv(m.SymlinkEnvKey) != config.GetPath(config.KeyNodeSymlink) {
		envKeyValues := make(map[string]string)
		var pathValues []string

		keyValues := m.EnvKeyValues
		if keyValues != nil {
			for k, v := range keyValues {
				envKeyValues[k] = v
			}
		}
		values := m.PathValues
		if len(values) > 0 {
			pathValues = append(pathValues, values...)
		}
		installErr = install.Install(envKeyValues, pathValues)
	}

	if runtime.GOOS != "windows" {
		// 只修改 /opt/ 目录下的文件**（不递归）
		// chmod +x /opt/*
		// 递归修改所有文件**（但不影响目录的权限
		// find /opt/ -type f -exec chmod +x {} +
		// find /path/to/directory -type f -exec chmod +x {} \
		// 递归修改所有文件和目录**（包括目录的执行权限）
		// chmod -R +x /opt/
		if _, execErr := invoke.GetInvoker().Command("chmod", "-R", "+x", filepath.Base(path)+"/"); execErr != nil {
			return util.WrapErrorMsg("failed to grant executable permissions").SetErr(execErr)
		}
	}

	pass := false
	checkCount := 0
	for {
		if version == nodeCmd.Version() {
			pass = true
			break
		}
		checkCount++
		if checkCount > 10 {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	if pass {
		if installErr != nil {
			return util.WrapErrorMsg("[%s] has been activated. but installation failed", version).SetErr(installErr)
		} else {
			fmt.Printf("[%s] has been activated\n", version)
		}
	} else {
		if installErr != nil {
			return util.WrapErrorMsg("unable to obtain [%s] activation status. installation failed", version).SetErr(installErr)
		} else {
			if runtime.GOOS == "windows" {
				fmt.Printf("unable to obtain [%s] activation status, please try restarting the terminal\n", version)
			} else {
				fmt.Printf("unable to obtain [%s] activation status, please try restarting the terminal or run 'source %s'\n",
					version, config.GetString(config.KeyShellConfigPath))
			}
		}
	}
	return nil
}

func (command *UseCommand) convertVersion(installHome, version string) (string, error) {
	version = config.GetStringWithDefault(config.KeyNodeAliasPrefix+version, version)

	var filter func(*version2.Version) bool

	if "latest" == version {
		filter = func(*version2.Version) bool {
			return true
		}
	} else {
		version = nodeCmd.FixVersion(version)
		semver, err := nodeCmd.Semver(version)
		if err != nil {
			return version, fmt.Errorf("[%s] is not a valid version", version)
		}

		if semver.Prerelease() == "" && semver.Metadata() == "" && version != nodeCmd.FixVersion(semver.Core().String()) {
			segments := semver.Segments()
			for i := len(segments) - 1; i >= 0; i-- {
				if segments[i] != 0 {
					segments[i] = segments[i] + 1
					break
				}
			}
			fmtParts := make([]string, len(segments))
			for i, s := range segments {
				fmtParts[i] = strconv.FormatInt(int64(s), 10)
			}
			constraints, err := version2.NewConstraint(fmt.Sprintf(">=%s,<%s", semver.String(), strings.Join(fmtParts, ".")))
			if err != nil {
				return version, fmt.Errorf("[%s] is not a valid version", version)
			}
			filter = func(version *version2.Version) bool {
				return constraints.Check(version)
			}
		}
	}
	if filter != nil {
		entries, err := os.ReadDir(installHome)
		if err != nil {
			if !os.IsNotExist(err) {
				return version, util.WrapErrorMsg("find local installed version error").SetErr(err)
			}
		}
		var vers []*version2.Version
		if len(entries) > 0 {
			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}
				if ver, _ := nodeCmd.Semver(entry.Name()); ver != nil && filter(ver) {
					vers = append(vers, ver)
				}
			}
		}
		if len(vers) > 1 {
			sort.Sort(version2.Collection(vers))
		}
		if len(vers) == 0 {
			return version, fmt.Errorf("unable to find a version that matches the criteria [%s]", version)
		}
		version = nodeCmd.RawVersion(vers[len(vers)-1])
	}
	return version, nil
}
