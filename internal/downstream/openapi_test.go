package downstream

import (
	"net/url"
	"testing"
)

func TestComputeOpenAPISignature(t *testing.T) {
	params := url.Values{}
	params.Set("secret_id", "osecret123456789012")
	params.Set("sign_type", "hmacsha1")
	params.Set("timestamp", "1609459200")
	params.Set("nonce", "1234567890")
	params.Set("iplist", "127.0.0.1")

	got := computeOpenAPISignature(
		"GET",
		"/api/setipwhitelist/",
		params,
		"test_secret_key_32_chars_padding!",
	)
	if got == "" {
		t.Fatal("signature empty")
	}
	// 同输入应稳定
	again := computeOpenAPISignature(
		"GET",
		"/api/setipwhitelist/",
		params,
		"test_secret_key_32_chars_padding!",
	)
	if got != again {
		t.Fatalf("signature not stable: %q vs %q", got, again)
	}
}
