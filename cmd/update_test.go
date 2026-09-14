package cmd

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUpdateCheck(t *testing.T) {
	for _, tc := range []struct {
		name, current, body, status string
		code                        int
	}{
		{"numeric beta", "0.1.0-beta.2", `{"name":"@kuaidaili/kdl-agent","version":"0.1.0-beta.10"}`, "update_available", 200},
		{"no downgrade", "0.1.0-beta.10", `{"name":"@kuaidaili/kdl-agent","version":"0.1.0-beta.2"}`, "up_to_date", 200},
		{"offline", "0.1.0-beta.2", `{}`, "unavailable", 503},
		{"injection", "0.1.0-beta.2", `{"name":"@kuaidaili/kdl-agent","version":"1.0.0;echo bad"}`, "unavailable", 200},
		{"wrong package", "0.1.0-beta.2", `{"name":"other","version":"1.0.0"}`, "unavailable", 200},
		{"development", "0.1.0-dev", `{}`, "development", 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			hits := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				hits++
				if r.Header.Get("Authorization") != "" {
					t.Error("credential sent")
				}
				w.WriteHeader(tc.code)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()
			result := checkUpdate(context.Background(), tc.current, "beta", server.Client(), server.URL+"/")
			if result.Status != tc.status {
				t.Fatalf("%+v", result)
			}
			if tc.status == "development" && hits != 0 {
				t.Fatal("dev requested registry")
			}
		})
	}
}
