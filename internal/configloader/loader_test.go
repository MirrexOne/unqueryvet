package configloader

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/MirrexOne/unqueryvet/pkg/config"
)

func TestConfigFileName(t *testing.T) {
	if ConfigFileName != ".unqueryvet.yaml" {
		t.Errorf("ConfigFileName = %s, want .unqueryvet.yaml", ConfigFileName)
	}
	if AlternateConfigFileName != ".unqueryvet.yml" {
		t.Errorf("AlternateConfigFileName = %s, want .unqueryvet.yml", AlternateConfigFileName)
	}
}

func TestLoadConfig(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	// Test valid config
	t.Run("valid config", func(t *testing.T) {
		configPath := filepath.Join(tmpDir, "valid.yaml")
		content := `
check-sql-builders: true
severity: warning
check-aliased-wildcard: true
allowed-patterns:
  - "(?i)COUNT\\(\\s*\\*\\s*\\)"
ignored-functions:
  - "debug.*"
ignored-files:
  - "*_test.go"
`
		if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}

		cfg, err := LoadConfig(configPath)
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}

		if !cfg.CheckSQLBuilders {
			t.Error("CheckSQLBuilders should be true")
		}
		if cfg.Severity != "warning" {
			t.Errorf("Severity = %s, want warning", cfg.Severity)
		}
		if !cfg.CheckAliasedWildcard {
			t.Error("CheckAliasedWildcard should be true")
		}
		if len(cfg.AllowedPatterns) != 1 {
			t.Errorf("AllowedPatterns len = %d, want 1", len(cfg.AllowedPatterns))
		}
		if len(cfg.IgnoredFunctions) != 1 {
			t.Errorf("IgnoredFunctions len = %d, want 1", len(cfg.IgnoredFunctions))
		}
		if len(cfg.IgnoredFiles) != 1 {
			t.Errorf("IgnoredFiles len = %d, want 1", len(cfg.IgnoredFiles))
		}
	})

	// Test file not found
	t.Run("file not found", func(t *testing.T) {
		_, err := LoadConfig(filepath.Join(tmpDir, "nonexistent.yaml"))
		if err == nil {
			t.Error("LoadConfig() should return error for nonexistent file")
		}
	})

	// Test invalid YAML
	t.Run("invalid yaml", func(t *testing.T) {
		configPath := filepath.Join(tmpDir, "invalid.yaml")
		content := `
check-sql-builders: [invalid yaml content
`
		if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}

		_, err := LoadConfig(configPath)
		if err == nil {
			t.Error("LoadConfig() should return error for invalid YAML")
		}
	})

	// Test empty config
	t.Run("empty config", func(t *testing.T) {
		configPath := filepath.Join(tmpDir, "empty.yaml")
		if err := os.WriteFile(configPath, []byte(""), 0o644); err != nil {
			t.Fatal(err)
		}

		cfg, err := LoadConfig(configPath)
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}

		// Empty config should use default values
		if !cfg.CheckSQLBuilders {
			t.Error("Empty config should have CheckSQLBuilders = true (default)")
		}
	})

	// Test config with SQL builders
	t.Run("sql builders config", func(t *testing.T) {
		configPath := filepath.Join(tmpDir, "builders.yaml")
		content := `
sql-builders:
  squirrel: true
  gorm: false
  sqlx: true
`
		if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}

		cfg, err := LoadConfig(configPath)
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}

		if !cfg.SQLBuilders.Squirrel {
			t.Error("SQLBuilders.Squirrel should be true")
		}
		if cfg.SQLBuilders.GORM {
			t.Error("SQLBuilders.GORM should be false")
		}
		if !cfg.SQLBuilders.SQLx {
			t.Error("SQLBuilders.SQLx should be true")
		}
	})
}

