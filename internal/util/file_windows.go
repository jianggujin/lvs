//go:build windows

package util

import (
	"jianggujin.com/lvs/internal/elevated"
	"os"
	"path/filepath"
	"time"
)

func Symlink(linkPath, targetPath string) error {
	if err := os.MkdirAll(filepath.Dir(linkPath), 0755); err != nil {
		return err
	}
	err := elevated.Symlink(linkPath, targetPath)
	if err == nil {
		time.Sleep(400)
		_ = elevated.SendEnvironmentUpdate()
	}
	return err
}

func SymlinkDir(linkPath, targetPath string) error {
	if err := os.MkdirAll(filepath.Dir(linkPath), 0755); err != nil {
		return err
	}
	err := elevated.SymlinkDir(linkPath, targetPath)
	if err == nil {
		time.Sleep(400)
		_ = elevated.SendEnvironmentUpdate()
	}
	return err
}
