package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/gizaZerozhang/kdl-agent-cli/internal/cmdutil"
	"github.com/gizaZerozhang/kdl-agent-cli/internal/m3b"
	"github.com/gizaZerozhang/kdl-agent-cli/internal/output"
	"github.com/gizaZerozhang/kdl-agent-cli/internal/secret"
	"github.com/spf13/cobra"
)

func newOrderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "order",
		Short: "订单只读 Capability",
	}
	list := &cobra.Command{
		Use:   "list",
		Short: "订单列表（order.read）",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGETWithQuery("/v1/orders", listQuery(cmd))(cmd, args)
		},
	}
	addListFlags(list)
	cmd.AddCommand(list)
	cmd.AddCommand(&cobra.Command{
		Use:   "show [order_id]",
		Short: "订单详情（order.read）",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGET(fmt.Sprintf("/v1/orders/%s", args[0]))(cmd, args)
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "stats [order_id]",
		Short: "订单用量摘要（order.usage.summary.read）",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGET(fmt.Sprintf("/v1/orders/%s/stats", args[0]))(cmd, args)
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "guide [order_id]",
		Short: "订单接入指引（order.access.guide.read）",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGET(fmt.Sprintf("/v1/orders/%s/guide", args[0]))(cmd, args)
		},
	})
	secretCmd := &cobra.Command{
		Use:   "secret",
		Short: "订单 Secret（order.secret.read）",
	}
	secretGet := &cobra.Command{
		Use:   "get",
		Short: "获取订单 Secret（明文 JSON；勿写入日志）",
		RunE:  runOrderSecretGet,
	}
	secretGet.Flags().String("order", "", "订单号（必填）")
	_ = secretGet.MarkFlagRequired("order")
	secretCmd.AddCommand(secretGet)
	cmd.AddCommand(secretCmd)
	whitelist := &cobra.Command{
		Use:   "whitelist",
		Short: "订单 IP 白名单便利命令（需 order.secret.read）",
	}
	setWL := &cobra.Command{
		Use:   "set",
		Short: "设置白名单 IP 列表",
		RunE:  runOrderWhitelist("set", false),
	}
	setWL.Flags().String("order", "", "订单号（必填）")
	setWL.Flags().StringSlice("ip", nil, "IP 地址（可重复 --ip）")
	clearWL := &cobra.Command{
		Use:   "clear",
		Short: "清空白名单",
		RunE:  runOrderWhitelist("clear", true),
	}
	clearWL.Flags().String("order", "", "订单号（必填）")
	clearWL.Flags().Bool("clear", true, "必须为 true")
	whitelist.AddCommand(setWL, clearWL)
	cmd.AddCommand(whitelist)
	return cmd
}

func runOrderSecretGet(cmd *cobra.Command, args []string) error {
	f := newFactory()
	orderID, _ := cmd.Flags().GetString("order")
	if strings.TrimSpace(orderID) == "" {
		return fail(f, fmt.Errorf("需要 --order"), cmdutil.ExitConfig)
	}
	c, err := loadClient(f)
	if err != nil {
		return err
	}
	fmt.Fprintln(f.Out.Stderr, "警告: Secret 将进入终端输出，可能被 Agent/模型读取；关闭 grant 不会使已获取 Secret 失效。")
	data, err := secret.FetchOrderSecret(context.Background(), c, orderID)
	if err != nil {
		return fail(f, err, cmdutil.ExitRuntime)
	}
	if f.Out.Format == output.ModeJSON {
		return f.Out.PrintJSON(data)
	}
	fmt.Fprintf(f.Out.Stdout, "order_id=%s secret_id=%s api_domain=%s\n", data.OrderID, data.SecretID, data.APIDomain)
	fmt.Fprintf(f.Out.Stdout, "secret_key=<redacted>\n")
	return nil
}

func runOrderWhitelist(action string, isClear bool) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		f := newFactory()
		orderID, _ := cmd.Flags().GetString("order")
		if strings.TrimSpace(orderID) == "" {
			return fail(f, fmt.Errorf("需要 --order"), cmdutil.ExitConfig)
		}
		c, err := loadClient(f)
		if err != nil {
			return err
		}
		ips, _ := cmd.Flags().GetStringSlice("ip")
		result, err := m3b.RunWhitelistWrite(context.Background(), c, m3b.WhitelistParams{
			OrderID: orderID,
			Action:  action,
			IPs:     ips,
			Clear:   isClear,
		})
		if err != nil {
			return fail(f, err, cmdutil.ExitRuntime)
		}
		if f.Out.Format == output.ModeJSON {
			return f.Out.PrintJSON(map[string]any{
				"whitelist_ip_count": result.WhitelistIPCount,
			})
		}
		fmt.Fprintf(f.Out.Stdout, "whitelist_ip_count=%d\n", result.WhitelistIPCount)
		return nil
	}
}

func newProductCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "product",
		Short: "产品发现与报价 Capability",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "产品目录（product.discovery.read）",
		RunE:  runGET("/v1/products"),
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "trial-eligibility",
		Short: "试用资格（product.discovery.read）",
		RunE:  runGET("/v1/products/trial-eligibility"),
	})
	spec := &cobra.Command{
		Use:   "spec",
		Short: "产品规格（product.purchase.read）",
		RunE: func(cmd *cobra.Command, args []string) error {
			pt, _ := cmd.Flags().GetString("product-type")
			if pt == "" {
				return fail(newFactory(), fmt.Errorf("需要 --product-type"), cmdutil.ExitConfig)
			}
			return runGET(fmt.Sprintf("/v1/products/%s/spec", pt))(cmd, args)
		},
	}
	spec.Flags().String("product-type", "", "产品类型，如 tps_pro")
	cmd.AddCommand(spec)
	quote := &cobra.Command{
		Use:   "quote",
		Short: "产品报价（product.purchase.read）",
		RunE: func(cmd *cobra.Command, args []string) error {
			bodyFile, _ := cmd.Flags().GetString("body-file")
			var body map[string]any
			if bodyFile != "" {
				raw, err := os.ReadFile(bodyFile)
				if err != nil {
					return fail(newFactory(), err, cmdutil.ExitConfig)
				}
				if err := json.Unmarshal(raw, &body); err != nil {
					return fail(newFactory(), fmt.Errorf("解析 body-file: %w", err), cmdutil.ExitConfig)
				}
			} else {
				pt, _ := cmd.Flags().GetString("product-type")
				sv, _ := cmd.Flags().GetString("spec-version")
				if pt == "" || sv == "" {
					return fail(newFactory(), fmt.Errorf("需要 --body-file 或同时提供 --product-type 与 --spec-version"), cmdutil.ExitConfig)
				}
				body = map[string]any{
					"product_type":  pt,
					"spec_version":  sv,
					"configuration": map[string]any{},
				}
				if payMode, _ := cmd.Flags().GetString("pay-mode"); payMode != "" {
					body["configuration"].(map[string]any)["pay_mode"] = payMode
				}
				if months, _ := cmd.Flags().GetInt("duration-months"); months > 0 {
					body["configuration"].(map[string]any)["duration_months"] = months
				}
			}
			return runPOST("/v1/products/quote", body)(cmd, args)
		},
	}
	quote.Flags().String("body-file", "", "完整报价 JSON 文件路径")
	quote.Flags().String("product-type", "", "产品类型")
	quote.Flags().String("spec-version", "", "规格版本")
	quote.Flags().String("pay-mode", "monthly", "付费模式")
	quote.Flags().Int("duration-months", 1, "时长（月）")
	cmd.AddCommand(quote)
	buy := &cobra.Command{
		Use:   "buy",
		Short: "创建待付款订单（product.purchase.create）",
		RunE: func(cmd *cobra.Command, args []string) error {
			idempotencyKey, _ := cmd.Flags().GetString("idempotency-key")
			if strings.TrimSpace(idempotencyKey) == "" {
				return fail(newFactory(), fmt.Errorf("需要 --idempotency-key（UUID v4）"), cmdutil.ExitConfig)
			}
			bodyFile, _ := cmd.Flags().GetString("body-file")
			var body map[string]any
			if bodyFile != "" {
				raw, err := os.ReadFile(bodyFile)
				if err != nil {
					return fail(newFactory(), err, cmdutil.ExitConfig)
				}
				if err := json.Unmarshal(raw, &body); err != nil {
					return fail(newFactory(), fmt.Errorf("解析 body-file: %w", err), cmdutil.ExitConfig)
				}
			} else {
				quoteID, _ := cmd.Flags().GetString("quote-id")
				pt, _ := cmd.Flags().GetString("product-type")
				sv, _ := cmd.Flags().GetString("spec-version")
				if quoteID == "" || pt == "" || sv == "" {
					return fail(
						newFactory(),
						fmt.Errorf("需要 --body-file 或同时提供 --quote-id、--product-type 与 --spec-version"),
						cmdutil.ExitConfig,
					)
				}
				body = map[string]any{
					"quote_id":      quoteID,
					"product_type":  pt,
					"spec_version":  sv,
					"configuration": map[string]any{},
				}
				if payMode, _ := cmd.Flags().GetString("pay-mode"); payMode != "" {
					body["configuration"].(map[string]any)["pay_mode"] = payMode
				}
				if months, _ := cmd.Flags().GetInt("duration-months"); months > 0 {
					body["configuration"].(map[string]any)["duration_months"] = months
				}
			}
			headers := map[string]string{"Idempotency-Key": idempotencyKey}
			return runPOSTWithHeaders("/v1/products/purchases", body, headers)(cmd, args)
		},
	}
	buy.Flags().String("idempotency-key", "", "幂等键（UUID v4，必填）")
	buy.Flags().String("body-file", "", "完整下单 JSON 文件路径")
	buy.Flags().String("quote-id", "", "报价 ID")
	buy.Flags().String("product-type", "", "产品类型")
	buy.Flags().String("spec-version", "", "规格版本")
	buy.Flags().String("pay-mode", "monthly", "付费模式")
	buy.Flags().Int("duration-months", 1, "时长（月）")
	cmd.AddCommand(buy)
	return cmd
}

func newMsgCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "msg",
		Aliases: []string{"message", "messages"},
		Short:   "站内信（account.notifications.read）",
	}
	list := &cobra.Command{
		Use:   "list",
		Short: "站内信列表",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGETWithQuery("/v1/messages", listQuery(cmd))(cmd, args)
		},
	}
	addListFlags(list)
	cmd.AddCommand(list)
	cmd.AddCommand(&cobra.Command{
		Use:   "show [message_id]",
		Short: "站内信详情（仅 metadata）",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGET(fmt.Sprintf("/v1/messages/%s", args[0]))(cmd, args)
		},
	})
	return cmd
}

func newTicketCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ticket",
		Short: "工单 Capability",
	}
	preview := &cobra.Command{
		Use:   "preview",
		Short: "工单预览（support.ticket.preview）",
		RunE: func(cmd *cobra.Command, args []string) error {
			subject, _ := cmd.Flags().GetString("subject")
			body, _ := cmd.Flags().GetString("body")
			if subject == "" || body == "" {
				return fail(newFactory(), fmt.Errorf("需要 --subject 与 --body"), cmdutil.ExitConfig)
			}
			payload := map[string]any{
				"subject": subject,
				"body":    body,
			}
			if oid, _ := cmd.Flags().GetString("order-id"); oid != "" {
				payload["order_id"] = oid
			}
			return runPOST("/v1/tickets/preview", payload)(cmd, args)
		},
	}
	preview.Flags().String("subject", "", "工单标题")
	preview.Flags().String("body", "", "工单正文")
	preview.Flags().String("order-id", "", "关联订单 ID（可选）")
	cmd.AddCommand(preview)
	create := &cobra.Command{
		Use:   "create",
		Short: "创建真实工单（support.ticket.create）",
		RunE: func(cmd *cobra.Command, args []string) error {
			idempotencyKey, _ := cmd.Flags().GetString("idempotency-key")
			if strings.TrimSpace(idempotencyKey) == "" {
				return fail(newFactory(), fmt.Errorf("需要 --idempotency-key（UUID v4）"), cmdutil.ExitConfig)
			}
			bodyFile, _ := cmd.Flags().GetString("body-file")
			var payload map[string]any
			if bodyFile != "" {
				raw, err := os.ReadFile(bodyFile)
				if err != nil {
					return fail(newFactory(), err, cmdutil.ExitConfig)
				}
				if err := json.Unmarshal(raw, &payload); err != nil {
					return fail(newFactory(), fmt.Errorf("解析 body-file: %w", err), cmdutil.ExitConfig)
				}
			} else {
				ticketType, _ := cmd.Flags().GetString("type")
				subject, _ := cmd.Flags().GetString("subject")
				body, _ := cmd.Flags().GetString("body")
				tz, _ := cmd.Flags().GetString("timezone")
				start, _ := cmd.Flags().GetString("occurred-at-start")
				end, _ := cmd.Flags().GetString("occurred-at-end")
				if ticketType == "" || subject == "" || body == "" || tz == "" || start == "" || end == "" {
					return fail(
						newFactory(),
						fmt.Errorf("需要 --body-file 或 --type、--subject、--body、--timezone、--occurred-at-start、--occurred-at-end"),
						cmdutil.ExitConfig,
					)
				}
				payload = map[string]any{
					"ticket_type":       ticketType,
					"subject":           subject,
					"body":              body,
					"timezone":          tz,
					"occurred_at_start": start,
					"occurred_at_end":   end,
				}
				if oid, _ := cmd.Flags().GetString("order-id"); oid != "" {
					payload["order_id"] = oid
				}
			}
			headers := map[string]string{"Idempotency-Key": idempotencyKey}
			return runPOSTWithHeaders("/v1/tickets", payload, headers)(cmd, args)
		},
	}
	create.Flags().String("idempotency-key", "", "幂等键（UUID v4，必填）")
	create.Flags().String("body-file", "", "完整工单 JSON 文件路径")
	create.Flags().String("type", "", "工单类型：technical|billing|account|product|other")
	create.Flags().String("subject", "", "工单标题")
	create.Flags().String("body", "", "工单正文")
	create.Flags().String("order-id", "", "关联订单 ID（可选）")
	create.Flags().String("timezone", "Asia/Shanghai", "时区")
	create.Flags().String("occurred-at-start", "", "问题发生起始时间（RFC3339）")
	create.Flags().String("occurred-at-end", "", "问题发生结束时间（RFC3339）")
	cmd.AddCommand(create)
	return cmd
}