func TestFindConfig(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "sub", "dir")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Save and restore working directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	t.Run("no config", func(t *testing.T) {
		if err := os.Chdir(subDir); err != nil {
			t.Fatal(err)
		}

		_, err := FindConfig()
		if err != nil {
			t.Fatalf("FindConfig() error = %v", err)
		}
		// No config found is not an error, just empty path
		// (actual result depends on parent directories)
	})

	t.Run("config in current dir", func(t *testing.T) {
		configPath := filepath.Join(subDir, ConfigFileName)
		if err := os.WriteFile(configPath, []byte("severity: warning"), 0o644); err != nil {
			t.Fatal(err)
		}
		defer os.Remove(configPath)

		if err := os.Chdir(subDir); err != nil {
			t.Fatal(err)
		}

		path, err := FindConfig()
		if err != nil {
			t.Fatalf("FindConfig() error = %v", err)
		}
		if path != configPath {
			t.Errorf("FindConfig() = %s, want %s", path, configPath)
		}
	})

	t.Run("config in parent dir", func(t *testing.T) {
		configPath := filepath.Join(tmpDir, ConfigFileName)
		if err := os.WriteFile(configPath, []byte("severity: error"), 0o644); err != nil {
			t.Fatal(err)
		}
		defer os.Remove(configPath)

		if err := os.Chdir(subDir); err != nil {
			t.Fatal(err)
		}

		path, err := FindConfig()
		if err != nil {
			t.Fatalf("FindConfig() error = %v", err)
		}
		if path != configPath {
			t.Errorf("FindConfig() = %s, want %s", path, configPath)
		}
	})

	t.Run("alternate config name", func(t *testing.T) {
		configPath := filepath.Join(subDir, AlternateConfigFileName)
		if err := os.WriteFile(configPath, []byte("severity: warning"), 0o644); err != nil {
			t.Fatal(err)
		}
		defer os.Remove(configPath)

		if err := os.Chdir(subDir); err != nil {
			t.Fatal(err)
		}

		path, err := FindConfig()
		if err != nil {
			t.Fatalf("FindConfig() error = %v", err)
		}
		if path != configPath {
			t.Errorf("FindConfig() = %s, want %s", path, configPath)
		}
	})
}

