package cmd_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWhitelistReadDoesNotWriteOrExposeSecret(t *testing.T) {
	authHome(t)
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/getipwhitelist/" || r.Method != "GET" || r.URL.Query().Get("signature") == "" || r.Header.Get("Authorization") != "" {
			t.Error("读取路径或鉴权错误")
		}
		_, _ = w.Write([]byte(`{"code":0,"data":{"ipwhitelist":["192.0.2.10"],"count":1}}`))
	}))
	defer api.Close()
	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/orders/123/secret" {
			t.Error("Gateway 路径错误")
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "data": map[string]string{"secret_id": "test-id", "secret_key": "key-sentinel", "api_domain": api.URL}})
	}))
	defer gateway.Close()
	t.Setenv("KDL_AGENT_GATEWAY_URL", gateway.URL)
	t.Setenv("KDL_AGENT_TOKEN", "agent-sentinel")
	t.Setenv("KDL_AGENT_ORDER_API_BASE_URL", "")
	out, diagnostic, err := runAuth([]string{"order", "whitelist", "get", "--order", "123", "--format", "json"}, "")
	if err != nil || !strings.Contains(out, "192.0.2.10") || strings.Contains(out+diagnostic, "sentinel") {
		t.Fatalf("白名单读取失败: %v", err)
	}
	var data struct {
		Count int      `json:"count"`
		IPs   []string `json:"ipwhitelist"`
	}
	if json.Unmarshal([]byte(out), &data) != nil || data.Count != 1 || len(data.IPs) != 1 {
		t.Fatal("结果结构错误")
	}
}
