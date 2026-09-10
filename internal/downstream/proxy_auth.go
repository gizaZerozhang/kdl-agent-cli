package downstream

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gizaZerozhang/kdl-agent-cli/internal/config"
)

// ProxyAuthorization 仅用于代理连接，credentials 为敏感的 Basic 编码值。
type ProxyAuthorization struct {
	Type        string `json:"type"`
	Credentials string `json:"credentials"`
}

// FetchProxyAuthorization 获取代理认证；错误不携带签名 URL 或服务端原文。
func FetchProxyAuthorization(ctx context.Context, apiDomain, secretID, secretKey string) (ProxyAuthorization, error) {
	var result ProxyAuthorization
	base, err := resolveAPIBase(apiDomain)
	if err != nil {
		return result, fmt.Errorf("订单 API 地址不可用")
	}
	base, err = config.NormalizeGateway(base)
	if err != nil {
		return result, fmt.Errorf("订单 API 必须使用 HTTPS 根地址；仅本机测试允许 HTTP")
	}
	const path = "/api/getproxyauthorization/"
	params := authParams(secretID, secretKey, http.MethodGet, path, nil)
	endpoint, _ := url.Parse(base + path)
	endpoint.RawQuery = params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return result, fmt.Errorf("无法创建代理认证请求")
	}
	transport := &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := transport.Do(req)
	if err != nil {
		return result, fmt.Errorf("获取代理认证失败，请检查订单 API 网络与证书")
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return result, fmt.Errorf("获取代理认证失败 HTTP %d", resp.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, (1<<20)+1))
	if err != nil || len(raw) > 1<<20 {
		return result, fmt.Errorf("代理认证响应读取失败或过大")
	}
	var envelope struct {
		Code *int               `json:"code"`
		Data ProxyAuthorization `json:"data"`
	}
	if json.Unmarshal(raw, &envelope) != nil || envelope.Code == nil {
		return result, fmt.Errorf("代理认证响应格式无效")
	}
	if *envelope.Code != 0 {
		return result, fmt.Errorf("获取代理认证失败 code=%d，请核对订单能力与状态", *envelope.Code)
	}
	result = envelope.Data
	decoded, err := base64.StdEncoding.DecodeString(result.Credentials)
	defer func() {
		for i := range decoded {
			decoded[i] = 0
		}
	}()
	if result.Type != "Basic" || err != nil || !strings.Contains(string(decoded), ":") || strings.ContainsAny(result.Credentials, "\r\n\t ") || strings.ContainsAny(string(decoded), "\r\n") {
		return ProxyAuthorization{}, fmt.Errorf("代理 Basic 认证字段无效")
	}
	return result, nil
}