func TestLoadOrDefault(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("explicit path", func(t *testing.T) {
		configPath := filepath.Join(tmpDir, "config.yaml")
		content := `severity: error`
		if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}

		cfg, err := LoadOrDefault(configPath)
		if err != nil {
			t.Fatalf("LoadOrDefault() error = %v", err)
		}
		if cfg.Severity != "error" {
			t.Errorf("Severity = %s, want error", cfg.Severity)
		}
	})

	t.Run("explicit path not found", func(t *testing.T) {
		_, err := LoadOrDefault(filepath.Join(tmpDir, "nonexistent.yaml"))
		if err == nil {
			t.Error("LoadOrDefault() should return error for nonexistent explicit path")
		}
	})

	t.Run("no config uses defaults", func(t *testing.T) {
		// Save and restore working directory
		origDir, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Chdir(origDir) }()

		emptyDir := filepath.Join(tmpDir, "empty")
		if err := os.MkdirAll(emptyDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Chdir(emptyDir); err != nil {
			t.Fatal(err)
		}

		cfg, err := LoadOrDefault("")
		if err != nil {
			t.Fatalf("LoadOrDefault() error = %v", err)
		}

		defaults := config.DefaultSettings()
		if cfg.CheckSQLBuilders != defaults.CheckSQLBuilders {
			t.Error("Should use default CheckSQLBuilders")
		}
	})
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *config.UnqueryvetSettings
		wantErr bool
	}{
		{
			name: "valid config with warning",
			cfg: &config.UnqueryvetSettings{
				Severity: "warning",
			},
			wantErr: false,
		},
		{
			name: "valid config with error",
			cfg: &config.UnqueryvetSettings{
				Severity: "error",
			},
			wantErr: false,
		},
		{
			name: "valid config with empty severity",
			cfg: &config.UnqueryvetSettings{
				Severity: "",
			},
			wantErr: false,
		},
		{
			name: "invalid severity",
			cfg: &config.UnqueryvetSettings{
				Severity: "invalid",
			},
			wantErr: true,
		},
		{
			name: "empty pattern in allowed-patterns",
			cfg: &config.UnqueryvetSettings{
				AllowedPatterns: []string{"valid", ""},
			},
			wantErr: true,
		},
		{
			name: "valid patterns",
			cfg: &config.UnqueryvetSettings{
				AllowedPatterns: []string{"(?i)COUNT\\(.*\\)", "SELECT.*"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadConfigWithAllOptions(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "full.yaml")

	content := `
check-sql-builders: true
allowed-patterns:
  - "(?i)COUNT\\(\\s*\\*\\s*\\)"
  - "(?i)SELECT \\* FROM information_schema"
ignored-functions:
  - "debug.Query"
  - "test.*"
ignored-files:
  - "*_test.go"
  - "testdata/**"
severity: error
check-aliased-wildcard: true
check-string-concat: true
check-format-strings: true
check-string-builder: true
check-subqueries: true
sql-builders:
  squirrel: true
  gorm: true
  sqlx: true
  ent: true
  pgx: true
  bun: true
  sqlboiler: true
  jet: true
`
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// Verify all options
	if !cfg.CheckSQLBuilders {
		t.Error("CheckSQLBuilders should be true")
	}
	if len(cfg.AllowedPatterns) != 2 {
		t.Errorf("AllowedPatterns len = %d, want 2", len(cfg.AllowedPatterns))
	}
	if len(cfg.IgnoredFunctions) != 2 {
		t.Errorf("IgnoredFunctions len = %d, want 2", len(cfg.IgnoredFunctions))
	}
	if len(cfg.IgnoredFiles) != 2 {
		t.Errorf("IgnoredFiles len = %d, want 2", len(cfg.IgnoredFiles))
	}
	if cfg.Severity != "error" {
		t.Errorf("Severity = %s, want error", cfg.Severity)
	}
	if !cfg.CheckAliasedWildcard {
		t.Error("CheckAliasedWildcard should be true")
	}
	if !cfg.CheckStringConcat {
		t.Error("CheckStringConcat should be true")
	}
	if !cfg.CheckFormatStrings {
		t.Error("CheckFormatStrings should be true")
	}
	if !cfg.CheckStringBuilder {
		t.Error("CheckStringBuilder should be true")
	}
	if !cfg.CheckSubqueries {
		t.Error("CheckSubqueries should be true")
	}

	// SQL Builders
	if !cfg.SQLBuilders.Squirrel {
		t.Error("SQLBuilders.Squirrel should be true")
	}
	if !cfg.SQLBuilders.GORM {
		t.Error("SQLBuilders.GORM should be true")
	}
	if !cfg.SQLBuilders.SQLx {
		t.Error("SQLBuilders.SQLx should be true")
	}
	if !cfg.SQLBuilders.Ent {
		t.Error("SQLBuilders.Ent should be true")
	}
	if !cfg.SQLBuilders.PGX {
		t.Error("SQLBuilders.PGX should be true")
	}
	if !cfg.SQLBuilders.Bun {
		t.Error("SQLBuilders.Bun should be true")
	}
	if !cfg.SQLBuilders.SQLBoiler {
		t.Error("SQLBuilders.SQLBoiler should be true")
	}
	if !cfg.SQLBuilders.Jet {
		t.Error("SQLBuilders.Jet should be true")
	}
}

func TestDefaultRulesInLoadedConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// Test that default Rules are preserved when loading a partial config
	t.Run("partial config preserves default rules", func(t *testing.T) {
		configPath := filepath.Join(tmpDir, "partial.yaml")
		// Config that only sets severity, doesn't mention rules
		content := `severity: error`
		if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}

		cfg, err := LoadConfig(configPath)
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}

		// Rules should have default values
		if cfg.Rules == nil {
			t.Fatal("Rules should not be nil")
		}
		if cfg.Rules["select-star"] != "warning" {
			t.Errorf("Rules[select-star] = %s, want warning", cfg.Rules["select-star"])
		}
		if cfg.Rules["n1-queries"] != "warning" {
			t.Errorf("Rules[n1-queries] = %s, want warning", cfg.Rules["n1-queries"])
		}
		if cfg.Rules["sql-injection"] != "error" {
			t.Errorf("Rules[sql-injection] = %s, want error", cfg.Rules["sql-injection"])
		}
	})

	// Test that user can override rules
	t.Run("config can override default rules", func(t *testing.T) {
		configPath := filepath.Join(tmpDir, "override.yaml")
		content := `
rules:
  select-star: error
  n1-queries: ignore
`
		if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}

		cfg, err := LoadConfig(configPath)
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}

		// User-specified rules should override defaults
		if cfg.Rules["select-star"] != "error" {
			t.Errorf("Rules[select-star] = %s, want error", cfg.Rules["select-star"])
		}
		if cfg.Rules["n1-queries"] != "ignore" {
			t.Errorf("Rules[n1-queries] = %s, want ignore", cfg.Rules["n1-queries"])
		}
		// sql-injection should still have default value
		if cfg.Rules["sql-injection"] != "error" {
			t.Errorf("Rules[sql-injection] = %s, want error", cfg.Rules["sql-injection"])
		}
	})

	// Test that empty config uses all defaults
	t.Run("empty config uses default rules", func(t *testing.T) {
		configPath := filepath.Join(tmpDir, "empty_rules.yaml")
		if err := os.WriteFile(configPath, []byte(""), 0o644); err != nil {
			t.Fatal(err)
		}

		cfg, err := LoadConfig(configPath)
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}

		// All default rules should be present
		if cfg.Rules == nil {
			t.Fatal("Rules should not be nil")
		}
		if len(cfg.Rules) != 4 {
			t.Errorf("Rules should have 4 entries, got %d", len(cfg.Rules))
		}
	})
}
