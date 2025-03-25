//go:build (windows && (amd64 || 386 || arm64)) || (linux && (amd64 || arm || armv7l || arm64 || ppc64le || s390x)) || (darwin && (amd64 || arm64))

package node

import (
	"fmt"
	"github.com/spf13/cobra"
	"jianggujin.com/lvs/internal/config"
	"jianggujin.com/lvs/internal/util"
	"os"
	"path/filepath"
)

func init() {
	nodeCmd.AddCommand(&UninstallCommand{})
}

type UninstallCommand struct {
}

func (command *UninstallCommand) Init() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall the installed version of node.js",
		RunE:  command.RunE,
	}
	return cmd
}

func (command *UninstallCommand) RunE(_ *cobra.Command, versions []string) error {
	installHome := config.GetPath(config.KeyNodeHome)
	for i, version := range versions {
		if i > 0 {
			fmt.Println()
		}

		fmt.Printf("uninstall %s start\n", version)
		version = nodeCmd.FixVersion(version)
		if _, err := nodeCmd.Semver(version); err != nil {
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
