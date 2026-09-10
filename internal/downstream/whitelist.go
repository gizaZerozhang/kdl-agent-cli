package downstream

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/gizaZerozhang/kdl-agent-cli/internal/config"
)

const envWhitelistMock = "KDL_AGENT_WHITELIST_MOCK"

// GetWhitelist 只读当前列表；不修改白名单，不暴露签名 URL 或下游原文。
func GetWhitelist(apiDomain, secretID, secretKey string) (whitelistData, error) {
	base, err := resolveAPIBase(apiDomain)
	if err != nil {
		return whitelistData{}, fmt.Errorf("订单 API 地址不可用")
	}
	base, err = config.NormalizeGateway(base)
	if err != nil {
		return whitelistData{}, fmt.Errorf("订单 API 地址必须为 HTTPS；本机测试可用 HTTP")
	}
	data, err := readWhitelist(base, secretID, secretKey)
	if err != nil {
		return whitelistData{}, fmt.Errorf("读取白名单失败，请检查订单权限、状态与网络")
	}
	if data.Count != len(data.IPWhitelist) {
		return whitelistData{}, fmt.Errorf("白名单响应数量不一致")
	}
	if data.IPWhitelist == nil {
		data.IPWhitelist = []string{}
	}
	return data, nil
}

func whitelistMockEnabled() bool {
	v := strings.TrimSpace(os.Getenv(envWhitelistMock))
	if v == "" {
		return false
	}
	return v == "1" || strings.EqualFold(v, "true") || strings.EqualFold(v, "yes")
}

// SetWhitelist 完整替换订单 IP 白名单，并通过公开 API 回读校验。
func SetWhitelist(apiDomain, secretID, secretKey string, ips []string) (WhitelistSetResult, error) {
	if whitelistMockEnabled() {
		return WhitelistSetResult{WhitelistIPCount: len(ips)}, nil
	}
	base, err := resolveAPIBase(apiDomain)
	if err != nil {
		return WhitelistSetResult{}, err
	}
	if len(ips) == 0 {
		return WhitelistSetResult{}, fmt.Errorf("set 白名单需要至少 1 个 IP")
	}
	const path = "/api/setipwhitelist/"
	extra := url.Values{}
	extra.Set("iplist", strings.Join(ips, ","))
	params := authParams(secretID, secretKey, http.MethodGet, path, extra)
	resp, err := callOpenAPIGet(base, path, params)
	if err != nil {
		return WhitelistSetResult{}, err
	}
	if resp.Code != 0 {
		return WhitelistSetResult{}, fmt.Errorf("setipwhitelist 失败 code=%d msg=%s", resp.Code, resp.Msg)
	}
	return verifyWhitelistAfterWrite(base, secretID, secretKey, len(ips), ips)
}

// AddWhitelistIPs 追加白名单 IP。
func AddWhitelistIPs(apiDomain, secretID, secretKey string, ips []string) (WhitelistSetResult, error) {
	if whitelistMockEnabled() {
		return WhitelistSetResult{WhitelistIPCount: len(ips)}, nil
	}
	base, err := resolveAPIBase(apiDomain)
	if err != nil {
		return WhitelistSetResult{}, err
	}
	const path = "/api/addwhiteip/"
	extra := url.Values{}
	extra.Set("iplist", strings.Join(ips, ","))
	params := authParams(secretID, secretKey, http.MethodGet, path, extra)
	resp, err := callOpenAPIGet(base, path, params)
	if err != nil {
		return WhitelistSetResult{}, err
	}
	if resp.Code != 0 {
		return WhitelistSetResult{}, fmt.Errorf("addwhiteip 失败 code=%d msg=%s", resp.Code, resp.Msg)
	}
	data, err := readWhitelist(base, secretID, secretKey)
	if err != nil {
		return WhitelistSetResult{}, err
	}
	return WhitelistSetResult{WhitelistIPCount: data.Count}, nil
}

// DeleteWhitelistIPs 删除指定白名单 IP。
func DeleteWhitelistIPs(apiDomain, secretID, secretKey string, ips []string) (WhitelistSetResult, error) {
	if whitelistMockEnabled() {
		return WhitelistSetResult{WhitelistIPCount: 0}, nil
	}
	base, err := resolveAPIBase(apiDomain)
	if err != nil {
		return WhitelistSetResult{}, err
	}
	const path = "/api/delwhiteip/"
	extra := url.Values{}
	extra.Set("iplist", strings.Join(ips, ","))
	params := authParams(secretID, secretKey, http.MethodGet, path, extra)
	resp, err := callOpenAPIGet(base, path, params)
	if err != nil {
		return WhitelistSetResult{}, err
	}
	if resp.Code != 0 {
		return WhitelistSetResult{}, fmt.Errorf("delwhiteip 失败 code=%d msg=%s", resp.Code, resp.Msg)
	}
	data, err := readWhitelist(base, secretID, secretKey)
	if err != nil {
		return WhitelistSetResult{}, err
	}
	return WhitelistSetResult{WhitelistIPCount: data.Count}, nil
}

// ClearWhitelist 清空白名单；显式传空 iplist，避免缺省为调用者 IP。
func ClearWhitelist(apiDomain, secretID, secretKey string) (WhitelistSetResult, error) {
	if whitelistMockEnabled() {
		return WhitelistSetResult{WhitelistIPCount: 0}, nil
	}
	base, err := resolveAPIBase(apiDomain)
	if err != nil {
		return WhitelistSetResult{}, err
	}
	const path = "/api/setipwhitelist/"
	extra := url.Values{}
	extra.Set("iplist", "")
	params := authParams(secretID, secretKey, http.MethodGet, path, extra)
	resp, err := callOpenAPIGet(base, path, params)
	if err != nil {
		return WhitelistSetResult{}, err
	}
	if resp.Code != 0 {
		return WhitelistSetResult{}, fmt.Errorf("清空白名单失败 code=%d msg=%s", resp.Code, resp.Msg)
	}
	return verifyWhitelistAfterWrite(base, secretID, secretKey, 0, nil)
}
