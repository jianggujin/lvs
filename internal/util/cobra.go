package util

import (
	"fmt"
	"github.com/spf13/cobra"
)

type Command interface {
	Init() *cobra.Command
}

func AddCommand(parent *cobra.Command, command Command) {
	cmd := command.Init()
	if cmd.RunE == nil {
		if cmd.Run != nil {
			cmd.RunE = func(self *cobra.Command, args []string) (err error) {
				defer func() {
					if r := recover(); r != nil {
						returnErr, ok := r.(error)
						if !ok {
							returnErr = fmt.Errorf("%v", r)
						}
						err = returnErr
					}
				}()
				cmd.Run(self, args)
				return
			}
		}
	}
	parent.AddCommand(cmd)
}
