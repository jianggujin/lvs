package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"jianggujin.com/lvs/internal/config"
	"jianggujin.com/lvs/internal/install"
	"jianggujin.com/lvs/internal/util"
	"os"
	"strings"
)

type UninstallCommand struct {
	all bool
}

func init() {
	util.AddCommand(rootCmd, &UninstallCommand{})
}

func (command *UninstallCommand) Init() *cobra.Command {
	var modules []string
	for _, module := range config.Modules {
		modules = append(modules, fmt.Sprintf("[%s]", module.Name))
	}
	cmd := &cobra.Command{
		Use:     "uninstall",
		Short:   "Uninstall LVS and delete relevant environment information, default to uninstalling all modules",
		Example: fmt.Sprintf("%s uninstall %s", config.Name(), strings.Join(modules, " ")),
		RunE:    command.RunE,
	}
	flags := cmd.Flags()
	flags.BoolVarP(&command.all, "all", "a", false, "uninstall all modules with lvs")
	return cmd
}

func (command *UninstallCommand) RunE(_ *cobra.Command, modules []string) error {
	var envKeys []string
	var pathValues []string
	var symlinks []string
	if command.all {
		for _, module := range config.Modules {
			keyValues := module.EnvKeyValues
			if keyValues != nil {
				for k := range keyValues {
					envKeys = append(envKeys, k)
				}
			}
			values := module.PathValues
			if len(values) > 0 {
				pathValues = append(pathValues, values...)
			}
			symlinks = append(symlinks, os.Getenv(module.SymlinkEnvKey))
		}
		envKeys = append(envKeys, config.EnvLvsHome)
		pathValues = append(pathValues, fmt.Sprintf("%%%s%%", config.EnvLvsHome))
	} else if len(modules) > 0 {
		for _, module := range modules {
			m := config.Modules[module]
			if m == nil {
				return util.WrapErrorMsg("module [%s] is illegal", module)
			}
			keyValues := m.EnvKeyValues
			if keyValues != nil {
				for k, _ := range keyValues {
					envKeys = append(envKeys, k)
				}
			}
			values := m.PathValues
			if len(values) > 0 {
				pathValues = append(pathValues, values...)
			}
			symlinks = append(symlinks, os.Getenv(m.SymlinkEnvKey))
		}
	}

	if err := install.Uninstall(envKeys, pathValues); err != nil {
		return util.WrapErrorMsg("uninstalling failed, please try again").SetErr(err)
	}
	for _, symlink := range symlinks {
		_ = os.Remove(symlink)
	}
	fmt.Println("uninstall complete")
	return nil
}
