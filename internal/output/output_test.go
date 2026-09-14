package output_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/kuaidaili/kdl-agent-cli/internal/client"
	"github.com/kuaidaili/kdl-agent-cli/internal/output"
)

func TestResolveColorNO_COLOR(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	if output.ResolveColor("always", true) {
		t.Fatal("NO_COLOR should disable color even with always")
	}
}

func TestResolveColorAlways(t *testing.T) {
	unsetNO_COLOR(t)
	if !output.ResolveColor("always", false) {
		t.Fatal("always should enable color without TTY")
	}
}

func TestResolveColorNever(t *testing.T) {
	unsetNO_COLOR(t)
	if output.ResolveColor("never", true) {
		t.Fatal("never should disable color")
	}
}

func TestResolveColorAutoNonTTY(t *testing.T) {
	unsetNO_COLOR(t)
	if output.ResolveColor("auto", false) {
		t.Fatal("auto should disable color when not TTY")
	}
}

func unsetNO_COLOR(t *testing.T) {
	t.Helper()
	prev, ok := os.LookupEnv("NO_COLOR")
	if ok {
		if err := os.Unsetenv("NO_COLOR"); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		if ok {
			os.Setenv("NO_COLOR", prev)
		}
	})
}

func TestJSONModeNoANSI(t *testing.T) {
	w := output.NewWriter(output.ModeJSON)
	w.ColorEnabled = true
	resp := &client.GatewayResponse{
		Data: json.RawMessage(`{"items":[{"id":"1"}]}`),
	}
	var buf bytes.Buffer
	w.Stdout = &buf
	if err := w.PrintEnvelope(resp); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), "\033[") {
		t.Fatalf("json stdout must not contain ANSI: %q", buf.String())
	}
}

func TestTableModeWithColor(t *testing.T) {
	w := output.NewWriter(output.ModeTable)
	w.ColorEnabled = true
	resp := &client.GatewayResponse{
		Data: json.RawMessage(`{"items":[{"id":"1","name":"test"}]}`),
	}
	var buf bytes.Buffer
	w.Stdout = &buf
	if err := w.PrintEnvelope(resp); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "\033[") {
		t.Fatalf("colored table should contain ANSI: %q", out)
	}
	if !strings.Contains(out, "id") || !strings.Contains(out, "test") {
		t.Fatalf("missing table content: %q", out)
	}
}

func TestTableModeWithoutColor(t *testing.T) {
	w := output.NewWriter(output.ModeTable)
	w.ColorEnabled = false
	resp := &client.GatewayResponse{
		Data: json.RawMessage(`{"items":[{"id":"1"}]}`),
	}
	var buf bytes.Buffer
	w.Stdout = &buf
	if err := w.PrintEnvelope(resp); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), "\033[") {
		t.Fatalf("plain table must not contain ANSI: %q", buf.String())
	}
}

func TestPrintErrorWithColor(t *testing.T) {
	w := output.NewWriter(output.ModeTable)
	w.ColorEnabled = true
	var buf bytes.Buffer
	w.Stderr = &buf
	w.PrintError(errors.New("missing token"))
	if !strings.Contains(buf.String(), "\033[") {
		t.Fatalf("colored error should contain ANSI: %q", buf.String())
	}
	if !strings.Contains(buf.String(), "missing token") {
		t.Fatalf("error text missing: %q", buf.String())
	}
}
