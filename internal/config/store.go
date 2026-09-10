package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gofrs/flock"
	"github.com/pelletier/go-toml/v2"
)

func regularFile(path string) error {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("配置文件 %s 不是可访问的普通文件；请移除符号链接或修复路径", path)
	}
	return nil
}

func secureDir(dir string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("创建配置目录 %s 失败；请检查路径和权限", dir)
	}
	info, err := os.Lstat(dir)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("配置目录 %s 不可用；请使用真实目录", dir)
	}
	return restrictAccess(dir, true)
}

func atomicWrite(path string, raw []byte) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("配置文件路径为空；请检查 --config")
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("创建配置目录 %s 失败；请检查访问权限", dir)
	}
	if err := regularFile(path); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".kdl-write-*")
	if err != nil {
		return fmt.Errorf("创建配置临时文件 %s 失败；请检查目录权限", dir)
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	defer f.Close()
	if err := restrictAccess(tmp, false); err != nil {
		return err
	}
	if _, err := f.Write(raw); err != nil {
		return fmt.Errorf("写入配置文件 %s 失败；请检查磁盘空间", path)
	}
	if err := f.Sync(); err != nil {
		return fmt.Errorf("同步配置文件 %s 失败；请检查磁盘状态", path)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("关闭配置文件 %s 失败；请重试", path)
	}
	if err := replaceFile(tmp, path); err != nil {
		return fmt.Errorf("替换配置文件 %s 失败；原文件保留，请检查权限后重试", path)
	}
	return nil
}

func withCredentials(update func(string, *credentials) error) error {
	dir, err := homeDir()
	if err != nil {
		return err
	}
	if err := secureDir(dir); err != nil {
		return err
	}
	lockPath := filepath.Join(dir, ".credentials.lock")
	if err := regularFile(lockPath); err != nil {
		return err
	}
	lock := flock.New(lockPath)
	defer lock.Close()
	ok, err := lock.TryLock()
	if err != nil || !ok {
		return errors.New("凭证配置文件正在使用或无法加锁；请等待其他登录/退出结束后重试")
	}
	if err := restrictAccess(lockPath, false); err != nil {
		return err
	}
	path := filepath.Join(dir, "credentials.toml")
	value, err := readCredentials(path)
	if err != nil {
		return err
	}
	return update(path, &value)
}

// SaveLogin 先保存已验证凭证，再切换普通配置；失败时恢复凭证原文。
// 多文件无法单次 rename；即使进程中断，网关绑定仍阻止错误地址复用凭证。
func SaveLogin(configPath, gateway, token string) error {
	canonical, err := NormalizeGateway(gateway)
	if err != nil {
		return err
	}
	if strings.TrimSpace(token) == "" {
		return errors.New("凭证为空；请重新执行 auth login")
	}
	return withCredentials(func(path string, value *credentials) error {
		if filepath.Clean(configPath) == filepath.Clean(path) || filepath.Clean(configPath) == filepath.Join(filepath.Dir(path), ".credentials.lock") {
			return errors.New("普通配置文件不能使用凭证文件路径；请移除 --config 后重试")
		}
		if _, err := readConfig(configPath); err != nil {
			return err
		}
		previous, readErr := os.ReadFile(path)
		if readErr != nil && !errors.Is(readErr, os.ErrNotExist) {
			return fmt.Errorf("读取凭证配置文件 %s 失败；请检查权限", path)
		}
		value.Gateways[canonical] = strings.TrimSpace(token)
		raw, err := toml.Marshal(value)
		if err != nil {
			return errors.New("序列化凭证配置失败；请检查配置结构")
		}
		if err := atomicWrite(path, raw); err != nil {
			return err
		}
		if err := Save(configPath, File{GatewayURL: canonical}); err != nil {
			var restoreErr error
			if errors.Is(readErr, os.ErrNotExist) {
				restoreErr = os.Remove(path)
			} else {
				restoreErr = atomicWrite(path, previous)
			}
			if restoreErr != nil {
				return fmt.Errorf("保存配置失败，恢复凭证配置文件 %s 也失败；请检查权限并重新登录", path)
			}
			return err
		}
		return nil
	})
}

func Logout(gateway string) error {
	canonical, err := NormalizeGateway(gateway)
	if err != nil {
		return err
	}
	return withCredentials(func(path string, value *credentials) error {
		delete(value.Gateways, canonical)
		raw, err := toml.Marshal(value)
		if err != nil {
			return errors.New("序列化凭证配置失败；请检查配置结构")
		}
		return atomicWrite(path, raw)
	})
}
