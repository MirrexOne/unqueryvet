package dsl

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Parser handles parsing of DSL configuration files.
type Parser struct{}

// NewParser creates a new DSL parser.
func NewParser() *Parser {
	return &Parser{}
}

// ParseFile parses a DSL configuration from a file.
func (p *Parser) ParseFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	return p.Parse(data)
}

// Parse parses a DSL configuration from bytes.
func (p *Parser) Parse(data []byte) (*Config, error) {
	var config Config

	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Apply defaults
	p.applyDefaults(&config)

	// Validate
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &config, nil
}

// FindConfigFile searches for a configuration file in standard locations.
// Returns empty string if no config file is found.
func (p *Parser) FindConfigFile(startDir string) string {
	configNames := []string{
		".unqueryvet.yaml",
		".unqueryvet.yml",
		"unqueryvet.yaml",
		"unqueryvet.yml",
	}

	dir := startDir
	for {
		for _, name := range configNames {
			path := filepath.Join(dir, name)
			if _, err := os.Stat(path); err == nil {
				return path
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return ""
}

// LoadConfig loads configuration from the first found config file.
// If no config file is found, returns a default configuration.
func (p *Parser) LoadConfig(startDir string) (*Config, error) {
	configPath := p.FindConfigFile(startDir)
	if configPath == "" {
		return p.DefaultConfig(), nil
	}

	return p.ParseFile(configPath)
}

// DefaultConfig returns a sensible default configuration.
func (p *Parser) DefaultConfig() *Config {
	return &Config{
		Rules: map[string]Severity{
			"select-star":   SeverityWarning,
			"n1-queries":    SeverityWarning,
			"sql-injection": SeverityError,
		},
		Ignore: []string{
			"*_test.go",
			"testdata/**",
			"vendor/**",
		},
		Allow: []string{
			"COUNT(*)",
		},
	}
}

// MergeConfigs merges multiple configs, with later configs taking precedence.
func (p *Parser) MergeConfigs(configs ...*Config) *Config {
	result := &Config{
		Rules: make(map[string]Severity),
	}

	for _, cfg := range configs {
		if cfg == nil {
			continue
		}

		// Merge rules
		maps.Copy(result.Rules, cfg.Rules)

		// Append ignore patterns
		result.Ignore = append(result.Ignore, cfg.Ignore...)

		// Append allow patterns
		result.Allow = append(result.Allow, cfg.Allow...)

		// Append custom rules
		result.CustomRules = append(result.CustomRules, cfg.CustomRules...)

		// Merge legacy config (later takes precedence)
		if cfg.CheckSQLBuilders {
			result.CheckSQLBuilders = true
		}
		result.AllowedPatterns = append(result.AllowedPatterns, cfg.AllowedPatterns...)
		result.IgnoredFiles = append(result.IgnoredFiles, cfg.IgnoredFiles...)
	}

	return result
}

// applyDefaults applies default values to the configuration.
func (p *Parser) applyDefaults(config *Config) {
	// Set default severity for rules without severity
	for i := range config.CustomRules {
		rule := &config.CustomRules[i]
		if rule.Severity == "" {
			rule.Severity = SeverityWarning
		}
		if rule.Action == "" {
			rule.Action = ActionReport
		}
	}

	// Merge legacy config into new format
	if len(config.AllowedPatterns) > 0 && len(config.Allow) == 0 {
		config.Allow = config.AllowedPatterns
	}
	if len(config.IgnoredFiles) > 0 && len(config.Ignore) == 0 {
		config.Ignore = config.IgnoredFiles
	}
}
