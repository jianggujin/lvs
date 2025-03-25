//go:build !windows

package shell

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/mitchellh/go-homedir"
	"jianggujin.com/lvs/internal/invoke"
	"jianggujin.com/lvs/internal/util"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

// ShellType 检测shell类型
func ShellType() string {
	// 优先检查SHELL环境变量
	shellPath := ShellPath()
	if shellPath == "" {
		return "unknown"
	}

	shellName := filepath.Base(shellPath)
	switch {
	case strings.Contains(shellName, "zsh"):
		return "zsh"
	case strings.Contains(shellName, "bash"):
		return "bash"
	case strings.Contains(shellName, "fish"):
		return "fish"
	case strings.Contains(shellName, "csh"), strings.Contains(shellName, "tcsh"):
		return "csh"
	default:
		return "unknown"
	}
}

// ShellPath 获取shell路径
func ShellPath() string {
	shellPath := os.Getenv("SHELL")
	if shellPath == "" {
		// macOS备用检测方法
		if runtime.GOOS == "darwin" {
			currentUser, err := user.Current()
			if err != nil {
				return ""
			}
			output, err := invoke.GetInvoker().Command("dscl", ".", "-read", currentUser.HomeDir, "UserShell")
			if err != nil {
				return ""
			}
			return strings.TrimSpace(strings.Split(string(output), ": ")[1])
		}
	}
	return shellPath
}

func ShellConfigPath(shellType string) string {
	var files []string
	switch shellType {
	case "zsh":
		files = []string{"~/.zshenv", "/etc/zshenv"}
	case "bash":
		files = []string{"~/.bash_profile", "/etc/profile"}
	case "fish":
		files = []string{"~/.config/fish/config.fish", "/etc/fish/config.fish"}
	case "csh":
		files = []string{"~/.cshrc", "/etc/csh.cshrc"}
	default:
		files = []string{"/etc/profile"}
	}
	for i := 0; i < len(files)-1; i++ {
		path, _ := homedir.Expand(files[i])
		if util.Exists(path) {
			return path
		}
	}
	return files[len(files)-1]
}

func NewShellAdapter(shellType, shellConfigPath string) *ShellAdapter {
	if shellType == "" {
		shellType = ShellType()
	}

	getPath := func(sType string) string {
		if shellConfigPath == "" {
			shellConfigPath = ShellConfigPath(sType)
		}
		shellConfigPath, _ = homedir.Expand(shellConfigPath)
		return shellConfigPath
	}

	switch shellType {
	case "zsh":
		return &ShellAdapter{
			ConfigPath:  getPath(shellType),
			KvSeparator: "=",
			VSeparator:  ":",
			Prefix:      "$",
			SetPrefix:   "export",
		}
	case "bash":
		return &ShellAdapter{
			ConfigPath:  getPath(shellType),
			KvSeparator: "=",
			VSeparator:  ":",
			Prefix:      "$",
			SetPrefix:   "export",
		}
	case "fish":
		return &ShellAdapter{
			ConfigPath:     getPath(shellType),
			KvSeparator:    " ",
			VSeparator:     " ",
			Prefix:         "$",
			SetPrefix:      "set",
			SetFlag:        "-x",
			SetFlagPattern: "-[a-zA-Z]+",
		}
	case "csh":
		return &ShellAdapter{
			ConfigPath:  getPath(shellType),
			KvSeparator: " ",
			VSeparator:  ":",
			Wrappers: map[string]string{
				"$": "",
			},
			Prefix:    "${",
			Suffix:    "}",
			SetPrefix: "setenv",
		}
	}
	return &ShellAdapter{
		ConfigPath:  getPath(shellType),
		KvSeparator: "=",
		VSeparator:  ":",
		Prefix:      "$",
		SetPrefix:   "export",
	}
}

type ShellAdapter struct {
	ConfigPath     string            // 配置文件路径
	KvSeparator    string            // 环境变量键值对分隔符
	VSeparator     string            // 环境变量多值分隔符
	Wrappers       map[string]string // 引用变量前缀、后缀
	Prefix         string            // 默认引用变量前缀
	Suffix         string            // 默认引用变量后缀
	SetPrefix      string            // 设置环境变量前缀
	SetFlag        string            // 设置环境变量标记
	SetFlagPattern string            // 设置环境变量标记正则表达式
}

