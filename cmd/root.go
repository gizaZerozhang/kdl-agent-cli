// Package cmd 定义 kdl-agent 根命令与子命令树。
package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"github.com/gizaZerozhang/kdl-agent-cli/internal/client"
	"github.com/gizaZerozhang/kdl-agent-cli/internal/cmdutil"
	"github.com/gizaZerozhang/kdl-agent-cli/internal/config"
	"github.com/gizaZerozhang/kdl-agent-cli/internal/output"
	"github.com/spf13/cobra"
)

// Version 与 Commit 由发行构建注入；开发构建不冒充正式版本。
var Version = "0.1.0-dev"
var Commit = "unknown"

var (
	globalFormat     string
	globalQuiet      bool
	globalConfigPath string
	globalColor      string
	commandOut       io.Writer = os.Stdout
	commandErr       io.Writer = os.Stderr
)

// Execute 运行 CLI 并返回退出码。
func Execute() int {
	root := NewRootCmd()
	if err := root.Execute(); err != nil {
		var exitErr *exitError
		if errors.As(err, &exitErr) {
			return exitErr.code
		}
		fmt.Fprintln(root.ErrOrStderr(), "命令执行失败:", err)
		fmt.Fprintln(root.ErrOrStderr(), "请使用 kdl-agent --help 检查命令和参数。")
		return cmdutil.ExitCodeFromError(err)
	}
	return cmdutil.ExitOK
}

type exitError struct {
	code int
}

func (e *exitError) Error() string {
	return fmt.Sprintf("exit %d", e.code)
}

// NewRootCmd 构造根命令。
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "kdl-agent",
		Short:         "快代理 Agent Gateway 官方 CLI",
		Long:          "调用快代理 Agent Gateway /v1/，提供查询、授权写入与代理便利命令。",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	root.PersistentFlags().StringVar(&globalFormat, "format", "table", "输出格式: table|json")
	root.PersistentFlags().BoolVarP(&globalQuiet, "quiet", "q", false, "减少非 JSON 摘要输出")
	root.PersistentFlags().StringVar(&globalConfigPath, "config", "", "普通配置路径（相对 CWD；覆写 KDL_AGENT_CONFIG 与家目录 .kdl/config.toml）")
	root.PersistentFlags().StringVar(&globalColor, "color", "auto", "终端颜色: auto|always|never（尊重 NO_COLOR）")
	root.PersistentFlags().Bool("print-paths", false, "打印普通配置和凭证绝对路径及来源后退出，不发请求")

	root.Version = Version
	root.SetVersionTemplate("kdl-agent {{.Version}} (commit " + Commit + ")\n")

	root.AddCommand(newAuthCmd())
	root.AddCommand(newAccountCmd())
	root.AddCommand(newOrderCmd())
	root.AddCommand(newProductCmd())
	root.AddCommand(newMsgCmd())
	root.AddCommand(newTicketCmd())
	root.AddCommand(newProxyCmd())
	root.AddCommand(newUpdateCmd())

	root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		commandOut, commandErr = cmd.OutOrStdout(), cmd.ErrOrStderr()
		config.SetPathOverride(globalConfigPath)
		if wantsHelp() {
			return nil
		}
		printPaths, _ := cmd.Flags().GetBool("print-paths")
		if printPaths {
			if err := runPrintPaths(); err != nil {
				return err
			}
			return &exitError{code: cmdutil.ExitOK}
		}
		if globalFormat != "table" && globalFormat != "json" {
			return fail(newFactory(), errors.New("输出格式无效；请使用 --format table 或 json"), cmdutil.ExitConfig)
		}
		if globalColor != "auto" && globalColor != "always" && globalColor != "never" {
			return fail(newFactory(), errors.New("颜色选项无效；请使用 --color auto、always 或 never"), cmdutil.ExitConfig)
		}
		return nil
	}

	return root
}

func wantsHelp() bool {
	for _, a := range os.Args[1:] {
		if a == "-h" || a == "--help" {
			return true
		}
	}
	return false
}

func newFactory() *cmdutil.Factory {
	format := output.ModeTable
	if strings.EqualFold(globalFormat, "json") {
		format = output.ModeJSON
	}
	f := cmdutil.NewFactory(format)
	f.Out.Stdout, f.Out.Stderr = commandOut, commandErr
	f.Out.Quiet = globalQuiet
	if format != output.ModeJSON {
		f.Out.ColorEnabled = output.ResolveColor(globalColor, output.IsStdoutTTY())
	}
	return f
}

