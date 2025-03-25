package main

import (
	"errors"
	"fmt"
	"github.com/mitchellh/go-homedir"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"jianggujin.com/lvs/internal/config"
	"jianggujin.com/lvs/internal/install"
	"jianggujin.com/lvs/internal/util"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func init() {
	util.AddCommand(rootCmd, &ConfigCommand{})
}

type ConfigValidator struct {
	Getter func(string) (string, error)
	Setter func(string, string) error
}

type ConfigCommand struct {
	configKeys map[string]*ConfigValidator
}

func (command *ConfigCommand) Init() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "config",
		Short:  "Set or read LVS configuration",
		Long:   "Set or read LVS configuration. When only the configuration key is specified, it is read configuration; otherwise, it is set configuration",
		PreRun: command.preRun,
		RunE:   command.RunE,
	}
	return cmd
}

func (command *ConfigCommand) preRun(_ *cobra.Command, _ []string) {
	command.configKeys = map[string]*ConfigValidator{
		config.KeyLvsDataHome:       {Setter: command.setEnvDirConfig},
		config.KeyLvsTempHome:       {Setter: command.setDirConfig},
		config.KeyLvsProxy:          {Setter: command.setProxyConfig},
		config.KeyLvsDefaultCommand: {Setter: command.setConfig},

		config.KeyGoHome:    {Setter: command.setDirConfig},
		config.KeyGoSymlink: {Setter: command.setSymlinkConfig},
		config.KeyGoProxy:   {Setter: command.setProxyConfig},
		config.KeyGoMirror:  {Setter: command.setMirrorConfig},
	}
	if config.HasNode {
		command.configKeys[config.KeyNodeHome] = &ConfigValidator{Setter: command.setDirConfig}
		command.configKeys[config.KeyNodeSymlink] = &ConfigValidator{Setter: command.setSymlinkConfig}
		command.configKeys[config.KeyNodeProxy] = &ConfigValidator{Setter: command.setProxyConfig}
		command.configKeys[config.KeyNodeMirror] = &ConfigValidator{Setter: command.setMirrorConfig}
	}
	command.initConfigKeys()
}

func (command *ConfigCommand) RunE(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return command.listConfigKeys()
	}
	key := args[0]
	// 匹配只包含字母、数字和下划线的字符串
	//re := regexp.MustCompile("^[a-zA-Z0-9_]+$")
	//if !re.MatchString(key) {
	validator, ok := command.configKeys[key]
	if !ok {
		return util.WrapErrorMsg("configuration [%s] does not exist", key)
	}
	if len(args) == 1 {
		getter := command.getConfig
		if validator != nil && validator.Getter != nil {
			getter = validator.Getter
		}
		value, err := getter(key)
		if err != nil {
			return util.WrapErrorMsg("failed to obtain configuration [%s]", key).SetErr(err)
		}
		fmt.Printf("%s: %s\n", key, value)
		return nil
	}
	if validator == nil || validator.Setter == nil {
		return util.WrapErrorMsg("configuration [%s] is read-only and does not allow writing", key)
	}
	value := args[1]
	if err := validator.Setter(key, value); err != nil {
		return util.WrapErrorMsg("failed to write value [%s] for configuration [%s]", key, value).SetErr(err)
	}
	if err := config.SaveConfig(); err != nil {
		return util.WrapErrorMsg("failed to save configuration [%s: %s]", key, value).SetErr(err)
	}
	fmt.Printf("%s: %s\n", key, config.GetString(key))
	return nil
}

func (command *ConfigCommand) listConfigKeys() error {
	var configKeys []string
	for k := range command.configKeys {
		configKeys = append(configKeys, k)
	}
	sort.Strings(configKeys)
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "Value", "Mode"})
	table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	table.SetAlignment(tablewriter.ALIGN_CENTER)
	table.SetCenterSeparator("|")
	for _, key := range configKeys {
		validator := command.configKeys[key]
		readonly := validator == nil || validator.Setter == nil
		getter := command.getConfig
		if validator != nil && validator.Getter != nil {
			getter = validator.Getter
		}
		value, err := getter(key)
		if err != nil {
			return util.WrapErrorMsg("failed to obtain configuration [%s]", key).SetErr(err)
		}
		if readonly {
			table.Append([]string{key, value, "read"})
		} else {
			table.Append([]string{key, value, "read/write"})
		}
	}
	table.Render()
	return nil
}

