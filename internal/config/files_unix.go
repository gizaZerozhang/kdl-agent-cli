//go:build !windows

package config

import (
	"fmt"
	"os"
)

func restrictAccess(path string, directory bool) error {
	mode := os.FileMode(0o600)
	if directory {
		mode = 0o700
	}
	if err := os.Chmod(path, mode); err != nil {
		return fmt.Errorf("收紧配置权限 %s 失败；请检查文件归属", path)
	}
	return nil
}

func replaceFile(from, to string) error { return os.Rename(from, to) }
