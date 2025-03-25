//go:build windows

package main

import (
	"jianggujin.com/lvs/internal/config"
)

func (command *ConfigCommand) initConfigKeys() {
	command.configKeys[config.KeyLvsScriptHome] = &ConfigValidator{
		Setter: command.setDirConfig,
	}
}
