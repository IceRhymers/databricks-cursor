package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
)

// Config holds the persisted port and profile settings.
type Config struct {
	Port    int    `json:"port"`
	Profile string `json:"profile"`
}

// configDir returns the path to ~/.databricks-cursor, creating it if needed.
func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	dir := filepath.Join(home, ".databricks-cursor")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("cannot create config directory: %w", err)
	}
	return dir, nil
}

// configPath returns the full path to the config file.
func configPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// loadConfig reads the config file. Returns zero Config if the file doesn't exist.
func loadConfig() (Config, error) {
	path, err := configPath()
	if err != nil {
		return Config{}, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return Config{}, nil
	}
	if err != nil {
		return Config{}, fmt.Errorf("cannot read config: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("cannot parse config: %w", err)
	}
	return cfg, nil
}

// saveConfig writes the config to disk.
func saveConfig(cfg Config) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("cannot marshal config: %w", err)
	}
	return os.WriteFile(path, data, 0o600)
}

// isPortAvailable checks whether a TCP port is free on 127.0.0.1.
func isPortAvailable(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return false
	}
	ln.Close()
	return true
}

// LoadOrCreate resolves the effective port and profile. If portFlag is 0 and a
// saved port exists, it reuses the saved port. If the saved port is in use, it
// returns a descriptive error. The resolved config is always saved back to disk.
func LoadOrCreate(portFlag int, profileFlag string) (Config, error) {
	saved, err := loadConfig()
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		Port:    portFlag,
		Profile: profileFlag,
	}

	if cfg.Port == 0 {
		if saved.Port != 0 {
			// Reuse saved port.
			if !isPortAvailable(saved.Port) {
				return Config{}, fmt.Errorf("port %d already in use — is another databricks-cursor running?", saved.Port)
			}
			cfg.Port = saved.Port
		}
		// If saved.Port is also 0, we leave cfg.Port as 0 — caller will auto-assign.
	}

	if err := saveConfig(cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}