func (a *ShellAdapter) SetEnvs(data []byte, envKeyValues map[string]string, pathValues []string) ([]byte, error) {
	if envKeyValues == nil {
		envKeyValues = map[string]string{}
	}
	items := a.replacePathValues(pathValues)

	var buf bytes.Buffer
	if len(data) > 0 {
		scanner := bufio.NewScanner(bytes.NewReader(data))
		pattern := a.matchPattern()
		re := regexp.MustCompile(pattern)

		for scanner.Scan() {
			line := scanner.Text()
			lineTrim := strings.TrimSpace(line)
			if lineTrim == "" || !strings.HasPrefix(lineTrim, a.SetPrefix) {
				buf.WriteString(line + "\n")
				continue
			}
			matches := re.FindStringSubmatch(lineTrim)
			matchesCount := 3
			if a.SetFlagPattern != "" {
				matchesCount = 4
			}
			if len(matches) == matchesCount {
				envKey := strings.TrimSpace(matches[matchesCount-2])
				if envKeyValues[envKey] != "" {
					continue
				}
				if "PATH" == envKey {
					envValue := strings.TrimSpace(matches[matchesCount-1])
					values := a.splitPathValue(envValue)
					var copySlice []string
					changed := false
					for _, value := range values {
						if items.contains(value) {
							changed = true
							continue
						}
						copySlice = append(copySlice, a.escapeEnvValue(value))
					}
					if changed {
						if !a.isEmptyOrOnlyPath(copySlice) {
							if a.SetFlagPattern != "" {
								buf.WriteString(fmt.Sprintf("%s %s PATH %s\n", a.SetPrefix, matches[1], strings.Join(copySlice, a.VSeparator)))
							} else {
								buf.WriteString(a.exportEnv("PATH", strings.Join(copySlice, a.VSeparator)))
							}
						}
						continue
					}
				}
			}
			buf.WriteString(line + "\n")
		}
		if err := scanner.Err(); err != nil {
			return nil, err
		}
	}
	for key, value := range envKeyValues {
		buf.WriteString(a.exportEnv(key, a.escapeEnvValue(value)))
	}
	if len(items) > 0 {
		buf.WriteString(a.exportEnv("PATH", items.toPath(a)))
	}

	return buf.Bytes(), nil
}

