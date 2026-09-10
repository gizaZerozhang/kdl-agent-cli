package client_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gizaZerozhang/kdl-agent-cli/internal/client"
	"github.com/gizaZerozhang/kdl-agent-cli/internal/config"
)

func TestGetSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Fatalf("auth header: %q", r.Header.Get("Authorization"))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success":    true,
			"request_id": "req_test",
			"data":       map[string]any{"balance": 1.23},
		})
	}))
	defer srv.Close()

	c, err := client.New(config.Resolved{GatewayURL: srv.URL, Token: "test-token"})
	if err != nil {
		t.Fatal(err)
	}
	resp, err := c.Get(context.Background(), "/v1/account/funds", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Success || resp.RequestID != "req_test" {
		t.Fatalf("resp: %+v", resp)
	}
}

func TestRedirectDoesNotValidateOrForwardCredential(t *testing.T) {
	forwarded := false
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { forwarded = true }))
	defer target.Close()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", target.URL)
		w.WriteHeader(http.StatusFound)
		_, _ = w.Write([]byte(`{"success":true,"data":{}}`))
	}))
	defer srv.Close()
	c, err := client.New(config.Resolved{GatewayURL: srv.URL, Token: "sentinel-redirect"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Get(context.Background(), "/v1/account/summary", nil); err == nil || forwarded {
		t.Fatal("重定向不能验证成功或转发凭证")
	}
}

func TestRevocationPreservesCodeAndRedactsCredential(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"success":false,"request_id":"req-sentinel-reflected","error":{"code":"CREDENTIAL_REVOKED","message":"sentinel-reflected"}}`))
	}))
	defer srv.Close()
	c, err := client.New(config.Resolved{GatewayURL: srv.URL, Token: "sentinel-reflected"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.Get(context.Background(), "/v1/account/summary", nil)
	var apiErr *client.APIError
	if !errors.As(err, &apiErr) || apiErr.Code != "CREDENTIAL_REVOKED" || apiErr.RequestID == "" || strings.Contains(err.Error(), "sentinel-reflected") {
		t.Fatal("错误契约或脱敏行为错误")
	}
}

func TestGetAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success":    false,
			"request_id": "req_denied",
			"error":      map[string]any{"code": "AUTH_REQUIRED", "message": "需要有效的 Agent 凭证"},
		})
	}))
	defer srv.Close()

	c, err := client.New(config.Resolved{GatewayURL: srv.URL, Token: "bad"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.Get(context.Background(), "/v1/account/funds", nil)
	if err == nil {
		t.Fatal("expected error")
	}
	var apiErr *client.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("want APIError, got %T", err)
	}
	if apiErr.Code != "AUTH_REQUIRED" {
		t.Fatalf("code: %s", apiErr.Code)
	}
}
