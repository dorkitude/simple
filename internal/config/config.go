package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	configDirEnv   = "DNSIMPLE_CONFIG_DIR"
	tokenFileName  = "token"
	configFileName = "config.json"
)

var configDirOverride string

// Config holds the persisted configuration for dnsimplectl.
type Config struct {
	AccountID string `json:"account_id,omitempty"`
	Sandbox   bool   `json:"sandbox,omitempty"`
}

// SetConfigDir overrides the config directory for the current process.
// This is primarily used by the TUI auth flow before credentials are written.
func SetConfigDir(path string) {
	configDirOverride = strings.TrimSpace(path)
}

func defaultConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "dnsimplectl"), nil
}

func cleanConfigPath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("config directory cannot be empty")
	}

	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		switch path {
		case "~":
			path = home
		default:
			if strings.HasPrefix(path, "~/") || strings.HasPrefix(path, "~\\") {
				path = filepath.Join(home, path[2:])
			}
		}
	}

	return filepath.Clean(path), nil
}

func resolveConfigDir() (string, error) {
	if configDirOverride != "" {
		return cleanConfigPath(configDirOverride)
	}
	if env := strings.TrimSpace(os.Getenv(configDirEnv)); env != "" {
		return cleanConfigPath(env)
	}
	return defaultConfigDir()
}

// ResolveConfigDir returns the resolved config directory path without creating it.
func ResolveConfigDir() (string, error) {
	return resolveConfigDir()
}

// configDir returns the resolved config directory, creating it if needed.
func configDir() (string, error) {
	dir, err := resolveConfigDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return dir, nil
}

// ConfigDir returns the config directory path (public accessor).
func ConfigDir() (string, error) {
	return configDir()
}

// TokenPath returns the path to the token file.
func TokenPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, tokenFileName), nil
}

// LoadToken reads the API token from disk.
func LoadToken() (string, error) {
	path, err := TokenPath()
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("not authenticated — run 'simple auth login' first")
	}
	token := string(data)
	if token == "" {
		return "", fmt.Errorf("token file is empty — run 'simple auth login'")
	}
	return token, nil
}

// SaveToken writes the API token to disk with 0600 permissions.
func SaveToken(token string) error {
	path, err := TokenPath()
	if err != nil {
		return err
	}
	return os.WriteFile(path, []byte(token), 0600)
}

// RemoveToken deletes the stored token.
func RemoveToken() error {
	path, err := TokenPath()
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}
	return os.Remove(path)
}

// HasToken returns true if a token file exists.
func HasToken() bool {
	path, err := TokenPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

// ConfigPath returns the path to config.json.
func ConfigPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, configFileName), nil
}

// Load reads the config from disk.
func Load() (*Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Save writes the config to disk.
func Save(cfg *Config) error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}
