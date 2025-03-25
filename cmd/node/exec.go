//go:build (windows && (amd64 || 386 || arm64)) || (linux && (amd64 || arm || armv7l || arm64 || ppc64le || s390x)) || (darwin && (amd64 || arm64))

package node

import (
	"fmt"
	"github.com/spf13/cobra"
	"jianggujin.com/lvs/internal/config"
	"jianggujin.com/lvs/internal/invoke"
	"jianggujin.com/lvs/internal/util"
	"path/filepath"
)

func init() {
	nodeCmd.AddCommand(&ExecCommand{})
}

type ExecCommand struct {
}

func (command *ExecCommand) Init() *cobra.Command {
	cmd := &cobra.Command{
		Use:                "exec",
		Short:              "Execute commands using the workspace version",
		DisableFlagParsing: true,
		RunE:               command.RunE,
	}
	return cmd
}

func (command *ExecCommand) RunE(_ *cobra.Command, args []string) error {
	version, err := config.GetWorkspaceUseVersion(config.ModuleNode)
	if err != nil {
		return util.WrapError(err)
	}
	if version == "" {
		return util.WrapErrorMsg("valid version not found from workspace")
	}
	if len(args) < 1 {
		fmt.Printf("Usage: %s node exec commands...\n", config.Name())
		return nil
	}
	installHome := config.GetPath(config.KeyNodeHome)
	version = nodeCmd.FixVersion(config.GetStringWithDefault(config.KeyNodeAliasPrefix+version, version))
	if _, err := nodeCmd.Semver(version); err != nil {
		return util.WrapErrorMsg("[%s] is not a valid version", version)
	}
	dir := filepath.Join(installHome, version)
	if !util.Exists(dir) {
		return util.WrapErrorMsg("[%s] not found", version)
	}
	var arg []string
	if len(args) > 1 {
		arg = args[1:]
	}
	execPath := filepath.Join(dir, args[0])

	if util.ExecExists(execPath) {
		return invoke.GetInvoker().CommandOptions(execPath, arg, invoke.WithStd())
	}

	return invoke.GetInvoker().CommandOptions(args[0], arg, invoke.WithStd())
}
