package downstream

import (
	"encoding/json"
	"testing"
)

func TestParseGetDPSData_stringProxyList(t *testing.T) {
	raw := json.RawMessage(`{"count":1,"proxy_list":["115.231.184.18:16896"]}`)
	proxies, count, err := parseGetDPSData(raw)
	if err != nil {
		t.Fatalf("parseGetDPSData: %v", err)
	}
	if count != 1 {
		t.Fatalf("count=%d want 1", count)
	}
	if len(proxies) != 1 || proxies[0] != "115.231.184.18:16896" {
		t.Fatalf("proxies=%v", proxies)
	}
}

func TestParseGetDPSData_objectProxyList(t *testing.T) {
	raw := json.RawMessage(`{"count":2,"proxy_list":[{"ip":"1.2.3.4","port":8080},{"ip":"5.6.7.8","port":9090}]}`)
	proxies, count, err := parseGetDPSData(raw)
	if err != nil {
		t.Fatalf("parseGetDPSData: %v", err)
	}
	if count != 2 {
		t.Fatalf("count=%d want 2", count)
	}
	if len(proxies) != 2 || proxies[0] != "1.2.3.4:8080" || proxies[1] != "5.6.7.8:9090" {
		t.Fatalf("proxies=%v", proxies)
	}
}

func TestParseGetDPSData_empty(t *testing.T) {
	proxies, count, err := parseGetDPSData(nil)
	if err != nil {
		t.Fatalf("parseGetDPSData: %v", err)
	}
	if count != 0 || len(proxies) != 0 {
		t.Fatalf("count=%d proxies=%v", count, proxies)
	}
}
