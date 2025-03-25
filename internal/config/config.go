package config

import (
	"fmt"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	HasNode = (runtime.GOOS == "windows" && (runtime.GOARCH == "amd64" || runtime.GOARCH == "386" || runtime.GOARCH == "arm64")) ||
		(runtime.GOOS == "linux" && (runtime.GOARCH == "amd64" || runtime.GOARCH == "arm" || runtime.GOARCH == "armv7l" || runtime.GOARCH == "arm64" || runtime.GOARCH == "ppc64le" || runtime.GOARCH == "s390x")) ||
		(runtime.GOOS == "darwin" && (runtime.GOARCH == "amd64" || runtime.GOARCH == "arm64"))
)

const (
	KeyLvsDataHome       = "DATA_HOME"       // 程序数据目录
	KeyLvsProxy          = "PROXY"           // 全局代理配置
	KeyLvsTempHome       = "TEMP_HOME"       // 临时文件目录
	KeyLvsDefaultCommand = "DEFAULT_COMMAND" // 默认执行命令

	KeyShellConfigPath = "SHELL_CONFIG_PATH" // Shell配置文件 非windows生效

	KeyNodeSymlink = "NODE_SYMLINK"     // node.js软链的文件位置
	KeyNodeHome    = "NODE_HOME"        // node.js程序安装目录
	KeyNodeProxy   = "NODE_PROXY"       // node.js代理配置
	KeyNodeMirror  = "NODE_NODE_MIRROR" // node.js镜像地址

	KeyGoSymlink = "GO_SYMLINK" // go软链的文件位置
	KeyGoHome    = "GO_HOME"    // go程序安装目录
	KeyGoProxy   = "GO_PROXY"   // go代理配置
	KeyGoMirror  = "GO_MIRROR"  // go镜像地址

	KeyNodeAliasPrefix = "ALIAS_NODE_" // node.js版本别名
	KeyGoAliasPrefix   = "ALIAS_GO_"   // go版本别名
	KeyWorkspaceSuffix = ".lvsrc"      // 工作空间使用版本后缀
)

type Module struct {
	Name          string
	SymlinkEnvKey string
	EnvKeyValues  map[string]string
	PathValues    []string
}

var Modules = make(map[string]*Module)

const (
	ModuleNode = "node"
	ModuleGo   = "go"
)

const (
	EnvLvsPrefix = "LVS_"                // 程序目录
	EnvLvsHome   = EnvLvsPrefix + "HOME" // 程序目录
	EnvNodeHome  = "NODE_HOME"           // 指向node.js软链
	EnvGoRoot    = "GOROOT"
	EnvGoPath    = "GOPATH"
)

const (
	defaultLvsConfigFile = ".lvsrc"
	defaultLvsConfigType = "yaml"
	defaultLvsDataHome   = "~/.lvs"
	defaultLvsTempHome   = defaultLvsDataHome + "/temp"
	DefaultLvsCustomFile = "custom.json"

	defaultNodeHome       = defaultLvsDataHome + "/repository/nodejs"
	defaultNodeSymlink    = defaultLvsDataHome + "/symlink/nodejs"
	defaultNodeNodeMirror = "https://nodejs.org/dist/"
	//defaultNodeNpmMirror  = "https://github.com/npm/cli/archive/"

	defaultGoHome    = defaultLvsDataHome + "/repository/go"
	defaultGoSymlink = defaultLvsDataHome + "/symlink/go"
	defaultGoPath    = "~/go"
	// https://go.dev/dl/
	defaultGoMirror = "https://golang.google.cn/dl/"
)