func runPrintPaths() error {
	f := newFactory()
	cfg, err := config.Load()
	if err != nil {
		return fail(f, err, cmdutil.ExitConfig)
	}
	fmt.Fprintf(f.Out.Stdout, "config_path=%s\n", cfg.ConfigPath)
	fmt.Fprintf(f.Out.Stdout, "credentials_path=%s\n", cfg.CredentialsPath)
	fmt.Fprintf(f.Out.Stdout, "config_source=%s\ntoken_source=%s\n", cfg.ConfigSource, cfg.TokenSource)
	if cfg.GatewayURL != "" {
		fmt.Fprintf(f.Out.Stdout, "gateway_url=%s\n", cfg.GatewayURL)
	}
	fmt.Fprintf(f.Out.Stdout, "token_configured=%t\n", strings.TrimSpace(cfg.Token) != "")
	return nil
}

func runGET(path string) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		return runGETWithQuery(path, nil)(cmd, args)
	}
}

func runGETWithQuery(path string, query url.Values) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		f := newFactory()
		c, err := loadClient(f)
		if err != nil {
			return err
		}
		resp, err := c.Get(context.Background(), path, query)
		if err != nil {
			return fail(f, err, cmdutil.ExitRuntime)
		}
		if err := f.Out.PrintEnvelope(resp); err != nil {
			return fail(f, err, cmdutil.ExitRuntime)
		}
		return nil
	}
}

func runPOST(path string, body any) func(*cobra.Command, []string) error {
	return runPOSTWithHeaders(path, body, nil)
}

func runPOSTWithHeaders(path string, body any, headers map[string]string) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		f := newFactory()
		c, err := loadClient(f)
		if err != nil {
			return err
		}
		resp, err := c.PostJSONWithHeaders(context.Background(), path, body, headers)
		if err != nil {
			return fail(f, err, cmdutil.ExitRuntime)
		}
		if err := f.Out.PrintEnvelope(resp); err != nil {
			return fail(f, err, cmdutil.ExitRuntime)
		}
		return nil
	}
}

func loadClient(f *cmdutil.Factory) (*client.Client, error) {
	c, _, err := f.LoadClient()
	if err != nil {
		return nil, fail(f, err, cmdutil.ExitConfig)
	}
	return c, nil
}

func fail(f *cmdutil.Factory, err error, code int) error {
	// 所有命令统一处理 Gateway 错误，下游错误保持原诊断格式。
	var apiErr *client.APIError
	if errors.As(err, &apiErr) && apiErr != nil {
		if f.Out.Format == output.ModeJSON {
			if envelope := apiErr.Response; envelope != nil {
				if err := writeJSON(f.Out, envelope); err != nil {
					return fail(f, err, code)
				}
			}
		}
		f.Out.PrintError(err)
		if apiErr.RetryAfter > 0 {
			fmt.Fprintf(f.Out.Stderr, "建议: 至少等待 %d 秒后重试（Retry-After）；不要自动重放结果不确定的写操作。\n", apiErr.RetryAfter)
		}
		for _, item := range apiErr.ValidationErrors() {
			fmt.Fprintf(f.Out.Stderr, "  - %s\n", item)
		}
		if apiErr.Code == "AUTH_REQUIRED" || apiErr.Code == "CREDENTIAL_REVOKED" {
			fmt.Fprintln(f.Out.Stderr, "建议: 前往会员中心 /uc/agent/settings/ 检查凭证，然后 kdl-agent auth login")
		}
		return &exitError{code: code}
	}
	f.Out.PrintError(err)
	return &exitError{code: code}
}

func writeJSON(out *output.Writer, v any) error {
	enc := json.NewEncoder(out.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func listQuery(cmd *cobra.Command) url.Values {
	q := url.Values{}
	if cursor, err := cmd.Flags().GetString("cursor"); err == nil && cursor != "" {
		q.Set("cursor", cursor)
	}
	if limit, err := cmd.Flags().GetInt("limit"); err == nil && limit > 0 {
		q.Set("limit", fmt.Sprintf("%d", limit))
	}
	return q
}

func addListFlags(cmd *cobra.Command) {
	cmd.Flags().String("cursor", "", "分页游标")
	cmd.Flags().Int("limit", 0, "每页条数 (1-100)")
}
