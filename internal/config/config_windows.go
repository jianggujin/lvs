//go:build windows

package config

import (
	"github.com/spf13/viper"
)

const (
	KeyLvsScriptHome = "SCRIPT_HOME" // 程序脚本目录
)

const (
	defaultLvsScriptHome = defaultLvsDataHome + "/script"
)

func initDefault() {
	viper.SetDefault(KeyLvsScriptHome, env(KeyLvsScriptHome, defaultLvsScriptHome, false))
}

func afterInit() {

}
