package gom

import (
	"github.com/hashicorp/go-version"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"jianggujin.com/lvs/internal/config"
	"jianggujin.com/lvs/internal/util"
	"os"
	"sort"
	"time"
)

func init() {
	goCmd.AddCommand(&ListCommand{})
}

type ListCommand struct {
	All bool
}

func (command *ListCommand) Init() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List all available versions of Go",
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
	// go1.20.5
	current := goCmd.Version()

	goHome := config.GetPath(config.KeyGoHome)
	entries, err := os.ReadDir(goHome)
	if err != nil {
		if !os.IsNotExist(err) {
			return util.WrapErrorMsg("list local installed version error").SetErr(err)
		}
	}
	var versions []*version.Version
	installed := make(map[string]time.Time)
	if len(entries) > 0 {
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			if ver, _ := goCmd.Semver(entry.Name()); ver != nil {
				versions = append(versions, ver)
				if info, _ := entry.Info(); info != nil {
					if modTime := info.ModTime(); !modTime.IsZero() {
						installed[entry.Name()] = modTime
					}
				}
			}
		}
	}
	if !command.All {
		sort.Sort(version.Collection(versions))
		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"", "Version", "Time"})
		table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
		table.SetAlignment(tablewriter.ALIGN_CENTER)
		table.SetCenterSeparator("|")
		for i := len(versions) - 1; i >= 0; i-- {
			ver := versions[i]
			if constraints != nil && !constraints.Check(ver) {
				continue
			}
			row := []string{"", goCmd.RawVersion(ver), installed[goCmd.RawVersion(ver)].Format(time.DateTime)}
			if row[1] == current {
				row[0] = " * "
			}
			table.Append(row)
		}
		table.Render()
		return nil
	}

	if command.All {
		list, err := goCmd.ListVersions(func(goVersion *Version) (bool, error) {
			if constraints != nil {
				v, _ := goVersion.Semver()
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
		table.SetHeader([]string{"", "Version", "Size"})
		table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
		table.SetAlignment(tablewriter.ALIGN_CENTER)
		table.SetCenterSeparator("|")
		for _, ver := range list {
			row := []string{"", ver.Version, ver.Size}
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
