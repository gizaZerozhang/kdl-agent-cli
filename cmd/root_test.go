package cmd_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kuaidaili/kdl-agent-cli/cmd"
	"github.com/kuaidaili/kdl-agent-cli/internal/config"
)

func TestRootHelp(t *testing.T) {
	root := cmd.NewRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"--help"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(buf.Bytes(), []byte("account")) {
		t.Fatalf("help missing subcommands: %s", buf.String())
	}
}

func TestAccountFundsMissingConfig(t *testing.T) {
	authHome(t)
	dir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	t.Setenv("KDL_AGENT_CONFIG", filepath.Join(dir, "missing.toml"))
	t.Setenv("KDL_AGENT_GATEWAY_URL", "")
	t.Setenv("KDL_AGENT_TOKEN", "")

	root := cmd.NewRootCmd()
	root.SetOut(new(bytes.Buffer))
	root.SetErr(new(bytes.Buffer))
	root.SetArgs([]string{"account", "funds"})
	if err = root.Execute(); err == nil {
		t.Fatal("expected error")
	}
}

func TestPrintPathsWithConfigFlag(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "custom.toml")
	t.Setenv("KDL_AGENT_CONFIG", "")
	config.SetPathOverride(cfgPath)
	t.Cleanup(func() { config.SetPathOverride("") })

	got, err := config.ResolvePath()
	if err != nil {
		t.Fatal(err)
	}
	want, err := filepath.Abs(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("ResolvePath with --config equivalent: got %q want %q", got, want)
	}
}

func TestRootHelpIncludesColorAndConfigFlags(t *testing.T) {
	root := cmd.NewRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"--help"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	help := buf.String()
	for _, flag := range []string{"--color", "--config"} {
		if !strings.Contains(help, flag) {
			t.Fatalf("help missing %s: %s", flag, help)
		}
	}
}
