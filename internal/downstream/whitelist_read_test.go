package downstream

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWhitelistReadRejectsInvalidResponse(t *testing.T) {
	for _, body := range []string{`{}`, `{"code":0}`, `{"code":0,"data":{}}`, `{"code":0,"data":{"ipwhitelist":[],"count":1}}`, `{"code":-1,"msg":"secret-sentinel"}`} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(body)) }))
		t.Setenv("KDL_AGENT_ORDER_API_BASE_URL", "")
		_, err := GetWhitelist(srv.URL, "id", "secret-sentinel")
		srv.Close()
		if err == nil || strings.Contains(err.Error(), "sentinel") || strings.Contains(err.Error(), "signature") {
			t.Fatalf("响应未安全拒绝: %v", err)
		}
	}
}
