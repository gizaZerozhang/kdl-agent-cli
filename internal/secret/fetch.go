// Package secret 调用 Gateway 订单 Secret 直读 API（修订 19）。
package secret

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gizaZerozhang/kdl-agent-cli/internal/client"
)

// OrderSecret 为 Gateway POST /v1/orders/{id}/secret 的稳定响应字段。
type OrderSecret struct {
	OrderID   string `json:"order_id"`
	SecretID  string `json:"secret_id"`
	SecretKey string `json:"secret_key"`
	APIDomain string `json:"api_domain"`
}

// FetchOrderSecret 获取订单 Secret；响应不含 execution/device 编排。
func FetchOrderSecret(ctx context.Context, c *client.Client, orderID string) (OrderSecret, error) {
	path := fmt.Sprintf("/v1/orders/%s/secret", orderID)
	resp, err := c.PostJSONWithHeaders(ctx, path, map[string]any{}, nil)
	if err != nil {
		return OrderSecret{}, err
	}
	var out OrderSecret
	if err := json.Unmarshal(resp.Data, &out); err != nil {
		return OrderSecret{}, fmt.Errorf("解析 Secret 响应失败: %w", err)
	}
	return out, nil
}
