package cmd_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kuaidaili/kdl-agent-cli/cmd"
)

func TestRateLimitWaitHint(t *testing.T) {
	authHome(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "7")
		w.WriteHeader(429)
		_, _ = w.Write([]byte(`{"success":false,"error":{"code":"RATE_LIMITED","message":"limited"}}`))
	}))
	defer srv.Close()
	t.Setenv("KDL_AGENT_GATEWAY_URL", srv.URL)
	t.Setenv("KDL_AGENT_TOKEN", "test-token")
	out, diagnostic, err := runAuth([]string{"account", "summary", "--format", "json"}, "")
	if err == nil || !strings.Contains(diagnostic, "7 秒") || !json.Valid([]byte(out)) {
		t.Fatal("限流提示或 JSON 错误结果缺失")
	}
}

func TestQuoteValidationErrorJSONStdout(t *testing.T) {
	authHome(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success":    false,
			"request_id": "req_validation",
			"error": map[string]any{
				"code":    "VALIDATION_ERROR",
				"message": "产品配置无效或不可售",
				"details": map[string]any{
					"validation_errors": []string{"configuration.tps_reqrate_num 低于最小值 5"},
				},
			},
		})
	}))
	defer srv.Close()
	t.Setenv("KDL_AGENT_GATEWAY_URL", srv.URL)
	t.Setenv("KDL_AGENT_TOKEN", "test-token")

	bodyFile := filepath.Join(t.TempDir(), "quote.json")
	if err := os.WriteFile(bodyFile, []byte(`{"product_type":"tps_pro","spec_version":"","configuration":{"product":"month","quantity":1,"tps_reqrate_num":4}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	root := cmd.NewRootCmd()
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs([]string{"--format", "json", "product", "quote", "--body-file", bodyFile})
	if err := root.Execute(); err == nil {
		t.Fatal("expected validation error")
	}
	if stdout.Len() == 0 {
		t.Fatalf("json mode should print error envelope to stdout: stderr=%q", stderr.String())
	}
	var envelope map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("stdout must be json: %q", stdout.String())
	}
	if envelope["success"] != false {
		t.Fatalf("envelope: %+v", envelope)
	}
	if !strings.Contains(stderr.String(), "configuration.tps_reqrate_num") {
		t.Fatalf("stderr missing field detail: %q", stderr.String())
	}
}

func TestGatewayErrorsAcrossCommandPaths(t *testing.T) {
	for _, args := range [][]string{
		{"account", "summary"},
		{"order", "secret", "get", "--order", "123"},
		{"proxy", "fetch", "--order", "123", "--num", "1"},
		{"proxy", "auth", "--order", "123"},
		{"order", "whitelist", "set", "--order", "123", "--ip", "192.0.2.10"},
		{"order", "whitelist", "clear", "--order", "123"},
		{"order", "whitelist", "get", "--order", "123"},
	} {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			authHome(t)
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"success":false,"request_id":"req_auth","error":{"code":"CREDENTIAL_REVOKED","message":"凭证已撤销"}}`))
			}))
			defer srv.Close()
			t.Setenv("KDL_AGENT_GATEWAY_URL", srv.URL)
			t.Setenv("KDL_AGENT_TOKEN", "sentinel")
			out, diagnostic, err := runAuth(append([]string{"--format", "json"}, args...), "")
			if err == nil {
				t.Fatal("应返回失败退出状态")
			}
			var envelope map[string]any
			if err := json.Unmarshal([]byte(out), &envelope); err != nil {
				t.Fatalf("错误输出不是 JSON: %q", out)
			}
			if envelope["success"] != false || envelope["request_id"] != "req_auth" {
				t.Fatalf("错误信封丢失: %v", envelope)
			}
			if !strings.Contains(diagnostic, "kdl-agent auth login") {
				t.Fatalf("缺少登录建议: %s", diagnostic)
			}
		})
	}
}
