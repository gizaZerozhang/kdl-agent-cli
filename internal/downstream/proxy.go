// Package downstream 封装订单公开 API：代理提取可 Mock，白名单默认真实 DB 读写。
package downstream

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
)

const envDownstreamMock = "KDL_AGENT_DOWNSTREAM_MOCK"

// ProxyResult 裁剪后的代理提取结果（不含 Secret）。
type ProxyResult struct {
	ProxyCount int      `json:"proxy_count"`
	Proxies    []string `json:"proxies,omitempty"`
}

// WhitelistSetResult 白名单写操作摘要。
type WhitelistSetResult struct {
	WhitelistIPCount int `json:"whitelist_ip_count"`
}

// FetchDPS 调用 getdps 公开 API；Mock 模式返回 fixture 代理。
func FetchDPS(apiDomain, secretID, secretKey string, num int) (ProxyResult, error) {
	if mockEnabled() {
		proxies := make([]string, 0, num)
		for i := 0; i < num; i++ {
			proxies = append(proxies, fmt.Sprintf("127.0.0.1:%d", 10000+i))
		}
		return ProxyResult{ProxyCount: num, Proxies: proxies}, nil
	}
	if num < 1 || num > 200 {
		return ProxyResult{}, fmt.Errorf("num 须在 1–200 之间")
	}
	base, err := resolveAPIBase(apiDomain)
	if err != nil {
		return ProxyResult{}, err
	}
	const path = "/api/getdps/"
	extra := url.Values{}
	extra.Set("num", strconv.Itoa(num))
	extra.Set("format", "json")
	params := authParams(secretID, secretKey, http.MethodGet, path, extra)
	resp, err := callOpenAPIGet(base, path, params)
	if err != nil {
		return ProxyResult{}, err
	}
	if resp.Code != 0 {
		return ProxyResult{}, fmt.Errorf("getdps 失败 code=%d msg=%s", resp.Code, resp.Msg)
	}
	proxies, count, err := parseGetDPSData(resp.Data)
	if err != nil {
		return ProxyResult{}, fmt.Errorf("解析 getdps 响应失败: %w", err)
	}
	if count == 0 {
		count = len(proxies)
	}
	return ProxyResult{ProxyCount: count, Proxies: proxies}, nil
}

// parseGetDPSData 解析 getdps data；兼容 proxy_list 为 "ip:port" 字符串或 {ip,port} 对象。
func parseGetDPSData(raw json.RawMessage) ([]string, int, error) {
	if len(raw) == 0 {
		return nil, 0, nil
	}
	var data struct {
		Count     int             `json:"count"`
		ProxyList json.RawMessage `json:"proxy_list"`
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, 0, err
	}
	proxies, err := parseProxyListItems(data.ProxyList)
	if err != nil {
		return nil, 0, err
	}
	return proxies, data.Count, nil
}

func parseProxyListItems(raw json.RawMessage) ([]string, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var asStrings []string
	if err := json.Unmarshal(raw, &asStrings); err == nil {
		return normalizeProxyEndpoints(asStrings), nil
	}
	var asObjects []struct {
		IP   string `json:"ip"`
		Port int    `json:"port"`
	}
	if err := json.Unmarshal(raw, &asObjects); err != nil {
		return nil, err
	}
	proxies := make([]string, 0, len(asObjects))
	for _, item := range asObjects {
		if item.IP == "" || item.Port <= 0 {
			continue
		}
		proxies = append(proxies, fmt.Sprintf("%s:%d", item.IP, item.Port))
	}
	return proxies, nil
}

func normalizeProxyEndpoints(items []string) []string {
	proxies := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		proxies = append(proxies, item)
	}
	return proxies
}

func mockEnabled() bool {
	v := strings.TrimSpace(os.Getenv(envDownstreamMock))
	return v == "1" || strings.EqualFold(v, "true") || strings.EqualFold(v, "yes")
}

// FormatProxyJSON 供 complete 回执 result_summary 序列化。
func FormatProxyJSON(r ProxyResult) map[string]any {
	return map[string]any{"proxy_count": r.ProxyCount}
}

// FormatWhitelistJSON 供 complete 回执 result_summary 序列化。
func FormatWhitelistJSON(r WhitelistSetResult) map[string]any {
	return map[string]any{"whitelist_ip_count": r.WhitelistIPCount}
}

// MustJSON 调试辅助。
func MustJSON(v any) string {
	raw, _ := json.Marshal(v)
	return string(raw)
}