func (command *ConfigCommand) setSymlinkConfig(name, value string) error {
	if value != "none" && value != "" {
		return errors.New("setting an empty path is not allowed")
	}
	newLinkPath := filepath.Clean(value)
	var err error
	newLinkPath, err = homedir.Expand(newLinkPath)
	if err != nil {
		return err
	}
	// 读取现在的地址
	oldLinkPath := config.GetPath(name)
	// 无变化，直接返回
	if newLinkPath == oldLinkPath {
		return nil
	}
	newIsSymlink, newExist, err := util.IsSymlink(newLinkPath)
	if err != nil {
		return err
	}
	if newExist && !newIsSymlink {
		return fmt.Errorf("the path [%s] already exists but is not a valid symlink", newLinkPath)
	}

	module := config.Modules[strings.SplitN(strings.ToLower(name), "_", 2)[0]]

	if module != nil {
		if err = install.Install(map[string]string{
			module.SymlinkEnvKey: newLinkPath,
		}, nil); err != nil {
			return err
		}
	} else {
		config.Set(name, newLinkPath)
	}

	if newExist {
		if err = os.Remove(newLinkPath); err != nil {
			return err
		}
	}

	targetPath, err := util.ReadSymlink(oldLinkPath)
	if err != nil {
		return err
	}

	if targetPath != "" {
		if err = os.Remove(oldLinkPath); err != nil {
			return err
		}
		// 目标路径是文件夹则重新设置软链接
		if info, err := os.Stat(targetPath); err != nil && info.IsDir() {
			return util.SymlinkDir(newLinkPath, targetPath)
		}
	}
	return nil
}

func (command *ConfigCommand) setProxyConfig(name, value string) error {
	if value != "none" && value != "" {
		if _, err := url.Parse(value); err != nil {
			return err
		}
	}
	return command.setConfig(name, value)
}

func (command *ConfigCommand) setMirrorConfig(name, value string) error {
	if value != "none" && value != "" {
		if !strings.HasPrefix(value, "http://") && !strings.HasPrefix(value, "https://") {
			return fmt.Errorf("the mirror address [%s] protocol is illegal, only http or https is allowed", value)
		}
		if !strings.HasSuffix(value, "/") {
			value = value + "/"
		}
		if _, err := url.Parse(value); err != nil {
			return err
		}
	}
	return command.setConfig(name, value)
}

func (command *ConfigCommand) setEnvDirConfig(name, value string) error {
	return command.checkDirConfig(value, func(s string) error {
		return install.Install(map[string]string{
			config.EnvLvsPrefix + name: s,
		}, nil)
	})
}

func (command *ConfigCommand) setDirConfig(name, value string) error {
	return command.checkDirConfig(value, func(s string) error {
		return command.setConfig(name, s)
	})
}

func (command *ConfigCommand) checkDirConfig(value string, consumer func(string) error) error {
	value = filepath.Clean(value)
	var err error
	value, err = homedir.Expand(value)
	if err != nil {
		return err
	}
	info, err := os.Stat(value)
	if err != nil {
		if os.IsNotExist(err) {
			return consumer(value)
		}
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("[%s] is not a directory", value)
	}
	return consumer(value)
}

func (command *ConfigCommand) setFileConfig(name, value string) error {
	value = filepath.Clean(value)
	var err error
	value, err = homedir.Expand(value)
	if err != nil {
		return err
	}
	info, err := os.Stat(value)
	if err != nil {
		if os.IsNotExist(err) {
			return command.setConfig(name, value)
		}
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("[%s] is not a regular file", value)
	}
	return command.setConfig(name, value)
}

func (command *ConfigCommand) setBooleanConfig(name, value string) error {
	value = strings.ToLower(value)
	if "true" == value || "1" == value || "y" == value {
		value = "true"
	} else {
		value = "false"
	}
	return command.setConfig(name, value)
}

func (command *ConfigCommand) getConfig(name string) (string, error) {
	return config.GetString(name), nil
}

func (command *ConfigCommand) setConfig(name, value string) error {
	if "none" == value {
		value = ""
	}
	config.Set(name, value)
	return nil
}
