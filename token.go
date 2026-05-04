package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// execCommandFunc is overridable for testing.
var execCommandFunc = exec.Command

// tokenResponse matches the JSON output of `databricks auth token`.
type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresOn   int64  `json:"expires_on"`
}

// databricksFetcher implements tokencache.TokenFetcher by shelling out to the
// Databricks CLI to fetch a fresh OAuth token.
type databricksFetcher struct {
	profile string
}

// FetchToken runs `databricks auth token --profile <profile>` and parses the result.
func (f *databricksFetcher) FetchToken(ctx context.Context) (string, time.Time, error) {
	cmd := execCommandFunc("databricks", "auth", "token", "--profile", f.profile)
	out, err := cmd.Output()
	if err != nil {
		return "", time.Time{}, fmt.Errorf("databricks auth token failed: %w", err)
	}
	var resp tokenResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		return "", time.Time{}, fmt.Errorf("cannot parse token response: %w", err)
	}
	if resp.AccessToken == "" {
		return "", time.Time{}, fmt.Errorf("empty access_token in response")
	}
	expiry := time.Unix(resp.ExpiresOn, 0)
	return resp.AccessToken, expiry, nil
}

// DiscoverHost reads ~/.databrickscfg to find the host for the given profile.
func DiscoverHost(profile string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	cfgPath := filepath.Join(home, ".databrickscfg")
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return "", fmt.Errorf("cannot read %s: %w", cfgPath, err)
	}
	return parseHostFromCfg(string(data), profile)
}

// parseHostFromCfg extracts the host value for a given profile from databrickscfg content.
func parseHostFromCfg(content, profile string) (string, error) {
	lines := strings.Split(content, "\n")
	sectionHeader := fmt.Sprintf("[%s]", profile)
	inSection := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check for section header.
		if strings.HasPrefix(trimmed, "[") {
			inSection = strings.EqualFold(trimmed, sectionHeader)
			continue
		}

		if inSection && strings.HasPrefix(trimmed, "host") {
			parts := strings.SplitN(trimmed, "=", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1]), nil
			}
		}
	}

	return "", fmt.Errorf("profile %q not found in databrickscfg", profile)
}

// ConstructGatewayURL builds the AI Gateway URL from a workspace host.
// Input:  https://my-workspace.cloud.databricks.com
// Output: https://my-workspace.cloud.databricks.com/ai-gateway/cursor/v1
func ConstructGatewayURL(host string) string {
	return strings.TrimRight(host, "/") + "/ai-gateway/cursor/v1"
}
