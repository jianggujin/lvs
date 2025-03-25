//go:build (windows && (amd64 || 386 || arm64)) || (linux && (amd64 || arm || armv7l || arm64 || ppc64le || s390x)) || (darwin && (amd64 || arm64))

package node

import (
	"fmt"
	"github.com/spf13/cobra"
	"jianggujin.com/lvs/internal/config"
)

func init() {
	nodeCmd.AddCommand(&CurrentCommand{})
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
	ver := nodeCmd.Version()

	if ver == "" {
		fmt.Printf("there is currently no version in use. You can run '%s node use x.x.x' to set a version", config.Name())
		return
	}
	fmt.Println(ver)
}
