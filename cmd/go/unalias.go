package gom

import (
	"fmt"
	"github.com/spf13/cobra"
	"jianggujin.com/lvs/internal/config"
	"jianggujin.com/lvs/internal/util"
	"strings"
)

func init() {
	goCmd.AddCommand(&UnAliasCommand{})
}

type UnAliasCommand struct {
}

func (command *UnAliasCommand) Init() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unalias",
		Short: "Cancel the alias of the specified version that has already been set",
		RunE:  command.RunE,
	}
	return cmd
}

func (command *UnAliasCommand) RunE(_ *cobra.Command, args []string) error {
	if len(args) == 0 {
		return nil
	}
	for _, name := range args {
		name = strings.ToLower(name)
		config.Set(config.KeyGoAliasPrefix+name, "")
		fmt.Printf("unalias: %s\n", name)
	}
	if err := config.SaveConfig(); err != nil {
		return util.WrapErrorMsg("failed to save configuration").SetErr(err)
	}
	return nil
}
