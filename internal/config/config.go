// Package config 负责 CLI 配置加载、网关凭证绑定与持久化。
package config

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

const DefaultFilename = "config.toml"
const DefaultGatewayURL = "https://agent-gateway.kdlapi.com"

var pathOverride string

// File 只保存非敏感配置。SchemaVersion 用于拒绝静默误读。
type File struct {
	SchemaVersion int    `toml:"schema_version"`
	GatewayURL    string `toml:"gateway_url"`
}

type credentials struct {
	SchemaVersion int               `toml:"schema_version"`
	Gateways      map[string]string `toml:"gateways"`
}

// Resolved 不得整体输出或记录日志，Token 仅供 HTTP 客户端使用。
type Resolved struct {
	GatewayURL      string
	Token           string
	ConfigPath      string
	CredentialsPath string
	ConfigSource    string
	TokenSource     string
}

func SetPathOverride(path string) { pathOverride = strings.TrimSpace(path) }

// ResolvePath 的显式相对路径相对 CWD；默认位置与 CWD 无关。
func ResolvePath() (string, error) {
	if pathOverride != "" {
		return filepath.Abs(pathOverride)
	}
	if path := strings.TrimSpace(os.Getenv("KDL_AGENT_CONFIG")); path != "" {
		return filepath.Abs(path)
	}
	dir, err := homeDir()
	return filepath.Join(dir, DefaultFilename), err
}

func homeDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil || !filepath.IsAbs(home) {
		return "", errors.New("无法定位用户家目录；请检查操作系统用户配置")
	}
	return filepath.Join(home, ".kdl"), nil
}

func CredentialsPath() (string, error) {
	dir, err := homeDir()
	return filepath.Join(dir, "credentials.toml"), err
}

// NormalizeGateway 固定凭证绑定键，只允许 HTTPS 或本机回环 HTTP。
func NormalizeGateway(raw string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.Opaque != "" || (u.Path != "" && u.Path != "/") {
		return "", errors.New("Gateway 地址无效；请使用不含用户信息、路径、查询或片段的 HTTPS 基址")
	}
	u.Scheme = strings.ToLower(u.Scheme)
	host := strings.ToLower(u.Hostname())
	if u.Scheme != "https" {
		ip := net.ParseIP(host)
		if u.Scheme != "http" || (host != "localhost" && (ip == nil || !ip.IsLoopback())) {
			return "", errors.New("Gateway 必须使用 HTTPS；仅本机回环测试允许 HTTP")
		}
	}
	port := u.Port()
	if (u.Scheme == "https" && port == "443") || (u.Scheme == "http" && port == "80") {
		port = ""
	}
	u.Host = host
	if strings.Contains(host, ":") {
		u.Host = "[" + host + "]"
	}
	if port != "" {
		u.Host = net.JoinHostPort(host, port)
	}
	u.Path, u.RawPath = "", ""
	return u.String(), nil
}

func readTOML(path string, value any) error {
	if err := regularFile(path); err != nil {
		return err
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("读取配置文件 %s 失败；请检查访问权限", path)
	}
	// 解析器错误可能含配置原文，禁止将其拼入用户诊断。
	if err := toml.NewDecoder(bytes.NewReader(raw)).DisallowUnknownFields().Decode(value); err != nil {
		return fmt.Errorf("配置文件 %s 格式或字段不兼容；请检查 TOML，旧 token/device 配置请按迁移说明重新登录", path)
	}
	return nil
}

func readConfig(path string) (File, error) {
	var file File
	if err := readTOML(path, &file); err != nil {
		return file, err
	}
	if file.SchemaVersion != 0 && file.SchemaVersion != 1 {
		return file, fmt.Errorf("配置文件 %s 版本不兼容；请使用匹配版本的 CLI", path)
	}
	return file, nil
}

func readCredentials(path string) (credentials, error) {
	value := credentials{SchemaVersion: 1, Gateways: make(map[string]string)}
	if info, err := os.Lstat(filepath.Dir(path)); err == nil && (!info.IsDir() || info.Mode()&os.ModeSymlink != 0) {
		return value, fmt.Errorf("凭证配置目录 %s 不能是符号链接；请恢复真实目录后重试", filepath.Dir(path))
	}
	if err := readTOML(path, &value); err != nil {
		return value, err
	}
	if value.SchemaVersion != 1 {
		return value, fmt.Errorf("凭证配置文件 %s 版本不兼容；请使用匹配版本的 CLI", path)
	}
	return value, nil
}

func Load() (Resolved, error) {
	path, err := ResolvePath()
	if err != nil {
		return Resolved{}, err
	}
	credentialPath, err := CredentialsPath()
	if err != nil {
		return Resolved{}, err
	}
	file, err := readConfig(path)
	if err != nil {
		return Resolved{}, err
	}
	r := Resolved{ConfigPath: path, CredentialsPath: credentialPath, GatewayURL: DefaultGatewayURL, ConfigSource: "default", TokenSource: "none"}
	if file.GatewayURL != "" {
		r.GatewayURL, r.ConfigSource = file.GatewayURL, "config"
	}
	if value := strings.TrimSpace(os.Getenv("KDL_AGENT_GATEWAY_URL")); value != "" {
		r.GatewayURL, r.ConfigSource = value, "environment"
	}
	r.GatewayURL, err = NormalizeGateway(r.GatewayURL)
	if err != nil {
		return r, err
	}
	if value := strings.TrimSpace(os.Getenv("KDL_AGENT_TOKEN")); value != "" {
		r.Token, r.TokenSource = value, "environment"
		return r, nil
	}
	value, err := readCredentials(credentialPath)
	if err != nil {
		return r, err
	}
	r.Token = value.Gateways[r.GatewayURL]
	if r.Token != "" {
		r.TokenSource = "credentials"
	}
	return r, nil
}

// Save 仅保存普通配置，永不保存运行时 token。
func Save(path string, file File) error {
	gateway, err := NormalizeGateway(file.GatewayURL)
	if err != nil {
		return err
	}
	file.SchemaVersion, file.GatewayURL = 1, gateway
	raw, err := toml.Marshal(file)
	if err != nil {
		return errors.New("序列化配置失败；请检查配置字段")
	}
	return atomicWrite(path, raw)
}

func ValidateForAPI(r Resolved) error {
	if _, err := NormalizeGateway(r.GatewayURL); err != nil {
		return err
	}
	if strings.TrimSpace(r.Token) == "" {
		return fmt.Errorf("未配置当前 Gateway 的 Agent 凭证\n  配置文件: %s\n  凭证文件: %s\n  修复: kdl-agent auth login，或在运行环境注入 KDL_AGENT_TOKEN", r.ConfigPath, r.CredentialsPath)
	}
	return nil
}
