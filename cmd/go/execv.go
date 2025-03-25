package gom

import (
	"fmt"
	"github.com/spf13/cobra"
	"jianggujin.com/lvs/internal/config"
	"jianggujin.com/lvs/internal/invoke"
	"jianggujin.com/lvs/internal/util"
	"path/filepath"
)

func init() {
	goCmd.AddCommand(&ExecvCommand{})
}

type ExecvCommand struct {
}

func (command *ExecvCommand) Init() *cobra.Command {
	cmd := &cobra.Command{
		Use:                "execv",
		Short:              "Execute commands using the specified version",
		DisableFlagParsing: true,
		RunE:               command.RunE,
	}
	return cmd
}

func (command *ExecvCommand) RunE(_ *cobra.Command, args []string) error {
	if len(args) < 2 {
		fmt.Printf("Usage: %s go execv x.x.x commands...\n", config.Name())
		return nil
	}
	installHome := config.GetPath(config.KeyGoHome)
	version := goCmd.FixVersion(config.GetStringWithDefault(config.KeyGoAliasPrefix+args[0], args[0]))
	if _, err := goCmd.Semver(version); err != nil {
		return util.WrapErrorMsg("[%s] is not a valid version", version)
	}
	dir := filepath.Join(installHome, version)
	if !util.Exists(dir) {
		return util.WrapErrorMsg("[%s] not found", version)
	}
	var arg []string
	if len(args) > 2 {
		arg = args[2:]
	}
	execPath := filepath.Join(dir, args[1])

	if util.ExecExists(execPath) {
		return invoke.GetInvoker().CommandOptions(execPath, arg, invoke.WithStd())
	}

	return invoke.GetInvoker().CommandOptions(args[1], arg, invoke.WithStd())
}