func init() {
	viper.SetDefault(KeyLvsDataHome, env(KeyLvsDataHome, defaultLvsDataHome, false))
	viper.SetDefault(KeyLvsTempHome, env(KeyLvsTempHome, defaultLvsTempHome, false))
	viper.SetDefault(KeyLvsDefaultCommand, env(KeyLvsDefaultCommand, "", false))

	viper.SetDefault(KeyNodeHome, env(KeyNodeHome, defaultNodeHome, false))
	viper.SetDefault(KeyNodeSymlink, env(KeyNodeSymlink, defaultNodeSymlink, false))
	viper.SetDefault(KeyNodeMirror, env(KeyNodeMirror, defaultNodeNodeMirror, false))

	viper.SetDefault(KeyGoHome, env(KeyGoHome, defaultGoHome, false))
	viper.SetDefault(KeyGoSymlink, env(KeyGoSymlink, defaultGoSymlink, false))
	viper.SetDefault(KeyGoMirror, env(KeyGoMirror, defaultGoMirror, false))

	initDefault()

	viper.SetConfigFile(expand(filepath.Join(viper.GetString(KeyLvsDataHome), defaultLvsConfigFile)))
	viper.SetConfigType(defaultLvsConfigType)
	_ = viper.ReadInConfig()

	if HasNode {
		module := &Module{
			Name:          ModuleNode,
			SymlinkEnvKey: EnvNodeHome,
			EnvKeyValues: map[string]string{
				EnvNodeHome: GetPath(KeyNodeSymlink),
			},
			PathValues: nil,
		}
		if runtime.GOOS == "windows" {
			module.PathValues = []string{fmt.Sprintf("%%%s%%", EnvNodeHome)}
		} else {
			module.PathValues = []string{fmt.Sprintf("%%%s%%%cbin", EnvNodeHome, filepath.Separator)}
		}
		Modules[ModuleNode] = module
	}
	Modules[ModuleGo] = &Module{
		Name:          ModuleGo,
		SymlinkEnvKey: EnvGoRoot,
		EnvKeyValues: map[string]string{
			EnvGoRoot: GetPath(KeyGoSymlink),
			EnvGoPath: expand(env(EnvGoPath, defaultGoPath, true)),
		},
		PathValues: []string{
			fmt.Sprintf("%%%s%%%cbin", EnvGoRoot, filepath.Separator),
			fmt.Sprintf("%%%s%%%cbin", EnvGoPath, filepath.Separator),
		},
	}
	afterInit()
}

func expand(path string) string {
	expanded, err := homedir.Expand(path)
	if err != nil {
		panic(err)
	}
	return expanded
}

func Set(key, value string) {
	viper.Set(key, value)
}

func SaveConfig() error {
	dir := filepath.Dir(viper.ConfigFileUsed())
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	v := viper.New()
	v.SetConfigFile(viper.ConfigFileUsed())
	v.SetConfigType(defaultLvsConfigType)
	keys := viper.AllKeys()
	m := make(map[string]any)
	for _, k := range keys {
		value := viper.GetString(k)
		if value != "" {
			m[k] = value
		}
	}
	if err := v.MergeConfigMap(m); err != nil {
		return err
	}
	return v.WriteConfig()
}

func GetString(key string) string {
	return viper.GetString(key)
}

func GetStringWithDefault(key string, defValue string) string {
	if viper.IsSet(key) {
		return viper.GetString(key)
	}
	return defValue
}

func GetPath(key string) string {
	path := viper.GetString(key)
	if path == "" {
		return path
	}
	return expand(path)
}

func Name() string {
	name := filepath.Base(os.Args[0])
	if runtime.GOOS == "windows" {
		name, _ = strings.CutSuffix(name, ".exe")
	}
	return name
}

func Filter(filter func(string) bool) map[string]string {
	keys := viper.AllKeys()
	m := make(map[string]string)
	if filter == nil {
		filter = func(key string) bool {
			return true
		}
	}
	for _, k := range keys {
		if filter(k) {
			m[k] = viper.GetString(k)
		}
	}
	return m
}

func GetWorkspaceUseVersion(module string) (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, fmt.Sprintf("%s%s", module, KeyWorkspaceSuffix))
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), err
}

func env(name, defValue string, raw bool) string {
	if !raw {
		name = fmt.Sprintf("%s%s", EnvLvsPrefix, name)
	}
	value := os.Getenv(name)
	value = strings.Trim(strings.TrimSpace(value), "\"'")
	if value == "" {
		return defValue
	}
	return value
}
