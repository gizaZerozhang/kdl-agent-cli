package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/gofrs/flock"
)

func isolatedHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	for _, name := range []string{"KDL_AGENT_CONFIG", "KDL_AGENT_GATEWAY_URL", "KDL_AGENT_TOKEN"} {
		t.Setenv(name, "")
	}
	SetPathOverride("")
	t.Cleanup(func() { SetPathOverride("") })
	return home
}

func TestDefaultHomeAndGateway(t *testing.T) {
	home := isolatedHome(t)
	r, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if r.ConfigPath != filepath.Join(home, ".kdl", "config.toml") || r.GatewayURL != DefaultGatewayURL || r.Token != "" {
		t.Fatal("默认路径、网关或空凭证不正确")
	}
	if ValidateForAPI(r) == nil {
		t.Fatal("未配置应拒绝调用")
	}
	if _, err := os.Stat(filepath.Join(home, ".kdl")); !os.IsNotExist(err) {
		t.Fatal("只读加载不应创建目录")
	}
}

func TestGatewayBindingEnvironmentAndLogout(t *testing.T) {
	isolatedHome(t)
	path, _ := ResolvePath()
	const first = "https://first.example"
	const second = "https://second.example"
	if err := SaveLogin(path, first+":443/", "sentinel-first"); err != nil {
		t.Fatal(err)
	}
	if err := SaveLogin(path, second, "sentinel-second"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("KDL_AGENT_GATEWAY_URL", first)
	r, err := Load()
	if err != nil || r.Token != "sentinel-first" {
		t.Fatal("应选中匹配网关的凭证", err)
	}
	t.Setenv("KDL_AGENT_GATEWAY_URL", "https://third.example")
	r, err = Load()
	if err != nil || r.Token != "" {
		t.Fatal("禁止跨网关复用凭证", err)
	}
	t.Setenv("KDL_AGENT_TOKEN", "sentinel-environment")
	r, err = Load()
	if err != nil || r.Token != "sentinel-environment" || r.TokenSource != "environment" {
		t.Fatal("环境注入应优先", err)
	}
	if err := Logout(first); err != nil {
		t.Fatal(err)
	}
	credentialPath, _ := CredentialsPath()
	raw, _ := os.ReadFile(credentialPath)
	if strings.Contains(string(raw), "sentinel-first") || strings.Contains(string(raw), "sentinel-environment") || !strings.Contains(string(raw), "sentinel-second") {
		t.Fatal("退出或注入持久化边界错误")
	}
	raw, _ = os.ReadFile(path)
	if strings.Contains(string(raw), "sentinel") {
		t.Fatal("普通配置不能包含凭证")
	}
}

func TestExplicitConfigAndLegacyRejected(t *testing.T) {
	home := isolatedHome(t)
	t.Setenv("KDL_AGENT_CONFIG", filepath.Join(t.TempDir(), "environment.toml"))
	flagPath := filepath.Join(t.TempDir(), "flag.toml")
	SetPathOverride(flagPath)
	if err := SaveLogin(flagPath, DefaultGatewayURL, "sentinel"); err != nil {
		t.Fatal(err)
	}
	r, err := Load()
	if err != nil || r.ConfigPath != flagPath || r.CredentialsPath != filepath.Join(home, ".kdl", "credentials.toml") {
		t.Fatal("显式配置不能改变凭证位置", err)
	}
	if err := os.WriteFile(flagPath, []byte("token = 'sentinel-legacy'\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = Load()
	if err == nil || strings.Contains(err.Error(), "sentinel-legacy") {
		t.Fatal("旧格式应拒绝且不泄露原文")
	}
}

func TestFilePermissionsAndSymlinks(t *testing.T) {
	home := isolatedHome(t)
	path, _ := ResolvePath()
	if err := SaveLogin(path, DefaultGatewayURL, "sentinel"); err != nil {
		t.Fatal(err)
	}
	credentialPath, _ := CredentialsPath()
	if runtime.GOOS != "windows" {
		for _, p := range []string{filepath.Join(home, ".kdl"), credentialPath} {
			if err := os.Chmod(p, 0o777); err != nil {
				t.Fatal(err)
			}
		}
		if err := SaveLogin(path, DefaultGatewayURL, "sentinel-updated"); err != nil {
			t.Fatal(err)
		}
		for p, mode := range map[string]os.FileMode{filepath.Join(home, ".kdl"): 0o700, credentialPath: 0o600, path: 0o600} {
			info, err := os.Stat(p)
			if err != nil || info.Mode().Perm() != mode {
				t.Fatalf("权限错误 %s", p)
			}
		}
	}
	link := filepath.Join(t.TempDir(), "linked.toml")
	if err := os.Symlink(credentialPath, link); err != nil {
		t.Skip("当前平台无法创建测试符号链接")
	}
	if err := SaveLogin(link, DefaultGatewayURL, "must-not-save"); err == nil {
		t.Fatal("不能写入符号链接")
	}
	raw, _ := os.ReadFile(credentialPath)
	if strings.Contains(string(raw), "must-not-save") {
		t.Fatal("失败保存不应修改凭证")
	}
}

func TestRollbackWhenConfigCannotBeWritten(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX 只读目录故障注入；Windows 保存失败需平台验收")
	}
	isolatedHome(t)
	path, _ := ResolvePath()
	if err := SaveLogin(path, DefaultGatewayURL, "sentinel-original"); err != nil {
		t.Fatal(err)
	}
	// 配置可读取但父目录不可写，在凭证更新后触发配置原子替换失败。
	blockedDir := t.TempDir()
	if err := os.Chmod(blockedDir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(blockedDir, 0o700) })
	if err := SaveLogin(filepath.Join(blockedDir, "config.toml"), DefaultGatewayURL, "sentinel-new"); err == nil {
		t.Fatal("应保存失败")
	}
	r, err := Load()
	if err != nil || r.Token != "sentinel-original" {
		t.Fatal("保存失败必须保留原值", err)
	}
}

func TestCredentialDirectorySymlinkRejected(t *testing.T) {
	home := isolatedHome(t)
	if err := os.Symlink(t.TempDir(), filepath.Join(home, ".kdl")); err != nil {
		t.Skip("当前平台无法创建测试符号链接")
	}
	if _, err := Load(); err == nil {
		t.Fatal("不能从符号链接目录加载凭证")
	}
	path, _ := ResolvePath()
	if err := SaveLogin(path, DefaultGatewayURL, "sentinel"); err == nil {
		t.Fatal("不能向符号链接目录写入凭证")
	}
}

func TestConcurrentUpdateRefusesLockedStore(t *testing.T) {
	home := isolatedHome(t)
	path, _ := ResolvePath()
	if err := SaveLogin(path, DefaultGatewayURL, "sentinel-original"); err != nil {
		t.Fatal(err)
	}
	lock := flock.New(filepath.Join(home, ".kdl", ".credentials.lock"))
	if err := lock.Lock(); err != nil {
		t.Fatal(err)
	}
	defer lock.Close()
	if err := Logout(DefaultGatewayURL); err == nil {
		t.Fatal("并发更新必须拒绝")
	}
	r, err := Load()
	if err != nil || r.Token != "sentinel-original" {
		t.Fatal("冲突不能丢失凭证")
	}
}

func TestNormalizeGateway(t *testing.T) {
	for _, raw := range []string{"https://user:sentinel@example.com", "http://example.com", "https://example.com/v1", "https://example.com?token=sentinel", "https://example.com/#fragment", "https://", "file:///tmp/config"} {
		if _, err := NormalizeGateway(raw); err == nil || strings.Contains(err.Error(), "sentinel") {
			t.Fatal("应拒绝非法地址且不回显输入")
		}
	}
	for raw, want := range map[string]string{"https://EXAMPLE.com:443/": "https://example.com", "http://127.0.0.1:8080/": "http://127.0.0.1:8080", "https://[::1]:443": "https://[::1]"} {
		got, err := NormalizeGateway(raw)
		if err != nil || got != want {
			t.Fatalf("规范化失败: %s", raw)
		}
	}
}
