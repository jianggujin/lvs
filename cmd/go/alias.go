package gom

import (
	"fmt"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"jianggujin.com/lvs/internal/config"
	"jianggujin.com/lvs/internal/util"
	"os"
	"sort"
	"strings"
)

func init() {
	goCmd.AddCommand(&AliasCommand{})
}

type AliasCommand struct {
}

func (command *AliasCommand) Init() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "alias",
		Short:   "Set an alias for the specified version",
		Aliases: []string{"tag"},
		RunE:    command.RunE,
	}
	return cmd
}

func (command *AliasCommand) RunE(_ *cobra.Command, args []string) error {
	if len(args) == 2 {
		version := goCmd.FixVersion(args[1])
		if _, err := goCmd.Semver(version); err != nil {
			fmt.Printf("[%s] is not a valid version\n", version)
			return nil
		}
		name := strings.ToLower(args[0])
		config.Set(config.KeyGoAliasPrefix+name, version)
		if err := config.SaveConfig(); err != nil {
			return util.WrapErrorMsg("failed to save alias [%s: %s]", name, version).SetErr(err)
		}
		fmt.Printf("%s: %s\n", name, version)
		return nil
	}
	if len(args) == 1 {
		name := strings.ToLower(args[0])
		version := config.GetString(config.KeyGoAliasPrefix + name)
		fmt.Printf("%s: %s\n", name, version)
		return nil
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Alias", "Version"})
	table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	table.SetAlignment(tablewriter.ALIGN_CENTER)
	table.SetCenterSeparator("|")

	var keys []string
	lowerPrefix := strings.ToLower(config.KeyGoAliasPrefix)
	m := config.Filter(func(s string) bool {
		if after, ok := strings.CutPrefix(s, lowerPrefix); ok {
			keys = append(keys, after)
			return true
		}
		return false
	})

	sort.Strings(keys)

	for _, key := range keys {
		table.Append([]string{key, m[lowerPrefix+key]})
	}

	table.Render()
	return nil
}
