//go:build (windows && (amd64 || 386 || arm64)) || (linux && (amd64 || arm || armv7l || arm64 || ppc64le || s390x)) || (darwin && (amd64 || arm64))

package node

import (
	"fmt"
	"github.com/spf13/cobra"
	"jianggujin.com/lvs/internal/config"
	"jianggujin.com/lvs/internal/install"
	"jianggujin.com/lvs/internal/invoke"
	"jianggujin.com/lvs/internal/util"
	"os"
	"path/filepath"
	"runtime"
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
	version := nodeCmd.FixVersion(config.GetStringWithDefault(config.KeyNodeAliasPrefix+versions[0], versions[0]))
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