func (a *ShellAdapter) DelEnvs(data []byte, envKeys []string, pathValues []string) ([]byte, error) {
	items := a.replacePathValues(pathValues)

	var buf bytes.Buffer
	keyMap := make(map[string]bool)
	for _, key := range envKeys {
		keyMap[key] = true
	}
	if len(data) > 0 {
		scanner := bufio.NewScanner(bytes.NewReader(data))
		pattern := a.matchPattern()
		re := regexp.MustCompile(pattern)

		for scanner.Scan() {
			line := scanner.Text()
			lineTrim := strings.TrimSpace(line)
			if lineTrim == "" || !strings.HasPrefix(lineTrim, a.SetPrefix) {
				buf.WriteString(line + "\n")
				continue
			}
			matches := re.FindStringSubmatch(lineTrim)
			matchesCount := 3
			if a.SetFlagPattern != "" {
				matchesCount = 4
			}
			if len(matches) == matchesCount {
				envKey := strings.TrimSpace(matches[matchesCount-2])
				if keyMap[envKey] {
					continue
				}
				if "PATH" == envKey {
					envValue := strings.TrimSpace(matches[matchesCount-1])
					values := a.splitPathValue(envValue)
					var copySlice []string
					changed := false
					for _, value := range values {
						if items.contains(value) {
							changed = true
							continue
						}
						copySlice = append(copySlice, a.escapeEnvValue(value))
					}
					if changed {
						if !a.isEmptyOrOnlyPath(copySlice) {
							if a.SetFlagPattern != "" {
								buf.WriteString(fmt.Sprintf("%s %s PATH %s\n", a.SetPrefix, matches[1], strings.Join(copySlice, a.VSeparator)))
							} else {
								buf.WriteString(a.exportEnv("PATH", strings.Join(copySlice, a.VSeparator)))
							}
						}
						continue
					}
				}
			}
			buf.WriteString(line + "\n")
		}
		if err := scanner.Err(); err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

// 构建匹配导出环境变量的正则表达式
func (a *ShellAdapter) matchPattern() string {
	if a.SetFlagPattern == "" {
		a.SetFlagPattern = a.SetFlag
	}
	if a.SetFlagPattern == "" {
		return fmt.Sprintf(`^%s\s+([A-Za-z_][A-Za-z0-9_]*)%s(.*)$`, a.SetPrefix, a.KvSeparator)
	}
	return fmt.Sprintf(`^%s\s+(%s)\s+([A-Za-z_][A-Za-z0-9_]*)%s(.*)$`, a.SetPrefix, a.SetFlagPattern, a.KvSeparator)
}

type pathValue struct {
	result    string
	extracted []string
	wrappers  []string
}

func (i *pathValue) Contains(value string) bool {
	if value == i.result {
		return true
	}
	for _, w := range i.wrappers {
		if value == w {
			return true
		}
	}
	return false
}

type pathValues []*pathValue

func (i pathValues) contains(value string) bool {
	for _, w := range i {
		if w.Contains(value) {
			return true
		}
	}
	return false
}

func (i pathValues) toPath(a *ShellAdapter) string {
	var list []string
	for _, w := range i {
		list = append(list, a.escapeEnvValue(w.result))
	}
	list = append(list, a.refEnvKey("PATH"))
	return strings.Join(list, a.VSeparator)
}

func (a *ShellAdapter) replacePathValues(inputs []string) pathValues {
	if len(inputs) == 0 {
		return nil
	}
	rgx := regexp.MustCompile(`%([^%]+)%`)

	var items pathValues
	for _, input := range inputs {
		var extracted []string
		result := rgx.ReplaceAllStringFunc(input, func(match string) string {
			key := match[1 : len(match)-1]
			extracted = append(extracted, key)
			return a.Prefix + key + a.Suffix
		})
		var wrappers []string
		if a.Wrappers != nil {
			for prefix, suffix := range a.Wrappers {
				wrappers = append(wrappers, rgx.ReplaceAllStringFunc(input, func(match string) string {
					key := match[1 : len(match)-1]
					extracted = append(extracted, key)
					return prefix + key + suffix
				}))
			}
		}
		items = append(items, &pathValue{
			result:    result,
			extracted: extracted,
			wrappers:  wrappers,
		})
	}

	return items
}

// 格式化导出环境变量语句
func (a *ShellAdapter) exportEnv(key, value string) string {
	if a.SetFlag != "" {
		// set -x LVS_HOME /opt/lvs
		return fmt.Sprintf("%s %s %s%s%s\n", a.SetPrefix, a.SetFlag, key, a.KvSeparator, value)
	} else {
		// export LVS_HOME=/opt/lvs
		return fmt.Sprintf("%s %s%s%s\n", a.SetPrefix, key, a.KvSeparator, value)
	}
}

// 格式化引用环境变量
func (a *ShellAdapter) refEnvKey(envKey string) string {
	return fmt.Sprintf("%s%s%s", a.Prefix, envKey, a.Suffix)
}

// escapeEnvValue 判断环境变量的值是否需要添加双引号或转义
func (a *ShellAdapter) escapeEnvValue(value string) string {
	// 主要是文件路径，暂时只处理空格
	if strings.Contains(value, " ") {
		return fmt.Sprintf("\"%s\"", value)
	}
	return value
}

// 将指定的环境变量值进行拆分
func (a *ShellAdapter) splitPathValue(envValue string) []string {
	var result []string
	var current strings.Builder
	inQuotes := false
	quoteChar := rune(0) // 记录使用的引号类型（单引号或双引号）
	escaped := false

	for _, char := range envValue {
		switch {
		case char == '\\' && !escaped:
			// 处理转义字符
			escaped = true
		case (char == '"' || char == '\'') && !escaped:
			// 遇到引号时，切换 inQuotes 状态
			if inQuotes {
				// 如果已经在引号内，只有相同的引号才会结束
				if char == quoteChar {
					inQuotes = false
				} else {
					current.WriteRune(char) // 不同引号嵌套，视作普通字符
				}
			} else {
				// 进入引号模式
				inQuotes = true
				quoteChar = char
			}
		case char == rune(a.VSeparator[0]) && !inQuotes:
			// 遇到分隔符且不在引号内，拆分
			if current.Len() > 0 {
				result = append(result, current.String())
				current.Reset()
			}
		default:
			// 追加正常字符
			current.WriteRune(char)
			escaped = false
		}
	}

	// 追加最后一个值
	if current.Len() > 0 {
		result = append(result, current.String())
	}

	return result
}

// 判断环境变量值是否为空或仅包含PATH引用
func (a *ShellAdapter) isEmptyOrOnlyPath(envValues []string) bool {
	length := len(envValues)
	if length == 1 {
		value := envValues[0]
		if value == a.refEnvKey("PATH") {
			return true
		}
		if a.Wrappers != nil {
			for prefix, suffix := range a.Wrappers {
				if value == fmt.Sprintf("%sPATH%s", prefix, suffix) {
					return true
				}
			}
		}
	}
	return false
}
