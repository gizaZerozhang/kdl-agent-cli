package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/kuaidaili/kdl-agent-cli/internal/cmdutil"
	"github.com/kuaidaili/kdl-agent-cli/internal/downstream"
	"github.com/kuaidaili/kdl-agent-cli/internal/m3b"
	"github.com/kuaidaili/kdl-agent-cli/internal/output"
	"github.com/kuaidaili/kdl-agent-cli/internal/secret"
	"github.com/spf13/cobra"
)

func newProxyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "proxy",
		Short: "代理提取与连接认证（需 order.secret.read）",
	}
	fetch := &cobra.Command{
		Use:   "fetch",
		Short: "提取代理（Secret + 直连订单 OpenAPI）",
		RunE: func(cmd *cobra.Command, args []string) error {
			f := newFactory()
			orderID, _ := cmd.Flags().GetString("order")
			if strings.TrimSpace(orderID) == "" {
				return fail(f, fmt.Errorf("需要 --order"), cmdutil.ExitConfig)
			}
			c, err := loadClient(f)
			if err != nil {
				return err
			}
			num, _ := cmd.Flags().GetInt("num")
			proxyFormat, _ := cmd.Flags().GetString("proxy-format")
			result, err := m3b.RunProxyFetch(context.Background(), c, m3b.ProxyFetchParams{
				OrderID: orderID,
				Num:     num,
				Format:  proxyFormat,
			})
			if err != nil {
				return fail(f, err, cmdutil.ExitRuntime)
			}
			fmt.Fprintf(f.Out.Stderr, "代理连接可能需要 Basic 认证；使用 kdl-agent proxy auth --order %s 获取。订单 SecretKey 不是代理密码。\n", orderID)
			if f.Out.Format == output.ModeJSON {
				return f.Out.PrintJSON(map[string]any{
					"proxy_count": result.ProxyCount,
					"proxies":     result.Proxies,
				})
			}
			fmt.Fprintf(f.Out.Stdout, "proxy_count=%d\n", result.ProxyCount)
			for _, p := range result.Proxies {
				fmt.Fprintf(f.Out.Stdout, "  %s\n", p)
			}
			return nil
		},
	}
	fetch.Flags().String("order", "", "订单号（必填）")
	fetch.Flags().Int("num", 1, "提取数量")
	fetch.Flags().String("proxy-format", "json", "下游 Mock/真实 API 响应格式 json|text")
	cmd.AddCommand(fetch)
	cmd.AddCommand(newProxyAuthCmd())
	return cmd
}

func newProxyAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "auth", Short: "获取代理 Basic 鉴权（默认隐藏；--format json 输出敏感值）", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			f := newFactory()
			orderID, _ := cmd.Flags().GetString("order")
			if strings.TrimSpace(orderID) == "" {
				return fail(f, fmt.Errorf("需要 --order"), cmdutil.ExitConfig)
			}
			c, err := loadClient(f)
			if err != nil {
				return err
			}
			sec, err := secret.FetchOrderSecret(cmd.Context(), c, orderID)
			if err != nil {
				return fail(f, err, cmdutil.ExitRuntime)
			}
			auth, err := downstream.FetchProxyAuthorization(cmd.Context(), sec.APIDomain, sec.SecretID, sec.SecretKey)
			if err != nil {
				return fail(f, err, cmdutil.ExitRuntime)
			}
			if f.Out.Format == output.ModeJSON {
				fmt.Fprintln(f.Out.Stderr, "代理认证将输出敏感值；仅用于 Proxy-Authorization，不要发送给目标网站或写入日志。")
				return f.Out.PrintJSON(auth)
			}
			fmt.Fprintln(f.Out.Stdout, "type=Basic credentials=<redacted>")
			fmt.Fprintln(f.Out.Stderr, "使用 --format json 供客户端在内存中读取代理认证。")
			return nil
		},
	}
	cmd.Flags().String("order", "", "订单号（必填）")
	return cmd
}
