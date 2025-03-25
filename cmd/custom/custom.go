package custom

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
	"io"
	"jianggujin.com/lvs/internal/config"
	"jianggujin.com/lvs/internal/install"
	"jianggujin.com/lvs/internal/invoke"
	"jianggujin.com/lvs/internal/util"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

type Custom struct {
	Name          string            `json:"name"`
	Home          string            `json:"home"`
	SymlinkEnvKey string            `json:"symlinkEnvKey"`
	SymlinkPath   string            `json:"symlinkPath"`
	EnvKeyValues  map[string]string `json:"envKeyValues"`
	PathValues    []string          `json:"pathValues"`
	Version       *Version          `json:"version"`
}

type Version struct {
	Cmd    []string `json:"cmd"`
	Regexp string   `json:"regexp"`
	Group  int      `json:"group"`
}

func (c *Custom) CanInstall() bool {
	if (c.EnvKeyValues == nil || len(c.EnvKeyValues) == 0) && len(c.PathValues) == 0 {
		return false
	}
	return true
}

func Init(rootCmd *cobra.Command) {
	reader, err := os.Open(filepath.Join(config.GetPath(config.KeyLvsDataHome), config.DefaultLvsCustomFile))
	if err != nil {
		return
	}
	defer reader.Close()
	data, err := io.ReadAll(reader)
	var customs []*Custom
	if err = json.Unmarshal(data, &customs); err != nil {
		return
	}

	injects := []func(*cobra.Command, *Custom){injectCurrent, injectInstall, injectList, injectUninstall, injectUse}
	// 注册自定义命令
	for _, custom := range customs {
		name := strings.TrimSpace(custom.Name)
		if name == "" {
			continue
		}
		if custom.SymlinkPath == "" && custom.SymlinkEnvKey != "" && custom.EnvKeyValues != nil {
			custom.SymlinkPath = custom.EnvKeyValues[custom.SymlinkEnvKey]
		}
		if custom.SymlinkEnvKey != "" && (custom.EnvKeyValues == nil || custom.EnvKeyValues[custom.SymlinkEnvKey] == "") && custom.SymlinkPath != "" {
			if custom.EnvKeyValues == nil {
				custom.EnvKeyValues = map[string]string{custom.SymlinkEnvKey: custom.SymlinkPath}
			} else {
				custom.EnvKeyValues[custom.SymlinkEnvKey] = custom.SymlinkPath
			}
		}
		if _, ok := config.Modules[name]; ok {
			continue
		}
		command := &cobra.Command{
			Use:   name,
			Short: name + " version management",
		}
		rootCmd.AddCommand(command)
		for _, inject := range injects {
			inject(command, custom)
		}
	}
}

func Current(version *Version) string {
	if version == nil {
		return ""
	}
	var data []byte
	var err error
	if len(version.Cmd) > 1 {
		data, err = invoke.GetInvoker().Command(version.Cmd[0], version.Cmd[1:]...)
	} else {
		data, err = invoke.GetInvoker().Command(version.Cmd[0])
	}
	if err != nil {
		return ""
	}
	if runtime.GOOS == "windows" {
		reader := transform.NewReader(bytes.NewReader(data), simplifiedchinese.GBK.NewDecoder())
		d, e := io.ReadAll(reader)
		if e == nil {
			data = d
		}
	}
	if version.Regexp == "" {
		return string(data)
	}

	re, err := regexp.Compile(version.Regexp)
	if err != nil {
		return string(data)
	}
	matches := re.FindStringSubmatch(string(data))
	if len(matches) == 0 {
		return string(data)
	}
	if version.Group < len(matches) {
		return matches[version.Group]
	}
	return string(data)
}

func injectCurrent(rootCmd *cobra.Command, custom *Custom) {
	version := custom.Version
	if version == nil || len(version.Cmd) == 0 {
		return
	}
	cmd := &cobra.Command{
		Use:     "current",
		Short:   "Display the current version being used",
		Aliases: []string{"v"},
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(Current(version))
		},
	}
	rootCmd.AddCommand(cmd)
}

func injectInstall(rootCmd *cobra.Command, custom *Custom) {
	if !custom.CanInstall() {
		return
	}
	cmd := &cobra.Command{
		Use:     "install",
		Short:   fmt.Sprintf("Install the specified %s version", custom.Name),
		Aliases: []string{"i"},
		RunE: func(cmd *cobra.Command, args []string) error {
			envKeyValues := make(map[string]string)
			var pathValues []string

			keyValues := custom.EnvKeyValues
			if keyValues != nil {
				for k, v := range keyValues {
					envKeyValues[k] = v
				}
			}
			values := custom.PathValues
			if len(values) > 0 {
				pathValues = append(pathValues, values...)
			}
			if err := install.Install(envKeyValues, pathValues); err != nil {
				return util.WrapErrorMsg("installation failed, please try again").SetErr(err)
			}
			if runtime.GOOS == "windows" {
				fmt.Println("installation completed, if unable to use normally, please try restarting the terminal")
			} else {
				fmt.Printf("installation completed, if unable to use normally, please try restarting the terminal or run 'source %s'\n",
					config.GetString(config.KeyShellConfigPath))
			}
			return nil
		},
	}
	rootCmd.AddCommand(cmd)
}

