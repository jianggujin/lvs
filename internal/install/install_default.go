//go:build !windows

package install

import (
	"bytes"
	"fmt"
	"io"
	"jianggujin.com/lvs/internal/config"
	"jianggujin.com/lvs/internal/invoke"
	"jianggujin.com/lvs/internal/shell"
	"os"
	"path/filepath"
	"time"
)

func Install(envKeyValues map[string]string, pathValues []string) error {
	if (envKeyValues == nil || len(envKeyValues) == 0) && len(pathValues) == 0 {
		return nil
	}
	adapter := shell.NewShellAdapter(config.GetString(config.KeyShellType), config.GetPath(config.KeyShellConfigPath))
	// 备份文件
	_, err := os.Stat(adapter.ConfigPath)
	var data []byte
	if err == nil {
		if data, err = backup(adapter.ConfigPath); err != nil {
			return err
		}
	}
	if data, err = adapter.SetEnvs(data, envKeyValues, pathValues); err != nil {
		return err
	}

	writer, err := os.Create(adapter.ConfigPath)
	if err != nil {
		return err
	}
	defer writer.Close()
	if _, err = writer.Write(bytes.TrimSpace(data)); err != nil {
		return err
	}

	_, _ = invoke.GetInvoker().Command("source", adapter.ConfigPath)
	return err
}

func Uninstall(envKeys []string, pathValues []string) error {
	if len(envKeys) == 0 && len(pathValues) == 0 {
		return nil
	}
	adapter := shell.NewShellAdapter(config.GetString(config.KeyShellType), config.GetPath(config.KeyShellConfigPath))
	// 备份文件
	_, err := os.Stat(adapter.ConfigPath)
	var data []byte
	if err == nil {
		if data, err = backup(adapter.ConfigPath); err != nil {
			return err
		}
	}
	if data, err = adapter.DelEnvs(data, envKeys, pathValues); err != nil {
		return err
	}

	writer, err := os.Create(adapter.ConfigPath)
	if err != nil {
		return err
	}
	defer writer.Close()
	if _, err = writer.Write(bytes.TrimSpace(data)); err != nil {
		return err
	}

	_, _ = invoke.GetInvoker().Command("source", adapter.ConfigPath)
	return err
}

func backup(path string) ([]byte, error) {
	reader, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	home := config.GetPath(config.KeyLvsBackupHome)
	if err = os.MkdirAll(home, os.ModePerm); err != nil {
		return nil, err
	}
	writer, err := os.Create(filepath.Join(home, fmt.Sprintf("%s.bak_%s", filepath.Base(path), time.Now().Format("20060102150405"))))
	if err != nil {
		return nil, err
	}
	defer writer.Close()

	_, err = writer.Write(data)
	return data, err
}
