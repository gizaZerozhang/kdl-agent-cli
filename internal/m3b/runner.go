// Package m3b 编排 Secret 获取 + 下游 OpenAPI 的便利命令（修订 19）。
package m3b

import (
	"context"
	"fmt"

	"github.com/kuaidaili/kdl-agent-cli/internal/client"
	"github.com/kuaidaili/kdl-agent-cli/internal/downstream"
	"github.com/kuaidaili/kdl-agent-cli/internal/secret"
)

// ProxyFetchParams 代理提取命令参数。
type ProxyFetchParams struct {
	OrderID string
	Num     int
	Format  string
}

// ProxyFetchResult 命令输出摘要。
type ProxyFetchResult struct {
	ProxyCount int
	Proxies    []string
}

// RunProxyFetch 执行 order secret + getdps 下游调用。
func RunProxyFetch(ctx context.Context, c *client.Client, p ProxyFetchParams) (ProxyFetchResult, error) {
	if p.Num < 1 {
		p.Num = 1
	}
	sec, err := secret.FetchOrderSecret(ctx, c, p.OrderID)
	if err != nil {
		return ProxyFetchResult{}, err
	}
	fetchResult, err := downstream.FetchDPS(sec.APIDomain, sec.SecretID, sec.SecretKey, p.Num)
	if err != nil {
		return ProxyFetchResult{}, err
	}
	return ProxyFetchResult{
		ProxyCount: fetchResult.ProxyCount,
		Proxies:    fetchResult.Proxies,
	}, nil
}

// WhitelistParams 白名单写命令参数。
type WhitelistParams struct {
	OrderID string
	Action  string
	IPs     []string
	Clear   bool
}

// WhitelistResult 命令输出摘要。
type WhitelistResult struct {
	WhitelistIPCount int
}

// RunWhitelistWrite 执行 order secret + setipwhitelist 等下游调用。
func RunWhitelistWrite(ctx context.Context, c *client.Client, p WhitelistParams) (WhitelistResult, error) {
	action := p.Action
	if action == "" {
		if p.Clear {
			action = "clear"
		} else {
			action = "set"
		}
	}
	sec, err := secret.FetchOrderSecret(ctx, c, p.OrderID)
	if err != nil {
		return WhitelistResult{}, err
	}
	var wl downstream.WhitelistSetResult
	switch action {
	case "clear":
		wl, err = downstream.ClearWhitelist(sec.APIDomain, sec.SecretID, sec.SecretKey)
	case "add":
		wl, err = downstream.AddWhitelistIPs(sec.APIDomain, sec.SecretID, sec.SecretKey, p.IPs)
	case "delete":
		wl, err = downstream.DeleteWhitelistIPs(sec.APIDomain, sec.SecretID, sec.SecretKey, p.IPs)
	default:
		wl, err = downstream.SetWhitelist(sec.APIDomain, sec.SecretID, sec.SecretKey, p.IPs)
	}
	if err != nil {
		return WhitelistResult{}, fmt.Errorf("白名单下游调用失败: %w", err)
	}
	return WhitelistResult{WhitelistIPCount: wl.WhitelistIPCount}, nil
}
