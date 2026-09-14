package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"
)

const updatePackage = "@kuaidaili/kdl-agent"

var releaseVersionPattern = regexp.MustCompile(`^\d+\.\d+\.\d+(?:-beta\.\d+)?$`)

type updateResult struct {
	CurrentVersion string `json:"current_version"`
	LatestVersion  string `json:"latest_version,omitempty"`
	Channel        string `json:"channel"`
	Status         string `json:"status"`
	Message        string `json:"message"`
	Package        string `json:"package"`
}

func checkUpdate(ctx context.Context, current, channel string, client *http.Client, endpoint string) updateResult {
	result := updateResult{CurrentVersion: current, Channel: channel, Package: updatePackage, Status: "unavailable", Message: "暂时无法检查更新，可继续使用现有版本，稍后重试。"}
	version := "v" + strings.TrimPrefix(current, "v")
	if !semver.IsValid(version) || strings.Contains(semver.Prerelease(version), "dev") {
		result.Status, result.Message = "development", "开发构建不参与自动版本检查。"
		return result
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+channel, nil)
	if err != nil {
		return result
	}
	resp, err := client.Do(req)
	if err != nil {
		return result
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return result
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 65537))
	if err != nil || len(data) > 65536 {
		return result
	}
	var metadata struct {
		Version string `json:"version"`
		Name    string `json:"name"`
	}
	if json.Unmarshal(data, &metadata) != nil || metadata.Name != updatePackage {
		return result
	}
	latest := "v" + metadata.Version
	if !releaseVersionPattern.MatchString(metadata.Version) || !semver.IsValid(latest) || semver.Canonical(latest) != latest || (channel == "latest" && semver.Prerelease(latest) != "") {
		return result
	}
	result.LatestVersion = metadata.Version
	result.Status, result.Message = "up_to_date", "当前版本无需升级。"
	if semver.Compare(latest, version) > 0 {
		result.Status, result.Message = "update_available", "发现新版；用户确认后使用目标版本 npm 向导同步升级 CLI 和 Skill。"
	}
	return result
}

func newUpdateCmd() *cobra.Command {
	parent := &cobra.Command{Use: "update", Short: "检查新版；升级通过外部 npm 向导执行"}
	var channel string
	check := &cobra.Command{Use: "check", Short: "匿名检查更新，不修改本机安装或凭证", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		selected := channel
		if selected == "" {
			selected = "latest"
			if strings.Contains(Version, "-beta.") {
				selected = "beta"
			}
		}
		if selected != "beta" && selected != "latest" {
			return fmt.Errorf("渠道无效；请使用 --channel beta 或 latest")
		}
		client := &http.Client{Timeout: 5 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
		result := checkUpdate(cmd.Context(), Version, selected, client, "https://registry.npmjs.org/@kuaidaili%2Fkdl-agent/")
		if globalFormat == "json" {
			return writeJSON(newFactory().Out, result)
		}
		if !globalQuiet {
			fmt.Fprintf(cmd.OutOrStdout(), "%s\n当前：%s；渠道：%s；推荐：%s\n", result.Message, result.CurrentVersion, result.Channel, result.LatestVersion)
		}
		return nil
	}}
	check.Flags().StringVar(&channel, "channel", "", "检查渠道 beta|latest（默认跟随当前版本）")
	parent.AddCommand(check)
	return parent
}
