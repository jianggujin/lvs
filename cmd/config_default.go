//go:build !windows

package main

import "jianggujin.com/lvs/internal/config"

func (command *ConfigCommand) initConfigKeys() {
	command.configKeys[config.KeyShellType] = &ConfigValidator{
		Setter: command.setConfig,
	}
	command.configKeys[config.KeyShellConfigPath] = &ConfigValidator{
		Setter: command.setFileConfig,
	}
	command.configKeys[config.KeyLvsBackupHome] = &ConfigValidator{
		Setter: command.setDirConfig,
	}
}
