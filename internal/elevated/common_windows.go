//go:build windows

package elevated

import (
	"errors"
	"io"
	"jianggujin.com/lvs/internal/config"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

var (
	user32             = syscall.NewLazyDLL("user32.dll")
	sendMessageTimeout = user32.NewProc("SendMessageTimeoutW")
)

const (
	HWND_BROADCAST   = 0xFFFF
	WM_SETTINGCHANGE = 0x001A
	SMTO_ABORTIFHUNG = 0x0002
)

func ReleaseDynamicScript(name string, consumer func(io.StringWriter) error) (string, error) {
	home := config.GetPath(config.KeyLvsScriptHome)
	path := filepath.Join(home, name)
	_, err := os.Stat(path)
	if err == nil {
		return path, err
	}
	// 释放文件
	dir := filepath.Dir(path)
	if err = os.MkdirAll(dir, os.ModePerm); err != nil {
		return path, err
	}
	writer, err := os.Create(path)
	if err != nil {
		return path, err
	}
	defer writer.Close()
	return path, consumer(writer)
}

func SendEnvironmentUpdate() error {
	lpParam, err := syscall.UTF16PtrFromString("Environment")
	if err != nil {
		return err
	}

	ret, _, callErr := sendMessageTimeout.Call(
		HWND_BROADCAST,
		WM_SETTINGCHANGE,
		0,
		uintptr(unsafe.Pointer(lpParam)),
		SMTO_ABORTIFHUNG,
		5000, // 超时5秒
		0,
	)

	// 检查是否调用成功
	if ret == 0 {
		return errors.New("broadcast message sending failed")
	}

	if callErr != nil && callErr.(syscall.Errno) != 0 {
		return callErr
	}
	return nil
}
