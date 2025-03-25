package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"jianggujin.com/lvs/internal/config"
	"jianggujin.com/lvs/internal/util"
)

type VersionCommand struct {
	Short bool
}

func init() {
	util.AddCommand(rootCmd, &VersionCommand{})
}

func (command *VersionCommand) Init() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "version",
		Short:   "Display the version of LVS currently running",
		Aliases: []string{"v"},
		Run:     command.Run,
	}
	flags := cmd.Flags()
	flags.BoolVarP(&command.Short, "short", "s", false, "only display version number")
	return cmd
}

func (command *VersionCommand) Run(*cobra.Command, []string) {
	if command.Short {
		fmt.Println(config.BuildVersion)
		return
	}
	fmt.Printf("LVS(Lightweight Version Suite) Version %s, BuildTime: %s\n", config.BuildVersion, config.BuildTime)
}
