package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"jianggujin.com/lvs/internal/config"
	"jianggujin.com/lvs/internal/install"
	"jianggujin.com/lvs/internal/util"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type InstallCommand struct {
	all bool
}

func init() {
	util.AddCommand(rootCmd, &InstallCommand{})
}

func (command *InstallCommand) Init() *cobra.Command {
	var modules []string
	for _, module := range config.Modules {
		modules = append(modules, fmt.Sprintf("[%s]", module.Name))
	}
	cmd := &cobra.Command{
		Use:     "install",
		Short:   "Install and set the required environment information, default to installing all modules",
		Aliases: []string{"i"},
		Example: fmt.Sprintf("%s install %s", config.Name(), strings.Join(modules, " ")),
		RunE:    command.RunE,
	}
	flags := cmd.Flags()
	flags.BoolVarP(&command.all, "all", "a", false, "install all modules with lvs")
	return cmd
}

func (command *InstallCommand) RunE(_ *cobra.Command, modules []string) error {
	envKeyValues := make(map[string]string)
	var pathValues []string
	if command.all {
		for _, module := range config.Modules {
			keyValues := module.EnvKeyValues
			if keyValues != nil {
				for k, v := range keyValues {
					envKeyValues[k] = v
				}
			}
			values := module.PathValues
			if len(values) > 0 {
				pathValues = append(pathValues, values...)
			}
		}
	} else if len(modules) > 0 {
		for _, module := range modules {
			m := config.Modules[module]
			if m == nil {
				return util.WrapErrorMsg("module [%s] is illegal", module)
			}
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
		}
	}

	homeDir := os.Getenv(config.EnvLvsHome)
	// 已设置过环境变量，验证是否合法
	needResetHome := true
	if homeDir != "" {
		targetPath := filepath.Join(homeDir, filepath.Base(os.Args[0]))
		if runtime.GOOS == "windows" && !strings.HasSuffix(targetPath, ".exe") {
			targetPath = targetPath + ".exe"
		}

		if _, err := os.Stat(targetPath); err != nil {
			if !os.IsNotExist(err) {
				return util.WrapErrorMsg("failed to detect LVS installed directory [%s]", homeDir).SetErr(err)
			}
		} else {
			needResetHome = false
		}
	} else {
		needResetHome = !isLvsAvailable()
	}

	if needResetHome {
		targetPath, err := filepath.Abs(os.Args[0])
		if err != nil {
			return util.WrapErrorMsg("failed to obtain LVS absolute path").SetErr(err)
		}
		if runtime.GOOS == "windows" && !strings.HasSuffix(targetPath, ".exe") {
			targetPath = targetPath + ".exe"
		}
		if !util.Exists(targetPath) {
			return util.WrapErrorMsg("LVS program not detected in absolute path [%s], please enter the directory where LVS program is located for installation", targetPath)
		}
		envKeyValues[config.EnvLvsHome] = filepath.Dir(targetPath)
		pathValues = append(pathValues, fmt.Sprintf("%%%s%%", config.EnvLvsHome))
	}
	if err := install.Install(envKeyValues, pathValues); err != nil {
		return util.WrapErrorMsg("installation failed, please try again").SetErr(err)
	}
	if runtime.GOOS == "windows" {
		fmt.Println("installation completed, if unable to use normally, please try restarting the terminal")
	} else {
		fmt.Printf("installation completed, if unable to use normally, please try restarting the terminal or run 'source %s'\n",
			config.GetString(config.KeyShellConfigPath))
	}
	return nil
}

func isLvsAvailable() bool {
	_, err := exec.LookPath("lvs")
	return err == nil
}
