package downstream

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestProxyAuthRejectsInvalidResponses(t *testing.T) {
	for _, tc := range []struct {
		status int
		body   string
	}{
		{302, `{"code":0,"data":{"type":"Basic","credentials":"dTpw"}}`},
		{500, `{"code":0,"data":{"type":"Basic","credentials":"dTpw"}}`},
		{200, `{"code":-3,"msg":"secret-sentinel"}`},
		{200, `{"data":{"type":"Basic","credentials":"dTpw"}}`},
		{200, `{"code":0,"data":{"type":"Bearer","credentials":"secret-sentinel"}}`},
		{200, `{"code":0,"data":{"type":"Basic","credentials":"invalid"}}`},
		{200, `secret-sentinel`},
		{200, `{"code":0,"data":{"type":"Basic","credentials":"dXNlcg=="}}`},
		{200, `{"code":0,"data":{"type":"Basic","credentials":"dTpw\n"}}`},
	} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Location", "/redirect")
			w.WriteHeader(tc.status)
			_, _ = w.Write([]byte(tc.body))
		}))
		t.Setenv("KDL_AGENT_ORDER_API_BASE_URL", "")
		_, err := FetchProxyAuthorization(context.Background(), srv.URL, "id", "secret-sentinel")
		srv.Close()
		if err == nil || strings.Contains(err.Error(), "secret-sentinel") || strings.Contains(err.Error(), "signature=") {
			t.Fatalf("错误未安全处理: %v", err)
		}
	}
}
