package util

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"errors"
	"fmt"
	"github.com/ulikunitz/xz"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func ResetSymlink(linkPath, targetPath string, dir bool) error {
	target, err := ReadSymlink(linkPath)
	if err != nil {
		return err
	}
	if target != "" && target != targetPath {
		if err = os.Remove(linkPath); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	if dir {
		err = SymlinkDir(linkPath, targetPath)
	} else {
		err = Symlink(linkPath, targetPath)
	}
	if err != nil {
		return err
	}
	time.Sleep(250 * time.Millisecond)
	newTargetPath, err := ReadSymlink(linkPath)
	if err != nil {
		return err
	}
	if newTargetPath != targetPath {
		return errors.New("an abnormal operation result was detected, which may have caused the user to cancel the operation")
	}
	return nil
}

func ReadSymlink(linkPath string) (string, error) {
	info, err := os.Lstat(linkPath)
	if err != nil {
		if os.IsNotExist(err) {
			// 如果软链接不存在，返回空字符串
			return "", nil
		}
		return "", err
	}
	// 如果不是软链接，返回错误
	if info.Mode()&os.ModeSymlink == 0 {
		return "", fmt.Errorf("path [%s] is not a symlink", linkPath)
	}
	// 获取软链接的目标路径
	return os.Readlink(linkPath)
}

func IsSymlink(linkPath string) (bool, bool, error) {
	info, err := os.Lstat(linkPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, false, nil
		}
		return false, false, err
	}
	return info.Mode()&os.ModeSymlink != 0, true, nil
}

func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil || !os.IsNotExist(err)
}

type UnArchive interface {
	ChangeMax64(int64)
	Add64(int64) error
	Write(p []byte) (n int, err error)
}

func UnzipFile(src, dest string, function func(string) (string, error), unArchive UnArchive) error {
	zipReader, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer zipReader.Close()

	// Precompute total raw size and file count
	rawSize := uint64(0)

	for _, zipFile := range zipReader.File {
		rawSize += zipFile.UncompressedSize64
	}
	if unArchive != nil {
		unArchive.ChangeMax64(int64(rawSize))
	}

	nameProcessor := func(name string) (string, error) {
		if strings.Contains(name, "..") {
			return "", nil
		}
		if function != nil {
			return function(name)
		}
		return name, nil
	}
	if function == nil {
		// Simplify processor when no custom function
		nameProcessor = func(name string) (string, error) {
			if strings.Contains(name, "..") {
				return "", nil
			}
			return name, nil
		}
	}
	for _, zipFile := range zipReader.File {
		newName, err := nameProcessor(zipFile.Name)
		if err != nil {
			return err
		}
		// Handle skipped files
		if newName == "" {
			if unArchive == nil {
				continue
			}
			if err = unArchive.Add64(int64(zipFile.UncompressedSize64)); err != nil {
				return err
			}
			continue
		}

		// Process file entry with resource safety
		if err = func() error {
			entry, err := zipFile.Open()
			if err != nil {
				return err
			}
			defer entry.Close()

			targetPath := filepath.Join(dest, newName)

			// Handle directories
			if zipFile.FileInfo().IsDir() {
				if err := os.MkdirAll(targetPath, zipFile.Mode()); err != nil {
					return err
				}
				if unArchive == nil {
					return nil
				}
				return unArchive.Add64(int64(zipFile.UncompressedSize64))
			}

			// Create target file
			fileWriter, err := os.Create(targetPath)
			if err != nil {
				if os.IsNotExist(err) {
					dir := filepath.Dir(targetPath)
					if err = os.MkdirAll(dir, 0755); err != nil {
						return err
					}
					if fileWriter, err = os.Create(targetPath); err != nil {
						return err
					}
				}
				return err
			}
			defer fileWriter.Close()

			var writer io.Writer = fileWriter
			if unArchive != nil {
				writer = io.MultiWriter(fileWriter, unArchive)
			}
			_, err = io.Copy(writer, entry)
			return err
		}(); err != nil {
			return err
		}
	}
	return nil
}

func UntarFile(src, dest string, function func(string) (string, error), unArchive UnArchive) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	var baseReader io.Reader = srcFile
	if strings.HasSuffix(strings.ToLower(src), ".gz") {
		gzReader, err := gzip.NewReader(srcFile)
		if err != nil {
			return err
		}
		defer gzReader.Close()
		baseReader = gzReader
	} else if strings.HasSuffix(strings.ToLower(src), ".xz") {
		xzReader, err := xz.NewReader(srcFile)
		if err != nil {
			return err
		}
		baseReader = xzReader
	}

	nameProcessor := func(name string) (string, error) {
		if strings.Contains(name, "..") {
			return "", nil
		}
		if function != nil {
			return function(name)
		}
		return name, nil
	}

	var tarReader = tar.NewReader(baseReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar read error: %w", err)
		}

		processedName, err := nameProcessor(header.Name)
		if err != nil {
			return err
		}

		// 跳过处理空文件名
		if processedName == "" {
			continue
		}

		path := filepath.Join(dest, processedName)

		if tar.TypeDir == header.Typeflag {
			if err := os.MkdirAll(path, header.FileInfo().Mode()); err != nil {
				return err
			}
			continue
		}
		if tar.TypeReg != header.Typeflag {
			continue
		}

		err = func() error {
			fileWriter, err := os.Create(path)
			if err != nil {
				if os.IsNotExist(err) {
					dir := filepath.Dir(path)
					if err = os.MkdirAll(dir, 0755); err != nil {
						return err
					}
					if fileWriter, err = os.Create(path); err != nil {
						return err
					}
				}
				return err
			}
			defer fileWriter.Close()

			var writer io.Writer = fileWriter
			if unArchive != nil {
				writer = io.MultiWriter(fileWriter, unArchive)
			}
			_, err = io.Copy(writer, tarReader)
			return err
		}()
		if err != nil {
			return err
		}
	}

	return nil
}

func ExecExists(execPath string) bool {
	if Exists(execPath) {
		return true
	}

	if runtime.GOOS == "windows" {
		var exts []string
		x := os.Getenv(`PATHEXT`)
		if x != "" {
			for _, e := range strings.Split(strings.ToLower(x), `;`) {
				if e == "" {
					continue
				}
				if e[0] != '.' {
					e = "." + e
				}
				exts = append(exts, e)
			}
		} else {
			exts = []string{".com", ".exe", ".bat", ".cmd"}
		}
		for _, ext := range exts {
			// 存在后缀且文件存在
			if strings.HasPrefix(filepath.Base(execPath), ext) && Exists(execPath) {
				return true
			}
			if Exists(execPath + ext) {
				return true
			}
		}
	}
	return false
}
