package output

import (
	"os"
	"strings"
)

// 常见 ANSI 转义序列；仅在 ColorEnabled 且非 JSON 模式下使用。
const (
	ansiReset = "\033[0m"
	ansiBold  = "\033[1m"
	ansiDim   = "\033[2m"
	ansiRed   = "\033[31m"
	ansiGreen = "\033[32m"
	ansiCyan  = "\033[36m"
)

// ResolveColor 决定是否启用终端颜色。
// 优先级：NO_COLOR（任意非空值）> --color never > --color always > auto（TTY 检测）。
func ResolveColor(mode string, stdoutIsTTY bool) bool {
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "never":
		return false
	case "always":
		return true
	default:
		return stdoutIsTTY
	}
}

// IsStdoutTTY 判断 stdout 是否为交互终端（用于 auto 模式）。
func IsStdoutTTY() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

func (w *Writer) paint(code, s string) string {
	if !w.ColorEnabled || w.Format == ModeJSON {
		return s
	}
	return code + s + ansiReset
}

func (w *Writer) bold(s string) string  { return w.paint(ansiBold, s) }
func (w *Writer) dim(s string) string   { return w.paint(ansiDim, s) }
func (w *Writer) green(s string) string { return w.paint(ansiGreen, s) }
func (w *Writer) cyan(s string) string  { return w.paint(ansiCyan, s) }
func (w *Writer) red(s string) string   { return w.paint(ansiRed, s) }

// Green 成功态文本着色（JSON 模式或无颜色时原样返回）。
func (w *Writer) Green(s string) string { return w.green(s) }

// Dim 次要文本着色（JSON 模式或无颜色时原样返回）。
func (w *Writer) Dim(s string) string { return w.dim(s) }
