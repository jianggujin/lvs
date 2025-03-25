//go:build windows

package elevated

import (
	"fmt"
	"io"
	"jianggujin.com/lvs/internal/invoke"
	"path/filepath"
)

// Symlink 创建文件软连接
func Symlink(link, target string) error {
	return Elevate("mklink", link, target)
}

// SymlinkDir 创建目录符号软链接
func SymlinkDir(link, target string) error {
	return Elevate("mklink", "/D", link, target)
}

func Elevate(name string, args ...string) error {
	return run("elevate", func(writer io.StringWriter) error {
		_, err := writer.WriteString(`@setlocal
@echo off

%*
if %ERRORLEVEL% LSS 1 goto :EOF

:: The command failed without elevation, try with elevation
set CMD=%*
set APP=%1
start wscript //nologo "%~dpn0.vbs" %*`)
		return err
	}, func(writer io.StringWriter) error {
		_, err := writer.WriteString(`Set Shell = CreateObject("Shell.Application")
Set WShell = WScript.CreateObject("WScript.Shell")
Set ProcEnv = WShell.Environment("PROCESS")

cmd = ProcEnv("CMD")
app = ProcEnv("APP")
args= Right(cmd,(Len(cmd)-Len(app)))

If (WScript.Arguments.Count >= 1) Then
  Shell.ShellExecute app, args, "", "runas", 0
Else
  WScript.Quit
End If`)
		return err
	}, name, args...)
}

func RunAs(name string, args ...string) error {
	return run("runas", func(writer io.StringWriter) error {
		_, err := writer.WriteString(`@setlocal
@echo off

set CMD=%*
set APP=%1
start wscript //nologo "%~dpn0.vbs" %*`)
		return err
	}, func(writer io.StringWriter) error {
		_, err := writer.WriteString(`Set Shell = CreateObject("Shell.Application")
Set WShell = WScript.CreateObject("WScript.Shell")
Set ProcEnv = WShell.Environment("PROCESS")

cmd = ProcEnv("CMD")
app = ProcEnv("APP")
args= Right(cmd,(Len(cmd)-Len(app)))

If (WScript.Arguments.Count >= 1) Then
  Shell.ShellExecute app, args, "", "runas", 0
Else
  WScript.Quit
End If`)
		return err
	}, name, args...)
}

func run(script string, cmdConsumer func(io.StringWriter) error, vbsConsumer func(io.StringWriter) error, name string, args ...string) error {
	cmdName := fmt.Sprintf("%s.cmd", script)
	cmdPath, err := ReleaseDynamicScript(cmdName, cmdConsumer)
	if err != nil {
		return err
	}
	_, err = ReleaseDynamicScript(fmt.Sprintf("%s.vbs", script), vbsConsumer)
	if err != nil {
		return err
	}
	dir := filepath.Dir(cmdPath)
	return invoke.GetInvoker().CommandOptions(cmdPath, append([]string{"cmd", "/C", name}, args...), invoke.WithDir(dir))
}
