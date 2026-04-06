package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadOrCreate_NewConfig(t *testing.T) {
	// Use a temp dir as home to avoid touching real config.
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	cfg, err := LoadOrCreate(0, "DEFAULT")
	if err != nil {
		t.Fatalf("LoadOrCreate failed: %v", err)
	}
	if cfg.Profile != "DEFAULT" {
		t.Errorf("expected profile DEFAULT, got %s", cfg.Profile)
	}
	// Port should be 0 (auto-assign) since no saved config exists.
	if cfg.Port != 0 {
		t.Errorf("expected port 0 for new config, got %d", cfg.Port)
	}
}

func TestLoadOrCreate_ReuseSavedPort(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	// Pre-create a config with a known port.
	dir := filepath.Join(tmpHome, ".databricks-cursor")
	os.MkdirAll(dir, 0o755)
	cfg := Config{Port: 19876, Profile: "DEFAULT"}
	data, _ := json.Marshal(cfg)
	os.WriteFile(filepath.Join(dir, "config.json"), data, 0o644)

	result, err := LoadOrCreate(0, "DEFAULT")
	if err != nil {
		t.Fatalf("LoadOrCreate failed: %v", err)
	}
	if result.Port != 19876 {
		t.Errorf("expected reused port 19876, got %d", result.Port)
	}
}

func TestLoadOrCreate_OverridePort(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	cfg, err := LoadOrCreate(8080, "STAGING")
	if err != nil {
		t.Fatalf("LoadOrCreate failed: %v", err)
	}
	if cfg.Port != 8080 {
		t.Errorf("expected port 8080, got %d", cfg.Port)
	}
	if cfg.Profile != "STAGING" {
		t.Errorf("expected profile STAGING, got %s", cfg.Profile)
	}
}

func TestLoadOrCreate_PortInUse(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	// Bind a port so it's in use.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("cannot bind listener: %v", err)
	}
	defer ln.Close()
	usedPort := ln.Addr().(*net.TCPAddr).Port

	// Save that port in config.
	dir := filepath.Join(tmpHome, ".databricks-cursor")
	os.MkdirAll(dir, 0o755)
	cfg := Config{Port: usedPort, Profile: "DEFAULT"}
	data, _ := json.Marshal(cfg)
	os.WriteFile(filepath.Join(dir, "config.json"), data, 0o644)

	_, err = LoadOrCreate(0, "DEFAULT")
	if err == nil {
		t.Fatal("expected error for port in use, got nil")
	}
	expected := fmt.Sprintf("port %d already in use", usedPort)
	if got := err.Error(); !contains(got, expected) {
		t.Errorf("expected error containing %q, got %q", expected, got)
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	original := Config{Port: 12345, Profile: "TEST"}
	if err := saveConfig(original); err != nil {
		t.Fatalf("saveConfig failed: %v", err)
	}

	loaded, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig failed: %v", err)
	}
	if loaded.Port != original.Port || loaded.Profile != original.Profile {
		t.Errorf("loaded config %+v doesn't match original %+v", loaded, original)
	}
}

func TestIsPortAvailable(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("cannot bind: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port

	if isPortAvailable(port) {
		t.Error("expected port to be unavailable while listener is active")
	}

	ln.Close()

	if !isPortAvailable(port) {
		t.Error("expected port to be available after listener closed")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
