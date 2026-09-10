package cmdutil

import (
	"errors"
	"strings"

	"github.com/gizaZerozhang/kdl-agent-cli/internal/client"
	"github.com/gizaZerozhang/kdl-agent-cli/internal/config"
	"github.com/gizaZerozhang/kdl-agent-cli/internal/output"
)

const (
	ExitOK      = 0
	ExitRuntime = 1
	ExitConfig  = 2
)

// Factory 命令运行时依赖。
type Factory struct {
	Out *output.Writer
}

// NewFactory 构造默认工厂。
func NewFactory(format output.Mode) *Factory {
	return &Factory{Out: output.NewWriter(format)}
}

// LoadClient 加载配置并构造 Gateway 客户端。
func (f *Factory) LoadClient() (*client.Client, config.Resolved, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, cfg, err
	}
	if err := config.ValidateForAPI(cfg); err != nil {
		return nil, cfg, err
	}
	c, err := client.New(cfg)
	if err != nil {
		return nil, cfg, err
	}
	return c, cfg, nil
}

// ExitCodeFromError 映射错误到进程退出码。
func ExitCodeFromError(err error) int {
	if err == nil {
		return ExitOK
	}
	if isConfigLike(err) {
		return ExitConfig
	}
	var apiErr *client.APIError
	if errors.As(err, &apiErr) {
		return ExitRuntime
	}
	return ExitRuntime
}

func isConfigLike(err error) bool {
	msg := err.Error()
	for _, sub := range []string{"未配置", "配置文件", "需要 --", "缺少", "解析配置文件", "unknown command", "unknown flag", "requires", "accepts"} {
		if strings.Contains(msg, sub) {
			return true
		}
	}
	return false
}

// HandleAPIError 打印 API 错误（保留供扩展）。
func (f *Factory) HandleAPIError(err error) int {
	f.Out.PrintError(err)
	if isConfigLike(err) {
		return ExitConfig
	}
	return ExitRuntime
}
