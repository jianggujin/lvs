//go:build (windows && (amd64 || 386 || arm64)) || (linux && (amd64 || arm || armv7l || arm64 || ppc64le || s390x)) || (darwin && (amd64 || arm64))

package node

import (
	"github.com/hashicorp/go-version"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"
	"jianggujin.com/lvs/internal/config"
	"jianggujin.com/lvs/internal/util"
	"os"
	"sort"
)

func init() {
	nodeCmd.AddCommand(&ListCommand{})
}

type ListCommand struct {
	All bool
}

func (command *ListCommand) Init() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List all available versions of node.js",
		Aliases: []string{"ls"},
		RunE:    command.RunE,
	}
	flags := cmd.Flags()
	flags.BoolVarP(&command.All, "all", "a", false, "list all available versions")
	return cmd
}

func (command *ListCommand) RunE(_ *cobra.Command, consts []string) error {
	var constraints version.Constraints

	if len(consts) > 0 {
		var err error
		if constraints, err = version.NewConstraint(consts[0]); err != nil {
			return util.WrapErrorMsg("parse version constraint error").SetErr(err)
		}
	}
	current := nodeCmd.Version()

	nodeHome := config.GetPath(config.KeyNodeHome)
	entries, err := os.ReadDir(nodeHome)
	if err != nil {
		if !os.IsNotExist(err) {
			return util.WrapErrorMsg("list local installed version error").SetErr(err)
		}
	}
	var versions []*version.Version
	installed := make(map[string]bool)
	if len(entries) > 0 {
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			ver, _ := nodeCmd.Semver(entry.Name())
			if ver != nil {
				versions = append(versions, ver)
				installed[entry.Name()] = true
			}
		}
	}
	if !command.All {
		sort.Sort(version.Collection(versions))
		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"", "Version"})
		table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
		table.SetAlignment(tablewriter.ALIGN_CENTER)
		table.SetCenterSeparator("|")

		for i := len(versions) - 1; i >= 0; i-- {
			ver := versions[i]
			if constraints != nil && !constraints.Check(ver) {
				continue
			}
			row := []string{"", nodeCmd.RawVersion(ver)}
			if row[1] == current {
				row[0] = " * "
			}
			table.Append(row)
		}
		table.Render()
		return nil
	}

	if command.All {
		list, err := nodeCmd.ListVersions(func(nodeVersion *Version) (bool, error) {
			if constraints != nil {
				v, _ := nodeVersion.Semver()
				if v != nil && !constraints.Check(v) {
					return false, nil
				}
			}
			return true, nil
		})
		if err != nil {
			return util.WrapErrorMsg("list all available versions error").SetErr(err)
		}
		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"", "Version", "Npm", "Lts", "Security", "Date"})
		table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
		table.SetAlignment(tablewriter.ALIGN_CENTER)
		table.SetCenterSeparator("|")
		for _, ver := range list {
			row := []string{"", ver.Version, ver.Npm, cast.ToString(ver.Lts), cast.ToString(ver.Security), ver.Date}
			if ver.Version == current {
				row[0] = " * "
			} else if _, ok := installed[ver.Version]; ok {
				row[0] = " + "
			}
			table.Append(row)
		}
		table.Render()
	}
	return nil
}
