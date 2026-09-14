package cmd_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kuaidaili/kdl-agent-cli/cmd"
)

func authHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	for _, name := range []string{"KDL_AGENT_CONFIG", "KDL_AGENT_GATEWAY_URL", "KDL_AGENT_TOKEN"} {
		t.Setenv(name, "")
	}
	return home
}

func TestAuthStatusMissingGrantsRemainsUnknown(t *testing.T) {
	authHome(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"success":true,"data":{"grants":{"unrecognized":true}},"request_id":"test"}`))
	}))
	defer srv.Close()
	t.Setenv("KDL_AGENT_GATEWAY_URL", srv.URL)
	t.Setenv("KDL_AGENT_TOKEN", "sentinel-valid")
	out, _, err := runAuth([]string{"auth", "status", "--format", "json"}, "")
	if err != nil || !strings.Contains(out, `"grants": {}`) || strings.Contains(out, "unrecognized") {
		t.Fatalf("缺失授权不应推断或透传: %s %v", out, err)
	}
	out, _, err = runAuth([]string{"auth", "status", "--format", "table"}, "")
	if err != nil || strings.Count(out, "未知（Gateway 未返回）") != 3 {
		t.Fatalf("缺失授权应展示未知: %s %v", out, err)
	}
}

func runAuth(args []string, input string) (string, string, error) {
	root := cmd.NewRootCmd()
	out, diagnostic := new(bytes.Buffer), new(bytes.Buffer)
	root.SetOut(out)
	root.SetErr(diagnostic)
	root.SetIn(strings.NewReader(input))
	root.SetArgs(args)
	err := root.Execute()
	return out.String(), diagnostic.String(), err
}

func TestAuthLoginStatusFailureAndLogout(t *testing.T) {
	home := authHome(t)
	valid := true
	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.URL.Path != "/v1/account/summary" {
			t.Error("验证端点错误")
		}
		if !valid {
			w.WriteHeader(401)
			_, _ = w.Write([]byte(`{"success":false,"error":{"code":"CREDENTIAL_REVOKED","message":"sentinel-invalid"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"success":true,"data":{"account_id":1,"grants":{"product.purchase.create":true,"support.ticket.create":false,"order.secret.read":true}},"request_id":"test"}`))
	}))
	defer srv.Close()
	out, diagnostic, err := runAuth([]string{"auth", "login", "--gateway-url", srv.URL, "--token-stdin", "--format", "json"}, "sentinel-valid\n")
	if err != nil || !strings.Contains(out, `"saved": true`) || strings.Contains(out+diagnostic, "sentinel") {
		t.Fatal("登录应验证保存且无凭证输出", err)
	}
	path := filepath.Join(home, ".kdl", "credentials.toml")
	before, _ := os.ReadFile(path)
	valid = false
	out, diagnostic, err = runAuth([]string{"auth", "login", "--token-stdin"}, "sentinel-invalid")
	after, _ := os.ReadFile(path)
	if err == nil || !bytes.Equal(before, after) || strings.Contains(out+diagnostic, "sentinel") {
		t.Fatal("失败必须保留旧值且脱敏")
	}
	out, _, err = runAuth([]string{"auth", "status", "--format", "json"}, "")
	if err == nil || !strings.Contains(out, `"status": "invalid"`) {
		t.Fatal("失效状态错误")
	}
	valid = true
	out, _, err = runAuth([]string{"auth", "status", "--format", "json"}, "")
	if err != nil || !strings.Contains(out, `"status": "valid"`) {
		t.Fatal("有效状态错误")
	}
	if !strings.Contains(out, `"product.purchase.create": true`) || !strings.Contains(out, `"order.secret.read": true`) {
		t.Fatal("有效状态应展示 grant")
	}
	count := requests
	_, _, _ = runAuth([]string{"account", "summary", "--print-paths"}, "")
	if requests != count {
		t.Fatal("print-paths 不得继续执行请求")
	}
	if _, _, err := runAuth([]string{"auth", "logout"}, ""); err != nil {
		t.Fatal(err)
	}
	out, _, err = runAuth([]string{"auth", "status", "--format", "json"}, "")
	if err == nil || !strings.Contains(out, `"status": "unconfigured"`) || requests != count {
		t.Fatal("退出后应未配置且不发请求")
	}
}

func TestAuthInputAndUnreachable(t *testing.T) {
	authHome(t)
	for _, input := range []string{"", "two tokens", strings.Repeat("x", 8193)} {
		if _, _, err := runAuth([]string{"auth", "login", "--token-stdin"}, input); err == nil {
			t.Fatal("非法输入应拒绝")
		}
	}
	if _, _, err := runAuth([]string{"auth", "login"}, "sentinel"); err == nil {
		t.Fatal("非终端不能静默读入")
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close()
	t.Setenv("KDL_AGENT_GATEWAY_URL", srv.URL)
	t.Setenv("KDL_AGENT_TOKEN", "sentinel-environment")
	out, diagnostic, err := runAuth([]string{"auth", "status", "--format", "json"}, "")
	if err == nil || !strings.Contains(out, `"status": "unreachable"`) || strings.Contains(out+diagnostic, "sentinel") {
		t.Fatal("不可达状态错误")
	}
}
