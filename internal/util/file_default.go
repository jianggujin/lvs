//go:build !windows

package util

import (
	"os"
	"path/filepath"
)

func Symlink(linkPath, targetPath string) error {
	if err := os.MkdirAll(filepath.Dir(linkPath), 0755); err != nil {
		return err
	}
	return os.Symlink(targetPath, linkPath)
}

func SymlinkDir(linkPath, targetPath string) error {
	if err := os.MkdirAll(filepath.Dir(linkPath), 0755); err != nil {
		return err
	}
	return os.Symlink(targetPath, linkPath)
}
