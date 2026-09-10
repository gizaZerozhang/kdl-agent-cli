package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/gizaZerozhang/kdl-agent-cli/internal/client"
	"github.com/gizaZerozhang/kdl-agent-cli/internal/cmdutil"
	"github.com/gizaZerozhang/kdl-agent-cli/internal/config"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newAuthCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "auth", Short: "Agent 凭证与 Gateway 连接配置"}
	cmd.AddCommand(newAuthLoginCmd(), newAuthStatusCmd(), newAuthLogoutCmd())
	return cmd
}

func readLoginToken(cmd *cobra.Command, stdin bool) (string, error) {
	var raw []byte
	var err error
	if stdin {
		raw, err = io.ReadAll(io.LimitReader(cmd.InOrStdin(), 8193))
	} else {
		input, ok := cmd.InOrStdin().(*os.File)
		if !ok || !term.IsTerminal(int(input.Fd())) {
			return "", errors.New("登录需要交互终端；请在本地终端执行 auth login，自动化可显式使用 --token-stdin")
		}
		fmt.Fprint(cmd.ErrOrStderr(), "Agent 凭证（隐藏输入）: ")
		raw, err = term.ReadPassword(int(input.Fd()))
		fmt.Fprintln(cmd.ErrOrStderr())
	}
	defer func() {
		for i := range raw {
			raw[i] = 0
		}
	}()
	if err != nil {
		return "", errors.New("读取凭证失败或输入已取消；未保存，请重新执行 auth login")
	}
	token := strings.TrimSpace(string(raw))
	if token == "" || len(raw) > 8192 || strings.ContainsAny(token, "\r\n\t ") {
		return "", errors.New("凭证为空、过长或含空白；未保存，请重新输入完整 Agent 凭证")
	}
	return token, nil
}

// verifyCredential 只消费验证结果，不输出账户数据或服务端原始错误。
func verifyCredential(cmd *cobra.Command, cfg config.Resolved) (string, error) {
	c, err := client.New(cfg)
	if err != nil {
		return "error", err
	}
	ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
	defer cancel()
	_, err = c.Get(ctx, "/v1/account/summary", nil)
	if err == nil {
		return "valid", nil
	}
	var apiErr *client.APIError
	if errors.As(err, &apiErr) {
		if apiErr.StatusCode == 401 || apiErr.Code == "AUTH_REQUIRED" || apiErr.Code == "CREDENTIAL_REVOKED" || apiErr.Code == "CREDENTIAL_EXPIRED" {
			return "invalid", errors.New("Agent 凭证无效、过期或已撤销；请在会员中心检查后重新执行 auth login")
		}
		return "error", errors.New("Gateway 拒绝验证或服务暂不可用；请检查授权、服务状态，限流时稍后重试")
	}
	return "unreachable", errors.New("无法完成 Gateway 验证；请检查地址、网络、TLS 及服务响应后重试")
}

func newAuthLoginCmd() *cobra.Command {
	var gatewayURL string
	var tokenStdin bool
	cmd := &cobra.Command{
		Use: "login", Short: "隐藏输入 Agent 凭证，验证成功后保存到家目录", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			f := newFactory()
			cfg, err := config.Load()
			if err != nil {
				return fail(f, err, cmdutil.ExitConfig)
			}
			if cmd.Flags().Changed("gateway-url") {
				cfg.GatewayURL, err = config.NormalizeGateway(gatewayURL)
				if err != nil {
					return fail(f, err, cmdutil.ExitConfig)
				}
			}
			cfg.Token, err = readLoginToken(cmd, tokenStdin)
			if err != nil {
				return fail(f, err, cmdutil.ExitConfig)
			}
			if !globalQuiet {
				fmt.Fprintf(f.Out.Stderr, "正在验证 Gateway: %s\n", cfg.GatewayURL)
			}
			if _, err := verifyCredential(cmd, cfg); err != nil {
				return fail(f, err, cmdutil.ExitRuntime)
			}
			if err := config.SaveLogin(cfg.ConfigPath, cfg.GatewayURL, cfg.Token); err != nil {
				return fail(f, err, cmdutil.ExitConfig)
			}
			if os.Getenv("KDL_AGENT_TOKEN") != "" {
				fmt.Fprintln(f.Out.Stderr, "运行环境中的 KDL_AGENT_TOKEN 仍优先；本次保存未改变该注入值。")
			}
			if strings.EqualFold(globalFormat, "json") {
				return writeJSON(f.Out, map[string]any{"config_path": cfg.ConfigPath, "credentials_path": cfg.CredentialsPath, "gateway_url": cfg.GatewayURL, "saved": true, "status": "valid"})
			}
			fmt.Fprintf(f.Out.Stdout, "已验证并保存凭证: %s\nGateway: %s\n", cfg.CredentialsPath, cfg.GatewayURL)
			return nil
		},
	}
	cmd.Flags().StringVar(&gatewayURL, "gateway-url", "", "Gateway HTTPS 基址（默认使用当前配置；本机回环测试可用 HTTP）")
	cmd.Flags().BoolVar(&tokenStdin, "token-stdin", false, "从标准输入读取凭证并验证保存；不要将凭证放入命令参数")
	return cmd
}

func newAuthStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use: "status", Short: "验证当前凭证并显示来源，不回显明文", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			f := newFactory()
			cfg, err := config.Load()
			if err != nil {
				return fail(f, err, cmdutil.ExitConfig)
			}
			status, code := "unconfigured", cmdutil.ExitConfig
			var verifyErr error
			if cfg.Token != "" {
				if !globalQuiet {
					fmt.Fprintf(f.Out.Stderr, "正在验证 Gateway: %s\n", cfg.GatewayURL)
				}
				status, verifyErr = verifyCredential(cmd, cfg)
				code = cmdutil.ExitRuntime
			}
			if strings.EqualFold(globalFormat, "json") {
				if err := writeJSON(f.Out, map[string]any{"config_path": cfg.ConfigPath, "credentials_path": cfg.CredentialsPath, "gateway_url": cfg.GatewayURL, "config_source": cfg.ConfigSource, "token_source": cfg.TokenSource, "token_configured": cfg.Token != "", "status": status}); err != nil {
					return err
				}
			} else {
				labels := map[string]string{"unconfigured": "未配置", "valid": "服务端验证有效", "invalid": "凭证无效", "unreachable": "无法完成远端验证", "error": "服务端验证失败"}
				fmt.Fprintf(f.Out.Stdout, "Gateway: %s\n配置文件: %s\n凭证文件: %s\n凭证来源: %s\n状态: %s\n", cfg.GatewayURL, cfg.ConfigPath, cfg.CredentialsPath, cfg.TokenSource, labels[status])
			}
			if status == "valid" {
				return nil
			}
			if verifyErr == nil {
				verifyErr = config.ValidateForAPI(cfg)
			}
			return fail(f, verifyErr, code)
		},
	}
}

func newAuthLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use: "logout", Short: "清除当前 Gateway 本地凭证，不撤销服务端凭证", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			f := newFactory()
			cfg, err := config.Load()
			if err != nil {
				return fail(f, err, cmdutil.ExitConfig)
			}
			if err := config.Logout(cfg.GatewayURL); err != nil {
				return fail(f, err, cmdutil.ExitConfig)
			}
			envActive := os.Getenv("KDL_AGENT_TOKEN") != ""
			if envActive {
				fmt.Fprintln(f.Out.Stderr, "KDL_AGENT_TOKEN 仍在运行环境中；请在注入端移除，本地退出不会使其失效。")
			}
			if strings.EqualFold(globalFormat, "json") {
				return writeJSON(f.Out, map[string]any{"gateway_url": cfg.GatewayURL, "cleared": true, "environment_token_active": envActive, "server_revoked": false})
			}
			fmt.Fprintf(f.Out.Stdout, "已清除当前 Gateway 的本地凭证: %s\n服务端吊销仍需在会员中心执行。\n", cfg.CredentialsPath)
			return nil
		},
	}
}

func newAccountCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "account", Short: "账户只读 Capability"}
	cmd.AddCommand(&cobra.Command{Use: "funds", Short: "查询账户余额（account.funds.read）", RunE: runGET("/v1/account/funds")})
	cmd.AddCommand(&cobra.Command{Use: "summary", Short: "查询账户摘要（account.summary.read）", RunE: runGET("/v1/account/summary")})
	return cmd
}
