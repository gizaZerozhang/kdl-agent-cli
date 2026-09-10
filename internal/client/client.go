// Package client 封装对 Gateway /v1/ 的 HTTP 调用。
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gizaZerozhang/kdl-agent-cli/internal/config"
)

// GatewayResponse 与 server SuccessEnvelope / ErrorEnvelope 对齐。
type GatewayResponse struct {
	Success   bool            `json:"success"`
	Data      json.RawMessage `json:"data"`
	RequestID string          `json:"request_id"`
	Error     *GatewayError   `json:"error"`
}

// GatewayError 结构化错误体。
type GatewayError struct {
	Code    string          `json:"code"`
	Message string          `json:"message"`
	Details json.RawMessage `json:"details,omitempty"`
}

// APIError 供 CLI 层映射退出码。
type APIError struct {
	StatusCode int
	Code       string
	Message    string
	RequestID  string
}

func (e *APIError) Error() string {
	if e.RequestID != "" {
		return fmt.Sprintf("%s: %s (request_id=%s)", e.Code, e.Message, e.RequestID)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Client Gateway HTTP 客户端。
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// New 从已解析配置构造客户端。
func New(cfg config.Resolved) (*Client, error) {
	base, err := config.NormalizeGateway(cfg.GatewayURL)
	if err != nil {
		return nil, err
	}
	return &Client{
		baseURL: base,
		token:   strings.TrimSpace(cfg.Token),
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}, nil
}

// Get 发起 GET 并解析 JSON 信封。
func (c *Client) Get(ctx context.Context, path string, query url.Values) (*GatewayResponse, error) {
	return c.do(ctx, http.MethodGet, path, query, nil, nil)
}

// PostJSON 发起 POST（application/json）。
func (c *Client) PostJSON(ctx context.Context, path string, body any) (*GatewayResponse, error) {
	return c.PostJSONWithHeaders(ctx, path, body, nil)
}

// PostJSONWithHeaders 发起 POST，可附加额外请求头（如 Idempotency-Key）。
func (c *Client) PostJSONWithHeaders(
	ctx context.Context,
	path string,
	body any,
	headers map[string]string,
) (*GatewayResponse, error) {
	var payload io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("编码请求体失败: %w", err)
		}
		payload = bytes.NewReader(raw)
	}
	return c.do(ctx, http.MethodPost, path, nil, payload, headers)
}

func (c *Client) do(
	ctx context.Context,
	method, path string,
	query url.Values,
	body io.Reader,
	extraHeaders map[string]string,
) (*GatewayResponse, error) {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	u, err := url.Parse(c.baseURL + path)
	if err != nil {
		return nil, err
	}
	if len(query) > 0 {
		u.RawQuery = query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for key, value := range extraHeaders {
		if strings.TrimSpace(value) != "" {
			req.Header.Set(key, value)
		}
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求 Gateway 失败 (%s): %w", u.String(), err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取 Gateway 响应失败: %w", err)
	}

	// 即使远端错误反射请求凭证，也不能让它进入普通结果或诊断。
	if c.token != "" {
		raw = bytes.ReplaceAll(raw, []byte(c.token), []byte("[REDACTED]"))
	}
	var envelope GatewayResponse
	if err := json.Unmarshal(raw, &envelope); err != nil {
		if resp.StatusCode == http.StatusUnauthorized {
			return nil, &APIError{StatusCode: resp.StatusCode, Code: "AUTH_REQUIRED", Message: "凭证无效、过期或已撤销；请重新登录"}
		}
		return nil, fmt.Errorf("解析 Gateway JSON 失败 (HTTP %d)；请检查 Gateway 版本与响应格式", resp.StatusCode)
	}
	if c.token != "" {
		envelope.RequestID = strings.ReplaceAll(envelope.RequestID, c.token, "[REDACTED]")
		if envelope.Error != nil {
			envelope.Error.Message = strings.ReplaceAll(envelope.Error.Message, c.token, "[REDACTED]")
			envelope.Error.Code = strings.ReplaceAll(envelope.Error.Code, c.token, "[REDACTED]")
		}
	}

	if !envelope.Success {
		code := "UNKNOWN"
		msg := fmt.Sprintf("HTTP %d", resp.StatusCode)
		if envelope.Error != nil {
			if envelope.Error.Code != "" {
				code = envelope.Error.Code
			}
			if envelope.Error.Message != "" {
				msg = envelope.Error.Message
			}
		}
		return &envelope, &APIError{
			StatusCode: resp.StatusCode,
			Code:       code,
			Message:    msg,
			RequestID:  envelope.RequestID,
		}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &envelope, &APIError{
			StatusCode: resp.StatusCode,
			Code:       "HTTP_ERROR",
			Message:    fmt.Sprintf("意外 HTTP 状态 %d", resp.StatusCode),
			RequestID:  envelope.RequestID,
		}
	}

	return &envelope, nil
}