func injectList(rootCmd *cobra.Command, custom *Custom) {
	home := custom.Home
	if home == "" {
		return
	}
	if !util.Exists(home) {
		return
	}
	cmd := &cobra.Command{
		Use:     "list",
		Short:   fmt.Sprintf("List all available versions of %s", custom.Name),
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			current := Current(custom.Version)
			entries, err := os.ReadDir(home)
			if err != nil {
				if !os.IsNotExist(err) {
					return util.WrapErrorMsg("list local installed version error").SetErr(err)
				}
			}
			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"", "Version", "Time"})
			table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
			table.SetAlignment(tablewriter.ALIGN_CENTER)
			table.SetCenterSeparator("|")
			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}
				installTime := ""
				if info, _ := entry.Info(); info != nil {
					if modTime := info.ModTime(); !modTime.IsZero() {
						installTime = modTime.Format(time.DateTime)
					}
				}

				row := []string{"", entry.Name(), installTime}
				if row[0] == current {
					row[0] = " * "
				}
				table.Append(row)
			}
			return nil
		},
	}
	rootCmd.AddCommand(cmd)
}

func injectUninstall(rootCmd *cobra.Command, custom *Custom) {
	if !custom.CanInstall() {
		return
	}
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: fmt.Sprintf("Uninstall the specified %s version", custom.Name),
		RunE: func(cmd *cobra.Command, args []string) error {
			var envKeys []string
			var pathValues []string

			keyValues := custom.EnvKeyValues
			if keyValues != nil {
				for k, _ := range keyValues {
					envKeys = append(envKeys, k)
				}
			}
			values := custom.PathValues
			if len(values) > 0 {
				pathValues = append(pathValues, values...)
			}
			if err := install.Uninstall(envKeys, pathValues); err != nil {
				return util.WrapErrorMsg("uninstalling failed, please try again").SetErr(err)
			}
			if custom.SymlinkPath != "" {
				_ = os.Remove(custom.SymlinkPath)
			}
			fmt.Println("uninstall complete")
			return nil
		},
	}
	rootCmd.AddCommand(cmd)
}

func injectUse(rootCmd *cobra.Command, custom *Custom) {
	if custom.SymlinkPath == "" || custom.SymlinkEnvKey == "" {
		return
	}
	home := custom.Home
	if home == "" {
		return
	}
	if !util.Exists(home) {
		return
	}

	cmd := &cobra.Command{
		Use:     "use",
		Short:   fmt.Sprintf("Activate the specified version of %s", custom.Name),
		Aliases: []string{"u"},
		RunE: func(cmd *cobra.Command, versions []string) error {
			if len(versions) != 1 {
				fmt.Printf("Usage: %s %s use x.x.x\n", config.Name(), custom.Name)
				return nil
			}
			version := versions[0]
			if version == Current(custom.Version) {
				fmt.Printf("[%s] has been activated\n", version)
				return nil
			}
			dir := filepath.Join(home, version)
			if !util.Exists(dir) {
				return util.WrapErrorMsg("[%s] not found\n", version)
			}
			if err := util.ResetSymlink(config.GetPath(custom.SymlinkPath), dir, true); err != nil {
				return util.WrapErrorMsg("reset symlink error").SetErr(err)
			}

			var installErr error
			// 需要安装
			if custom.CanInstall() && os.Getenv(custom.SymlinkEnvKey) != config.GetPath(custom.SymlinkPath) {
				envKeyValues := make(map[string]string)
				var pathValues []string

				keyValues := custom.EnvKeyValues
				if keyValues != nil {
					for k, v := range keyValues {
						envKeyValues[k] = v
					}
				}
				values := custom.PathValues
				if len(values) > 0 {
					pathValues = append(pathValues, values...)
				}
				installErr = install.Install(envKeyValues, pathValues)
			}
			if installErr != nil {
				return util.WrapErrorMsg("[%s] has been activated. but installation failed", version).SetErr(installErr)
			} else {
				if runtime.GOOS == "windows" {
					fmt.Printf("[%s] has been activated, if unable to use normally, please try restarting the terminal\n", version)
				} else {
					fmt.Printf("[%s] has been activated, if unable to use normally, please try restarting the terminal or run 'source %s'\n",
						version, config.GetString(config.KeyShellConfigPath))
				}
			}
			return nil
		},
	}
	rootCmd.AddCommand(cmd)
}
