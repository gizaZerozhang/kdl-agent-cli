// Package downstream 封装订单公开 API 的签名、调用与结果校验。
package downstream

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

const envOrderAPIBase = "KDL_AGENT_ORDER_API_BASE_URL"

type openAPIResponse struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

type whitelistData struct {
	IPWhitelist []string `json:"ipwhitelist"`
	Count       int      `json:"count"`
}

// resolveAPIBase 解析订单公开 API 根 URL；本地联调可用环境变量覆盖 envelope 中的 api_domain。
func resolveAPIBase(apiDomain string) (string, error) {
	if override := strings.TrimSpace(os.Getenv(envOrderAPIBase)); override != "" {
		return strings.TrimRight(override, "/"), nil
	}
	base := strings.TrimRight(strings.TrimSpace(apiDomain), "/")
	if base == "" {
		return "", fmt.Errorf("api_domain 为空；本地请设置 %s", envOrderAPIBase)
	}
	return base, nil
}

// authParams 构造 HmacSHA1 鉴权参数；白名单写接口不使用 simple 签名。
func authParams(secretID, secretKey, method, path string, extra url.Values) url.Values {
	values := url.Values{}
	values.Set("secret_id", secretID)
	values.Set("sign_type", "hmacsha1")
	values.Set("timestamp", strconv.FormatInt(time.Now().Unix(), 10))
	values.Set("nonce", strconv.FormatInt(time.Now().UnixNano(), 10))
	for k, vs := range extra {
		for _, v := range vs {
			values.Set(k, v)
		}
	}
	values.Set("signature", computeOpenAPISignature(method, path, values, secretKey))
	return values
}

func computeOpenAPISignature(method, path string, params url.Values, secretKey string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		if k == "signature" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+params.Get(k))
	}
	stringToSign := method + path + "?" + strings.Join(parts, "&")
	mac := hmac.New(sha1.New, []byte(secretKey))
	_, _ = mac.Write([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func callOpenAPIGet(baseURL, path string, params url.Values) (openAPIResponse, error) {
	endpoint, err := url.Parse(baseURL + path)
	if err != nil {
		return openAPIResponse{}, fmt.Errorf("解析 API URL 失败: %w", err)
	}
	endpoint.RawQuery = params.Encode()

	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Get(endpoint.String())
	if err != nil {
		return openAPIResponse{}, fmt.Errorf("调用订单公开 API 失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return openAPIResponse{}, fmt.Errorf("订单公开 API HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return openAPIResponse{}, fmt.Errorf("读取下游响应失败: %w", err)
	}
	var parsed openAPIResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return openAPIResponse{}, fmt.Errorf("解析下游 JSON 失败: %w", err)
	}
	var required struct {
		Code *int `json:"code"`
	}
	if json.Unmarshal(body, &required) != nil || required.Code == nil {
		return openAPIResponse{}, fmt.Errorf("下游响应缺少 code")
	}
	return parsed, nil
}

func readWhitelist(baseURL, secretID, secretKey string) (whitelistData, error) {
	const path = "/api/getipwhitelist/"
	params := authParams(secretID, secretKey, http.MethodGet, path, nil)
	resp, err := callOpenAPIGet(baseURL, path, params)
	if err != nil {
		return whitelistData{}, err
	}
	if resp.Code != 0 {
		return whitelistData{}, fmt.Errorf("getipwhitelist 失败 code=%d msg=%s", resp.Code, resp.Msg)
	}
	var data whitelistData
	if len(resp.Data) == 0 || string(resp.Data) == "null" {
		return whitelistData{}, fmt.Errorf("白名单响应缺少 data")
	}
	if len(resp.Data) > 0 {
		if err := json.Unmarshal(resp.Data, &data); err != nil {
			return whitelistData{}, fmt.Errorf("解析白名单回读失败: %w", err)
		}
	}
	var required struct {
		IPs   *[]string `json:"ipwhitelist"`
		Count *int      `json:"count"`
	}
	if json.Unmarshal(resp.Data, &required) != nil || required.IPs == nil || required.Count == nil || *required.Count < 0 {
		return whitelistData{}, fmt.Errorf("白名单响应缺少有效列表或数量")
	}
	return data, nil
}

func verifyWhitelistAfterWrite(baseURL, secretID, secretKey string, expectCount int, expectIPs []string) (WhitelistSetResult, error) {
	data, err := readWhitelist(baseURL, secretID, secretKey)
	if err != nil {
		return WhitelistSetResult{}, err
	}
	if data.Count != expectCount {
		return WhitelistSetResult{}, fmt.Errorf(
			"白名单回读数量不一致: 期望 %d 实际 %d",
			expectCount, data.Count,
		)
	}
	if expectIPs != nil {
		have := make(map[string]struct{}, len(data.IPWhitelist))
		for _, ip := range data.IPWhitelist {
			have[strings.TrimSpace(ip)] = struct{}{}
		}
		for _, want := range expectIPs {
			if _, ok := have[strings.TrimSpace(want)]; !ok {
				return WhitelistSetResult{}, fmt.Errorf("白名单回读缺少 IP: %s", want)
			}
		}
	}
	return WhitelistSetResult{WhitelistIPCount: data.Count}, nil
}
