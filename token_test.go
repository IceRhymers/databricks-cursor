package main

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestDatabricksFetcher_FetchToken(t *testing.T) {
	// Mock execCommandFunc to return a fake token response.
	origExec := execCommandFunc
	defer func() { execCommandFunc = origExec }()

	expiresOn := time.Now().Add(1 * time.Hour).Unix()
	resp := tokenResponse{
		AccessToken: "test-token-123",
		ExpiresOn:   expiresOn,
	}
	respJSON, _ := json.Marshal(resp)

	execCommandFunc = func(name string, args ...string) *exec.Cmd {
		// Use a helper process pattern: run "echo" with the JSON.
		cmd := exec.Command("echo", string(respJSON))
		return cmd
	}

	f := &databricksFetcher{profile: "TEST"}
	token, expiry, err := f.FetchToken(context.Background())
	if err != nil {
		t.Fatalf("FetchToken failed: %v", err)
	}
	if token != "test-token-123" {
		t.Errorf("expected token test-token-123, got %s", token)
	}
	if expiry.Unix() != expiresOn {
		t.Errorf("expected expiry %d, got %d", expiresOn, expiry.Unix())
	}
}

func TestDatabricksFetcher_FetchToken_Error(t *testing.T) {
	origExec := execCommandFunc
	defer func() { execCommandFunc = origExec }()

	execCommandFunc = func(name string, args ...string) *exec.Cmd {
		return exec.Command("false") // exits with error
	}

	f := &databricksFetcher{profile: "BAD"}
	_, _, err := f.FetchToken(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestParseHostFromCfg(t *testing.T) {
	cfg := `[DEFAULT]
host = https://my-workspace.cloud.databricks.com
token = dapi123

[STAGING]
host = https://staging.cloud.databricks.com
`
	tests := []struct {
		profile  string
		wantHost string
		wantErr  bool
	}{
		{"DEFAULT", "https://my-workspace.cloud.databricks.com", false},
		{"STAGING", "https://staging.cloud.databricks.com", false},
		{"MISSING", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.profile, func(t *testing.T) {
			host, err := parseHostFromCfg(cfg, tt.profile)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if host != tt.wantHost {
				t.Errorf("expected host %q, got %q", tt.wantHost, host)
			}
		})
	}
}

func TestDiscoverHost(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	cfgContent := `[DEFAULT]
host = https://test-workspace.cloud.databricks.com
`
	os.WriteFile(filepath.Join(tmpHome, ".databrickscfg"), []byte(cfgContent), 0o644)

	host, err := DiscoverHost("DEFAULT")
	if err != nil {
		t.Fatalf("DiscoverHost failed: %v", err)
	}
	if host != "https://test-workspace.cloud.databricks.com" {
		t.Errorf("unexpected host: %s", host)
	}
}

func TestConstructGatewayURL(t *testing.T) {
	tests := []struct {
		name string
		host string
		want string
	}{
		{
			name: "plain host",
			host: "https://my-workspace.cloud.databricks.com",
			want: "https://my-workspace.cloud.databricks.com/ai-gateway/cursor/v1",
		},
		{
			name: "trailing slash trimmed",
			host: "https://abc123.cloud.databricks.com/",
			want: "https://abc123.cloud.databricks.com/ai-gateway/cursor/v1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConstructGatewayURL(tt.host)
			if got != tt.want {
				t.Errorf("ConstructGatewayURL(%q) = %q, want %q", tt.host, got, tt.want)
			}
		})
	}
}
