package gom

import (
	"fmt"
	"github.com/spf13/cobra"
	"jianggujin.com/lvs/internal/config"
)

func init() {
	goCmd.AddCommand(&CurrentCommand{})
}

type CurrentCommand struct {
}

func (command *CurrentCommand) Init() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "current",
		Short:   "Display the current version being used",
		Aliases: []string{"v"},
		Run:     command.Run,
	}
	return cmd
}

func (command *CurrentCommand) Run(*cobra.Command, []string) {
	ver := goCmd.Version()

	if ver == "" {
		fmt.Printf("there is currently no version in use. You can run '%s go use x.x.x' to set a version\n", config.Name())
		return
	}
	fmt.Println(ver)
}
