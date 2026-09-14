// Package output 处理人读与 JSON 两种输出模式。
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/kuaidaili/kdl-agent-cli/internal/client"
)

// Mode 输出格式。
type Mode string

const (
	ModeTable Mode = "table"
	ModeJSON  Mode = "json"
)

// Writer CLI 输出目标。
type Writer struct {
	Stdout       io.Writer
	Stderr       io.Writer
	Format       Mode
	Quiet        bool
	ColorEnabled bool
}

// NewWriter 默认 stdout/stderr。
func NewWriter(format Mode) *Writer {
	return &Writer{
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Format: format,
	}
}

// PrintJSON 输出任意 JSON 到 stdout（--format json 模式）。
func (w *Writer) PrintJSON(v any) error {
	enc := json.NewEncoder(w.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// PrintEnvelope 按格式输出 Gateway 成功响应。
func (w *Writer) PrintEnvelope(resp *client.GatewayResponse) error {
	if w.Format == ModeJSON {
		enc := json.NewEncoder(w.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}
	if len(resp.Data) == 0 || string(resp.Data) == "null" {
		fmt.Fprintln(w.Stdout, w.dim("(无数据)"))
		return nil
	}

	var generic any
	if err := json.Unmarshal(resp.Data, &generic); err != nil {
		fmt.Fprintln(w.Stdout, string(resp.Data))
		return nil
	}
	return w.printGeneric(generic)
}

func (w *Writer) printGeneric(v any) error {
	switch val := v.(type) {
	case map[string]any:
		if items, ok := val["items"].([]any); ok {
			return w.printItemsTable(items)
		}
		return w.printMap(val)
	case []any:
		return w.printItemsTable(val)
	default:
		raw, _ := json.MarshalIndent(val, "", "  ")
		fmt.Fprintln(w.Stdout, string(raw))
	}
	return nil
}

func (w *Writer) printMap(m map[string]any) error {
	tw := tabwriter.NewWriter(w.Stdout, 0, 4, 2, ' ', 0)
	keys := sortedKeys(m)
	for _, k := range keys {
		fmt.Fprintf(tw, "%s\t%v\n", k, formatValue(m[k]))
	}
	return tw.Flush()
}

func (w *Writer) printItemsTable(items []any) error {
	if len(items) == 0 {
		fmt.Fprintln(w.Stdout, w.dim("(空列表)"))
		return nil
	}
	first, ok := items[0].(map[string]any)
	if !ok {
		raw, _ := json.MarshalIndent(items, "", "  ")
		fmt.Fprintln(w.Stdout, string(raw))
		return nil
	}
	cols := sortedKeys(first)
	tw := tabwriter.NewWriter(w.Stdout, 0, 4, 2, ' ', 0)
	headerCells := make([]string, len(cols))
	for i, c := range cols {
		headerCells[i] = w.cyan(c)
	}
	fmt.Fprintln(tw, strings.Join(headerCells, "\t"))
	for _, item := range items {
		row, ok := item.(map[string]any)
		if !ok {
			continue
		}
		cells := make([]string, len(cols))
		for i, c := range cols {
			cells[i] = formatValue(row[c])
		}
		fmt.Fprintln(tw, strings.Join(cells, "\t"))
	}
	return tw.Flush()
}

func formatValue(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case float64:
		if t == float64(int64(t)) {
			return fmt.Sprintf("%d", int64(t))
		}
		return fmt.Sprintf("%v", t)
	default:
		raw, err := json.Marshal(t)
		if err != nil {
			return fmt.Sprintf("%v", t)
		}
		s := string(raw)
		if len(s) > 80 {
			return s[:77] + "..."
		}
		return s
	}
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	// 简单排序：优先常见标识字段
	priority := []string{"order_id", "id", "title", "product_type", "status", "name"}
	ordered := make([]string, 0, len(keys))
	seen := map[string]bool{}
	for _, p := range priority {
		if _, ok := m[p]; ok {
			ordered = append(ordered, p)
			seen[p] = true
		}
	}
	rest := make([]string, 0)
	for _, k := range keys {
		if !seen[k] {
			rest = append(rest, k)
		}
	}
	for i := 0; i < len(rest); i++ {
		for j := i + 1; j < len(rest); j++ {
			if rest[j] < rest[i] {
				rest[i], rest[j] = rest[j], rest[i]
			}
		}
	}
	return append(ordered, rest...)
}

// PrintError 向 stderr 输出用户可恢复错误（不含 token）。
func (w *Writer) PrintError(err error) {
	fmt.Fprintln(w.Stderr, w.red("错误:"), err)
}
