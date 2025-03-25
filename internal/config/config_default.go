//go:build !windows

package config

import (
	"fmt"
	"github.com/spf13/viper"
	"jianggujin.com/lvs/internal/shell"
	"jianggujin.com/lvs/internal/util"
	"os"
)

const (
	KeyShellType = "SHELL_TYPE" // Shell类型

	KeyLvsBackupHome = "BACKUP_HOME" // 备份目录
)

const (
	defaultLvsBackupHome = defaultLvsDataHome + "/backup"
)

func initDefault() {
	viper.SetDefault(KeyLvsBackupHome, env(KeyLvsBackupHome, defaultLvsBackupHome, false))
}

func afterInit() {
	changed := false
	shellType := GetString(KeyShellType)
	if shellType == "" {
		shellType = shell.ShellType()
		Set(KeyShellType, shellType)
		changed = true
	}
	shellConfigPath := GetString(KeyShellConfigPath)
	if shellConfigPath == "" {
		shellConfigPath = shell.ShellConfigPath(shellType)
		Set(KeyShellConfigPath, shellConfigPath)
		changed = true
	}
	if changed {
		if err := util.Sudo(SaveConfig()); err != nil {
			fmt.Println(util.WrapErrorMsg("failed to save configuration").SetErr(err).Error())
			os.Exit(1)
		}
	}
}
