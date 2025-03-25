package gom

import (
	"fmt"
	"github.com/spf13/cobra"
	"jianggujin.com/lvs/internal/config"
	"jianggujin.com/lvs/internal/util"
	"os"
	"path/filepath"
)

func init() {
	goCmd.AddCommand(&UninstallCommand{})
}

type UninstallCommand struct {
}

func (command *UninstallCommand) Init() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall the installed version of Go",
		RunE:  command.RunE,
	}
	return cmd
}

func (command *UninstallCommand) RunE(_ *cobra.Command, versions []string) error {
	installHome := config.GetPath(config.KeyGoHome)
	for i, version := range versions {
		if i > 0 {
			fmt.Println()
		}

		fmt.Printf("uninstall %s start\n", version)
		version = goCmd.FixVersion(version)
		if _, err := goCmd.Semver(version); err != nil {
			return util.WrapErrorMsg("[%s] is not a valid version", version)
		}
		path := filepath.Join(installHome, version)
		if util.Exists(path) {
			if err := os.RemoveAll(path); err != nil {
				return util.WrapErrorMsg("uninstall %s error", version).SetErr(err)
			}
		}
		fmt.Printf("uninstall %s finish\n", version)
	}
	return nil
}
