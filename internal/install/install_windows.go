//go:build windows

package install

import (
	"errors"
	"fmt"
	"io"
	"jianggujin.com/lvs/internal/elevated"
	"jianggujin.com/lvs/internal/invoke"
	"os"
	"regexp"
	"strings"
	"time"
)

const (
	prefixUserEnv = `HKEY_CURRENT_USER\Environment`
	prefixSysEnv  = `HKEY_LOCAL_MACHINE\System\CurrentControlSet\Control\Session Manager\Environment`
)

var UserCancel = errors.New("an abnormal operation result was detected, which may have caused the user to cancel the operation")

func Install(envKeyValues map[string]string, pathValues []string) error {
	if (envKeyValues == nil || len(envKeyValues) == 0) && len(pathValues) == 0 {
		return nil
	}
	path, err := elevated.ReleaseDynamicScript(fmt.Sprintf("install%s.vbs", time.Now().Format("20060102150405")), func(writer io.StringWriter) error {
		if _, err := writer.WriteString(`Set WShell = CreateObject("WScript.Shell")
		
Dim prefix(1), i, j
prefix(0) = "HKEY_CURRENT_USER\Environment\"
prefix(1) = "HKEY_LOCAL_MACHINE\System\CurrentControlSet\Control\Session Manager\Environment\"

`); err != nil {
			return err
		}

		if envKeyValues != nil && len(envKeyValues) > 0 {
			if _, err := writer.WriteString(fmt.Sprintf("Dim envKeyValues(%d)\n", len(envKeyValues)*2-1)); err != nil {
				return err
			}
			index := 0
			for envKey, envValue := range envKeyValues {
				if _, err := writer.WriteString(fmt.Sprintf("envKeyValues(%d) = \"%s\"\nenvKeyValues(%d) = \"%s\"\n", index, envKey, index+1, envValue)); err != nil {
					return err
				}
				index += 2
			}
			if _, err := writer.WriteString(`
For i = 0 To UBound(envKeyValues) Step 2
    Dim envKey, envValue
    envKey = envKeyValues(i)
    envValue = envKeyValues(i + 1)

    For j = 0 To UBound(prefix)
        On Error Resume Next
        WShell.RegWrite prefix(j) & envKey, envValue, "REG_EXPAND_SZ"
        If Err.Number <> 0 Then
            On Error GoTo 0
            WScript.Quit 1
        End If
        On Error GoTo 0
    Next
Next

`); err != nil {
				return err
			}
		}
		if len(pathValues) > 0 {
			if _, err := writer.WriteString(fmt.Sprintf("Dim pathValues(%d)\n", len(pathValues)-1)); err != nil {
				return err
			}
			index := 0
			for _, pathValue := range pathValues {
				if _, err := writer.WriteString(fmt.Sprintf("pathValues(%d) = \"%s\"\n", index, pathValue)); err != nil {
					return err
				}
				index++
			}
			if _, err := writer.WriteString(`
For i = 0 To UBound(prefix)
    Dim regKey, pathValue
    On Error Resume Next
    regKey = prefix(i) & "Path"
    pathValue = WShell.RegRead(regKey)
    If Err.Number <> 0 Then
        On Error GoTo 0
        WScript.Quit 2
    End If
    On Error GoTo 0

    For j = 0 To UBound(pathValues)
        If InStr(pathValue, pathValues(j)) = False Then
            pathValue = pathValue & ";" & pathValues(j) & ";"
        End If
    Next

    Do While InStr(pathValue, ";;") > 0
        pathValue = Replace(pathValue, ";;", ";")
    Loop
    On Error Resume Next
    WShell.RegWrite regKey, pathValue, "REG_EXPAND_SZ"
    If Err.Number <> 0 Then
        On Error GoTo 0
        WScript.Quit 3
    End If
    On Error GoTo 0
Next

`); err != nil {
				return err
			}
		}
		if _, err := writer.WriteString(`Set WShell = Nothing
WScript.Quit`); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	defer os.Remove(path)
	if err = elevated.RunAs("wscript", path); err != nil {
		return err
	}
	time.Sleep(400 * time.Millisecond)
	_ = elevated.SendEnvironmentUpdate()
	var prefixes = []string{prefixUserEnv, prefixSysEnv}
	var value string
	// 检测自定义
	for envKey, envValue := range envKeyValues {
		for _, prefix := range prefixes {
			value, err = getEnv(prefix, envKey)
			if err == nil && value != envValue {
				return UserCancel
			}
		}
	}
	// 检测Path
	for _, prefix := range prefixes {
		value, err = getEnv(prefix, "Path")
		if err == nil {
			for _, pathValue := range pathValues {
				if !strings.Contains(value, pathValue) {
					return UserCancel
				}
			}
		}
	}
	return err
}

func Uninstall(envKeys []string, pathValues []string) error {
	if len(envKeys) == 0 && len(pathValues) == 0 {
		return nil
	}
	path, err := elevated.ReleaseDynamicScript(fmt.Sprintf("uninstall%s.vbs", time.Now().Format("20060102150405")), func(writer io.StringWriter) error {
		if _, err := writer.WriteString(`Set WShell = CreateObject("WScript.Shell")

Dim prefix(1), i
prefix(0) = "HKEY_CURRENT_USER\Environment\"
prefix(1) = "HKEY_LOCAL_MACHINE\System\CurrentControlSet\Control\Session Manager\Environment\"

`); err != nil {
			return err
		}

		if len(envKeys) > 0 {
			if _, err := writer.WriteString(fmt.Sprintf("Dim envKeys(%d)\n", len(envKeys)-1)); err != nil {
				return err
			}
			index := 0
			for _, envKey := range envKeys {
				if _, err := writer.WriteString(fmt.Sprintf("envKeys(%d) = \"%s\"\n", index, envKey)); err != nil {
					return err
				}
				index++
			}
			if _, err := writer.WriteString(`
For i = 0 To UBound(envKeys)
    Dim envKey
    envKey = envKeys(i)
    For j = 0 To UBound(prefix)
        On Error Resume Next
        WShell.RegRead(prefix(j) & envKey)
        If Err.Number = 0 Then
            On Error GoTo 0
            WShell.RegDelete prefix(j) & envKey
        End If
        On Error GoTo 0
    Next
Next

`); err != nil {
				return err
			}
		}
		if len(pathValues) > 0 {
			if _, err := writer.WriteString(fmt.Sprintf("Dim pathValues(%d)\n", len(pathValues)-1)); err != nil {
				return err
			}
			index := 0
			for _, pathValue := range pathValues {
				if _, err := writer.WriteString(fmt.Sprintf("pathValues(%d) = \"%s\"\n", index, pathValue)); err != nil {
					return err
				}
				index++
			}
			if _, err := writer.WriteString(`
For i = 0 To UBound(prefix)
    Dim regKey, regValue
    On Error Resume Next
    regKey = prefix(i) & "Path"
    regValue = WShell.RegRead(regKey)
    If Err.Number <> 0 Then
        On Error GoTo 0
        WScript.Quit 1
    End If
    On Error GoTo 0

    For j = 0 To UBound(pathValues)
        regValue = Replace(regValue, pathValues(j), "")
    Next

    Do While InStr(regValue, ";;") > 0
        regValue = Replace(regValue, ";;", ";")
    Loop
    On Error Resume Next
    WShell.RegWrite regKey, regValue, "REG_EXPAND_SZ"
    If Err.Number <> 0 Then
        On Error GoTo 0
        WScript.Quit 2
    End If
    On Error GoTo 0
Next

`); err != nil {
				return err
			}
		}
		if _, err := writer.WriteString(`Set WShell = Nothing
WScript.Quit`); err != nil {
			return err
		}
		return nil
	})
	defer os.Remove(path)
	if err = elevated.RunAs("wscript", path); err != nil {
		return err
	}

	time.Sleep(400 * time.Millisecond)
	_ = elevated.SendEnvironmentUpdate()

	var prefixes = []string{prefixUserEnv, prefixSysEnv}
	var value string
	// 检测自定义
	for _, key := range envKeys {
		for _, prefix := range prefixes {
			value, err = getEnv(prefix, key)
			if err == nil && value != "" {
				return UserCancel
			}
		}
	}
	// 检测Path
	for _, prefix := range prefixes {
		value, err = getEnv(prefix, "Path")
		if err == nil {
			for _, pathValue := range pathValues {
				if strings.Contains(value, pathValue) {
					return UserCancel
				}
			}
		}
	}
	return err
}

func getEnv(keyPrefix, name string) (string, error) {
	// reg query "HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Control\Session Manager\Environment" /v PATH 2>nul
	data, err := invoke.GetInvoker().Command("reg", "query", keyPrefix, "/v", name)
	if err != nil {
		// 处理中文乱码
		// reader := transform.NewReader(bytes.NewReader(data), simplifiedchinese.GBK.NewDecoder())
		// d, e := io.ReadAll(reader)
		// 出现错误，直接忽略认为不存在
		return "", nil
	}
	value := strings.TrimSpace(string(data))
	if value == "" {
		return "", err
	}
	value, _ = strings.CutPrefix(value, keyPrefix)
	value = strings.TrimSpace(value)
	items := regexp.MustCompile(" +").Split(value, 3)
	if len(items) != 3 {
		return "", errors.New("the obtained environment variable information is illegal")
	}
	return strings.TrimSpace(items[2]), nil
}
