package cmd_test

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kuaidaili/kdl-agent-cli/internal/downstream"
)

// 本地模拟完整链路，确保认证只用于 CONNECT，不进入目标网站请求。
func TestProxyAuthHTTPS(t *testing.T) {
	authHome(t)
	credential := base64.StdEncoding.EncodeToString([]byte("proxy-user:proxy-password"))
	var proxyEndpoint string
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("signature") == "" || r.Header.Get("Authorization") != "" {
			t.Error("订单 API 请求路径、签名或凭证隔离错误")
		}
		if r.URL.Path == "/api/getdps/" {
			if r.URL.Query().Get("num") != "1" {
				t.Error("应只提取一个代理")
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": map[string]any{"count": 1, "proxy_list": []string{proxyEndpoint}}})
			return
		}
		if r.URL.Path != "/api/getproxyauthorization/" {
			t.Error("非预期订单 API")
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": map[string]string{"type": "Basic", "credentials": credential}})
	}))
	defer api.Close()
	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/orders/123/secret" || r.Header.Get("Authorization") != "Bearer agent-sentinel" {
			t.Error("Gateway 请求不正确")
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "data": map[string]string{"order_id": "123", "secret_id": "api-id", "secret_key": "api-key-sentinel", "api_domain": api.URL}})
	}))
	defer gateway.Close()
	t.Setenv("KDL_AGENT_GATEWAY_URL", gateway.URL)
	t.Setenv("KDL_AGENT_TOKEN", "agent-sentinel")
	t.Setenv("KDL_AGENT_ORDER_API_BASE_URL", "")
	t.Setenv("KDL_AGENT_DOWNSTREAM_MOCK", "0")
	out, diagnostic, err := runAuth([]string{"proxy", "auth", "--order", "123"}, "")
	if err != nil || !strings.Contains(out, "<redacted>") || strings.Contains(out+diagnostic, credential) {
		t.Fatalf("默认输出必须遮蔽认证: %v", err)
	}
	out, diagnostic, err = runAuth([]string{"proxy", "auth", "--order", "123", "--format", "json"}, "")
	if err != nil {
		t.Fatal(err)
	}
	var auth downstream.ProxyAuthorization
	if json.Unmarshal([]byte(out), &auth) != nil || auth.Credentials != credential {
		t.Fatal("JSON 认证不可用")
	}
	if strings.Contains(out+diagnostic, "api-key-sentinel") || strings.Contains(out+diagnostic, "agent-sentinel") {
		t.Fatal("泄露其他凭据")
	}

	target := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Proxy-Authorization") != "" || r.Header.Get("Authorization") != "" {
			t.Error("代理凭据到达目标网站")
		}
		_, _ = w.Write([]byte("target-ok"))
	}))
	defer target.Close()
	targetURL, _ := url.Parse(target.URL)
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "CONNECT" || r.Host != targetURL.Host {
			w.WriteHeader(400)
			return
		}
		if r.Header.Get("Proxy-Authorization") != "Basic "+credential {
			w.WriteHeader(407)
			return
		}
		upstream, err := net.Dial("tcp", targetURL.Host)
		if err != nil {
			t.Error(err)
			w.WriteHeader(502)
			return
		}
		conn, buffer, err := w.(http.Hijacker).Hijack()
		if err != nil {
			upstream.Close()
			t.Error(err)
			return
		}
		defer conn.Close()
		defer upstream.Close()
		_, _ = buffer.WriteString("HTTP/1.1 200 Connection Established\r\n\r\n")
		_ = buffer.Flush()
		go func() { _, _ = io.Copy(upstream, buffer); upstream.Close() }()
		_, _ = io.Copy(conn, upstream)
	}))
	defer proxy.Close()
	proxyEndpoint = strings.TrimPrefix(proxy.URL, "http://")
	fetchOut, fetchDiagnostic, err := runAuth([]string{"proxy", "fetch", "--order", "123", "--num", "1", "--format", "json"}, "")
	var fetched downstream.ProxyResult
	if err != nil || json.Unmarshal([]byte(fetchOut), &fetched) != nil || fetched.ProxyCount != 1 || len(fetched.Proxies) != 1 || fetched.Proxies[0] != proxyEndpoint || !strings.Contains(fetchDiagnostic, "proxy auth --order 123") {
		t.Fatalf("提取与认证指引失败: %v", err)
	}
	proxyURL, _ := url.Parse("http://" + fetched.Proxies[0])
	transport := target.Client().Transport.(*http.Transport).Clone()
	transport.Proxy = http.ProxyURL(proxyURL)
	transport.TLSClientConfig = &tls.Config{RootCAs: transport.TLSClientConfig.RootCAs}
	defer transport.CloseIdleConnections()
	httpClient := &http.Client{Transport: transport, Timeout: 5 * time.Second}
	if resp, err := httpClient.Get(target.URL); err == nil {
		resp.Body.Close()
		t.Fatal("缺认证应返回 CONNECT 407")
	}
	transport.ProxyConnectHeader = http.Header{"Proxy-Authorization": {auth.Type + " " + auth.Credentials}}
	resp, err := httpClient.Get(target.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 || string(body) != "target-ok" {
		t.Fatal("HTTPS 连通性验证失败")
	}
	t.Run("指南中的curl代理认证", func(t *testing.T) {
		curl, err := exec.LookPath("curl")
		if err != nil {
			t.Skip("未安装 curl")
		}
		cert := filepath.Join(t.TempDir(), "target.pem")
		if err := os.WriteFile(cert, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: target.Certificate().Raw}), 0o600); err != nil {
			t.Fatal(err)
		}
		header, _ := json.Marshal("Proxy-Authorization: " + auth.Type + " " + auth.Credentials)
		config := "proxy = \"" + proxy.URL + "\"\nproxy-header = " + string(header) + "\nurl = \"" + target.URL + "\"\n"
		command := exec.Command(curl, "--config", "-", "--noproxy", "", "--silent", "--show-error", "--fail", "--max-time", "5", "--cacert", cert, "--output", os.DevNull, "--write-out", "%{http_code}\\n")
		command.Stdin = strings.NewReader(config)
		result, err := command.Output()
		if err != nil || strings.TrimSpace(string(result)) != "200" {
			t.Fatalf("curl CONNECT 验证失败: %v", err)
		}
	})
}

func TestProxyAuthRequiresOrder(t *testing.T) {
	_, _, err := runAuth([]string{"proxy", "auth"}, "")
	if err == nil {
		t.Fatal("缺订单号必须失败")
	}
}
