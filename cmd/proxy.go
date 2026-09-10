package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/gizaZerozhang/kdl-agent-cli/internal/cmdutil"
	"github.com/gizaZerozhang/kdl-agent-cli/internal/m3b"
	"github.com/gizaZerozhang/kdl-agent-cli/internal/output"
	"github.com/spf13/cobra"
)

func newProxyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "proxy",
		Short: "代理提取便利命令（需 order.secret.read）",
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
	return cmd
}
