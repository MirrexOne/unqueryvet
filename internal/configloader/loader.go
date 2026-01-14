// Package configloader provides functionality to load configuration from .unqueryvet.yaml file.
package configloader

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/MirrexOne/unqueryvet/pkg/config"
)

const (
	// ConfigFileName is the default configuration file name
	ConfigFileName = ".unqueryvet.yaml"
	// AlternateConfigFileName is an alternate configuration file name
	AlternateConfigFileName = ".unqueryvet.yml"
)

// LoadConfig loads configuration from a YAML file.
// It starts with default settings and overlays values from the file.
func LoadConfig(path string) (*config.UnqueryvetSettings, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Start with default settings so unspecified fields use defaults
	cfg := config.DefaultSettings()
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &cfg, nil
}

// FindConfig searches for a configuration file in the current directory and parent directories.
func FindConfig() (string, error) {
	// Start from current directory
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Search up the directory tree
	for {
		// Try primary name
		configPath := filepath.Join(dir, ConfigFileName)
		if _, err := os.Stat(configPath); err == nil {
			return configPath, nil
		}

		// Try alternate name
		configPath = filepath.Join(dir, AlternateConfigFileName)
		if _, err := os.Stat(configPath); err == nil {
			return configPath, nil
		}

		// Move to parent directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root, no config found
			break
		}
		dir = parent
	}

	return "", nil // No config found, not an error
}

// LoadOrDefault loads configuration from file or returns default settings.
func LoadOrDefault(configPath string) (*config.UnqueryvetSettings, error) {
	// If explicit path is provided, use it
	if configPath != "" {
		cfg, err := LoadConfig(configPath)
		if err != nil {
			return nil, err
		}
		return cfg, nil
	}

	// Try to find config file automatically
	foundPath, err := FindConfig()
	if err != nil {
		return nil, err
	}

	if foundPath != "" {
		cfg, err := LoadConfig(foundPath)
		if err != nil {
			// If config file exists but is invalid, return error
			return nil, err
		}
		return cfg, nil
	}

	// No config found, use defaults
	defaults := config.DefaultSettings()
	return &defaults, nil
}

// ValidateConfig validates the configuration.
func ValidateConfig(cfg *config.UnqueryvetSettings) error {
	// Validate severity
	if cfg.Severity != "" && cfg.Severity != "error" && cfg.Severity != "warning" {
		return fmt.Errorf("invalid severity: %s (must be 'error' or 'warning')", cfg.Severity)
	}

	// Validate allowed patterns (compile as regex)
	for _, pattern := range cfg.AllowedPatterns {
		if pattern == "" {
			return fmt.Errorf("empty pattern in allowed-patterns")
		}
	}

	return nil
}
