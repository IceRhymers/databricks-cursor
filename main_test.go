package main

import (
	"testing"
)

func TestVersionIsSet(t *testing.T) {
	// Version should have a default value.
	if Version == "" {
		t.Error("Version should not be empty")
	}
	if Version != "dev" {
		t.Logf("Version is %q (expected 'dev' in test builds)", Version)
	}
}

func TestConstructGatewayURL_Integration(t *testing.T) {
	// Verify the gateway URL format matches what pkg/proxy expects.
	url := ConstructGatewayURL("https://workspace-123.cloud.databricks.com")
	expected := "https://workspace-123.ai-gateway.cloud.databricks.com/anthropic"
	if url != expected {
		t.Errorf("got %q, want %q", url, expected)
	}
}
