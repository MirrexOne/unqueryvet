package analyzer

import (
	"testing"

	"github.com/MirrexOne/unqueryvet/pkg/config"
)

func TestNormalizeSQLQuery(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple quoted string",
			input:    `"SELECT * FROM users"`,
			expected: "SELECT * FROM USERS",
		},
		{
			name:     "backtick string",
			input:    "`SELECT * FROM users`",
			expected: "SELECT * FROM USERS",
		},
		{
			name:     "string with escape sequences",
			input:    `"SELECT * FROM \"users\""`,
			expected: "SELECT * FROM \"USERS\"",
		},
		{
			name:     "multiline string with tabs and newlines",
			input:    `"SELECT *\n\tFROM users\n\tWHERE id = 1"`,
			expected: "SELECT * FROM USERS WHERE ID = 1",
		},
		{
			name:     "string with SQL comment",
			input:    `"SELECT * FROM users -- this is a comment"`,
			expected: "SELECT * FROM USERS",
		},
		{
			name:     "string with multiple spaces",
			input:    `"SELECT   *   FROM   users"`,
			expected: "SELECT * FROM USERS",
		},
		{
			name:     "complex string with all features",
			input:    `"SELECT *\n\tFROM \"users\"\n\t-- comment\n\tWHERE id = 1"`,
			expected: "SELECT * FROM \"USERS\" WHERE ID = 1",
		},
		{
			name:     "empty string",
			input:    `""`,
			expected: "",
		},
		{
			name:     "string too short",
			input:    `"a"`,
			expected: "A",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeSQLQuery(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeSQLQuery(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsSelectStarQuery(t *testing.T) {
	cfg := &config.UnqueryvetSettings{
		AllowedPatterns: []string{},
	}

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "simple SELECT * with FROM",
			input:    "SELECT * FROM users",
			expected: true,
		},
		{
			name:     "SELECT * with WHERE clause",
			input:    "SELECT * FROM users WHERE active = 1",
			expected: true,
		},
		{
			name:     "SELECT * with JOIN",
			input:    "SELECT * FROM users JOIN orders ON users.id = orders.user_id",
			expected: true,
		},
		{
			name:     "SELECT with explicit columns",
			input:    "SELECT id, name FROM users",
			expected: false,
		},
		{
			name:     "SELECT COUNT(*) - should be allowed by default",
			input:    "SELECT COUNT(*) FROM users",
			expected: false,
		},
		{
			name:     "SELECT * without SQL keywords",
			input:    "SELECT *",
			expected: true,
		},
		{
			name:     "INSERT statement",
			input:    "INSERT INTO users VALUES (1, 'John')",
			expected: false,
		},
		{
			name:     "UPDATE statement",
			input:    "UPDATE users SET name = 'Jane' WHERE id = 1",
			expected: false,
		},
		{
			name:     "complex SELECT * query",
			input:    "SELECT * FROM users WHERE active = 1 ORDER BY created_at DESC",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSelectStarQuery(tt.input, cfg)
			if result != tt.expected {
				t.Errorf("isSelectStarQuery(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestConfigLoading(t *testing.T) {
	// Use default settings to test that they contain expected values
	defaults := config.DefaultSettings()
	cfg := &defaults

	// Test that SQL builders are checked by default
	if !cfg.CheckSQLBuilders {
		t.Error("CheckSQLBuilders should be enabled by default")
	}

	// Test that default allowed patterns include COUNT(*) and system tables
	if len(cfg.AllowedPatterns) == 0 {
		t.Error("Should have some default allowed patterns")
	}
}

func TestAllowedPatterns(t *testing.T) {
	// Use default settings which include the allowed patterns
	defaults := config.DefaultSettings()
	cfg := &defaults

	countQuery := "SELECT COUNT(*) FROM users"
	if isSelectStarQuery(countQuery, cfg) {
		t.Error("COUNT(*) should be allowed by default allowed patterns")
	}

	schemaQuery := "SELECT * FROM information_schema.tables"
	if isSelectStarQuery(schemaQuery, cfg) {
		t.Error("information_schema queries should be allowed by default")
	}

	normalQuery := "SELECT * FROM users WHERE active = 1"
	if !isSelectStarQuery(normalQuery, cfg) {
		t.Error("Normal SELECT * queries should not be allowed")
	}
}

func TestAllowedPatternsWithRegex(t *testing.T) {
	// Use default settings for consistent testing
	defaults := config.DefaultSettings()
	cfg := &defaults

	tests := []struct {
		name    string
		query   string
		allowed bool
	}{
		{
			name:    "COUNT(*) with spaces",
			query:   "SELECT COUNT( * ) FROM users",
			allowed: true,
		},
		{
			name:    "Case-insensitive COUNT",
			query:   "select count(*) from USERS",
			allowed: true,
		},
		{
			name:    "information_schema query",
			query:   "SELECT * FROM INFORMATION_SCHEMA.TABLES",
			allowed: true,
		},
		{
			name:    "Normal SELECT * query",
			query:   "SELECT * FROM users",
			allowed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// isSelectStarQuery returns true if the query is *not* allowed
			result := isSelectStarQuery(tt.query, cfg)
			if result == tt.allowed {
				t.Errorf("isSelectStarQuery(%q) = %v, want %v (allowed)", tt.query, result, !tt.allowed)
			}
		})
	}
}

func TestAliasedWildcard(t *testing.T) {
	defaults := config.DefaultSettings()
	cfg := &defaults

	tests := []struct {
		name     string
		query    string
		expected bool
	}{
		{
			name:     "SELECT t.* with alias",
			query:    "SELECT t.* FROM users t",
			expected: true,
		},
		{
			name:     "SELECT multiple aliases",
			query:    "SELECT u.*, o.* FROM users u JOIN orders o",
			expected: true,
		},
		{
			name:     "SELECT table.* without alias",
			query:    "SELECT users.* FROM users",
			expected: true,
		},
		{
			name:     "SELECT explicit columns with alias",
			query:    "SELECT t.id, t.name FROM users t",
			expected: false,
		},
		{
			name:     "SELECT explicit columns no alias",
			query:    "SELECT id, name FROM users",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSelectStarQuery(tt.query, cfg)
			if result != tt.expected {
				t.Errorf("isSelectStarQuery(%q) = %v, want %v", tt.query, result, tt.expected)
			}
		})
	}
}

func TestSubqueryDetection(t *testing.T) {
	defaults := config.DefaultSettings()
	cfg := &defaults

	tests := []struct {
		name     string
		query    string
		expected bool
	}{
		{
			name:     "SELECT * in subquery",
			query:    "SELECT id FROM (SELECT * FROM users)",
			expected: true,
		},
		{
			name:     "SELECT * in IN clause",
			query:    "SELECT id FROM users WHERE id IN (SELECT * FROM orders)",
			expected: true,
		},
		{
			name:     "SELECT * in EXISTS",
			query:    "SELECT id FROM users WHERE EXISTS (SELECT * FROM orders)",
			expected: true,
		},
		{
			name:     "explicit columns in subquery",
			query:    "SELECT id FROM (SELECT id, name FROM users)",
			expected: false,
		},
		{
			name:     "explicit columns in IN",
			query:    "SELECT id FROM users WHERE id IN (SELECT user_id FROM orders)",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSelectStarQuery(tt.query, cfg)
			if result != tt.expected {
				t.Errorf("isSelectStarQuery(%q) = %v, want %v", tt.query, result, tt.expected)
			}
		})
	}
}

func TestNewConfigurationOptions(t *testing.T) {
	defaults := config.DefaultSettings()

	// Test new detection flags are enabled by default
	if !defaults.CheckAliasedWildcard {
		t.Error("CheckAliasedWildcard should be enabled by default")
	}
	if !defaults.CheckStringConcat {
		t.Error("CheckStringConcat should be enabled by default")
	}
	if !defaults.CheckFormatStrings {
		t.Error("CheckFormatStrings should be enabled by default")
	}
	if !defaults.CheckStringBuilder {
		t.Error("CheckStringBuilder should be enabled by default")
	}
	if !defaults.CheckSubqueries {
		t.Error("CheckSubqueries should be enabled by default")
	}

	// Test severity default
	if defaults.Severity != "warning" {
		t.Errorf("Severity should be 'warning' by default, got %q", defaults.Severity)
	}

	// Test SQL builders config
	if !defaults.SQLBuilders.Squirrel {
		t.Error("SQLBuilders.Squirrel should be enabled by default")
	}
	if !defaults.SQLBuilders.GORM {
		t.Error("SQLBuilders.GORM should be enabled by default")
	}
	if !defaults.SQLBuilders.SQLx {
		t.Error("SQLBuilders.SQLx should be enabled by default")
	}
	if !defaults.SQLBuilders.Bun {
		t.Error("SQLBuilders.Bun should be enabled by default")
	}
	if !defaults.SQLBuilders.SQLBoiler {
		t.Error("SQLBuilders.SQLBoiler should be enabled by default")
	}
	if !defaults.SQLBuilders.Jet {
		t.Error("SQLBuilders.Jet should be enabled by default")
	}
}

func TestFilterContext(t *testing.T) {
	cfg := &config.UnqueryvetSettings{
		IgnoredFunctions: []string{"debug.*", "test.Query"},
		IgnoredFiles:     []string{"*_test.go", "testdata/**"},
		AllowedPatterns:  []string{`(?i)COUNT\(\s*\*\s*\)`},
	}

	filter, err := NewFilterContext(cfg)
	if err != nil {
		t.Fatalf("Failed to create FilterContext: %v", err)
	}

	// Test file filtering
	if !filter.IsIgnoredFile("foo_test.go") {
		t.Error("foo_test.go should be ignored")
	}
	if filter.IsIgnoredFile("foo.go") {
		t.Error("foo.go should not be ignored")
	}

	// Test allowed patterns
	if !filter.IsAllowedPattern("SELECT COUNT(*) FROM users") {
		t.Error("COUNT(*) should be allowed")
	}
	if filter.IsAllowedPattern("SELECT * FROM users") {
		t.Error("SELECT * FROM users should not be allowed")
	}
}

func TestIsRuleEnabled(t *testing.T) {
	tests := []struct {
		name     string
		rules    config.RuleSeverity
		ruleID   string
		expected bool
	}{
		{
			name:     "nil rules returns false",
			rules:    nil,
			ruleID:   "select-star",
			expected: false,
		},
		{
			name:     "rule not in map returns false",
			rules:    config.RuleSeverity{"other-rule": "warning"},
			ruleID:   "select-star",
			expected: false,
		},
		{
			name:     "rule with warning severity is enabled",
			rules:    config.RuleSeverity{"select-star": "warning"},
			ruleID:   "select-star",
			expected: true,
		},
		{
			name:     "rule with error severity is enabled",
			rules:    config.RuleSeverity{"sql-injection": "error"},
			ruleID:   "sql-injection",
			expected: true,
		},
		{
			name:     "rule with info severity is enabled",
			rules:    config.RuleSeverity{"n1-queries": "info"},
			ruleID:   "n1-queries",
			expected: true,
		},
		{
			name:     "rule with ignore severity is disabled",
			rules:    config.RuleSeverity{"n1-queries": "ignore"},
			ruleID:   "n1-queries",
			expected: false,
		},
		{
			name:     "empty string severity is enabled",
			rules:    config.RuleSeverity{"select-star": ""},
			ruleID:   "select-star",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRuleEnabled(tt.rules, tt.ruleID)
			if result != tt.expected {
				t.Errorf("isRuleEnabled(%v, %q) = %v, want %v", tt.rules, tt.ruleID, result, tt.expected)
			}
		})
	}
}

func TestDefaultRulesEnableDetection(t *testing.T) {
	defaults := config.DefaultSettings()

	// With default settings, all rules should be enabled
	if !isRuleEnabled(defaults.Rules, "select-star") {
		t.Error("select-star should be enabled with default settings")
	}
	if !isRuleEnabled(defaults.Rules, "n1-queries") {
		t.Error("n1-queries should be enabled with default settings")
	}
	if !isRuleEnabled(defaults.Rules, "sql-injection") {
		t.Error("sql-injection should be enabled with default settings")
	}
}

func TestRulesCanBeDisabled(t *testing.T) {
	cfg := config.DefaultSettings()

	// Override to disable n1-queries
	cfg.Rules["n1-queries"] = "ignore"

	// select-star and sql-injection should still be enabled
	if !isRuleEnabled(cfg.Rules, "select-star") {
		t.Error("select-star should still be enabled")
	}
	if !isRuleEnabled(cfg.Rules, "sql-injection") {
		t.Error("sql-injection should still be enabled")
	}

	// n1-queries should be disabled
	if isRuleEnabled(cfg.Rules, "n1-queries") {
		t.Error("n1-queries should be disabled when set to ignore")
	}
}

func TestDefaultRulesIntegration(t *testing.T) {
	// Test that default rules work correctly in RunWithConfig
	defaults := config.DefaultSettings()

	t.Run("default config has all rules enabled", func(t *testing.T) {
		// Verify the defaults are correct
		if defaults.Rules == nil {
			t.Fatal("Rules should not be nil in default config")
		}

		expectedRules := map[string]string{
			"select-star":   "warning",
			"n1-queries":    "warning",
			"sql-injection": "error",
		}

		for rule, expectedSeverity := range expectedRules {
			severity, ok := defaults.Rules[rule]
			if !ok {
				t.Errorf("rule %q not found in default config", rule)
				continue
			}
			if severity != expectedSeverity {
				t.Errorf("rule %q severity = %q, want %q", rule, severity, expectedSeverity)
			}
		}
	})

	t.Run("select-star rule is not ignore", func(t *testing.T) {
		if defaults.Rules["select-star"] == "ignore" {
			t.Error("select-star should not be ignore by default")
		}
	})

	t.Run("n1-queries rule is not ignore", func(t *testing.T) {
		if defaults.Rules["n1-queries"] == "ignore" {
			t.Error("n1-queries should not be ignore by default")
		}
	})

	t.Run("sql-injection rule is not ignore", func(t *testing.T) {
		if defaults.Rules["sql-injection"] == "ignore" {
			t.Error("sql-injection should not be ignore by default")
		}
	})
}

func TestRuleSeverityValues(t *testing.T) {
	validSeverities := []string{"error", "warning", "info", "ignore", ""}

	for _, severity := range validSeverities {
		rules := config.RuleSeverity{"test-rule": severity}

		if severity == "ignore" {
			if isRuleEnabled(rules, "test-rule") {
				t.Errorf("rule with severity %q should be disabled", severity)
			}
		} else {
			if !isRuleEnabled(rules, "test-rule") {
				t.Errorf("rule with severity %q should be enabled", severity)
			}
		}
	}
}

func TestDefaultSettingsAllFieldsInitialized(t *testing.T) {
	defaults := config.DefaultSettings()

	// Check all boolean fields are set to expected values
	boolFields := map[string]bool{
		"CheckSQLBuilders":     defaults.CheckSQLBuilders,
		"CheckAliasedWildcard": defaults.CheckAliasedWildcard,
		"CheckStringConcat":    defaults.CheckStringConcat,
		"CheckFormatStrings":   defaults.CheckFormatStrings,
		"CheckStringBuilder":   defaults.CheckStringBuilder,
		"CheckSubqueries":      defaults.CheckSubqueries,
	}

	for name, value := range boolFields {
		if !value {
			t.Errorf("%s should be true by default", name)
		}
	}

	// Check string fields
	if defaults.Severity != "warning" {
		t.Errorf("Severity should be 'warning', got %q", defaults.Severity)
	}

	// Check slice fields are not nil
	if defaults.AllowedPatterns == nil {
		t.Error("AllowedPatterns should not be nil")
	}
	if len(defaults.AllowedPatterns) == 0 {
		t.Error("AllowedPatterns should have default entries")
	}

	// Check Rules map
	if defaults.Rules == nil {
		t.Error("Rules should not be nil")
	}
	if len(defaults.Rules) != 4 {
		t.Errorf("Rules should have 4 entries, got %d", len(defaults.Rules))
	}

	// Check SQLBuilders config
	sqlBuilders := defaults.SQLBuilders
	if !sqlBuilders.Squirrel || !sqlBuilders.GORM || !sqlBuilders.SQLx ||
		!sqlBuilders.Ent || !sqlBuilders.PGX || !sqlBuilders.Bun ||
		!sqlBuilders.SQLBoiler || !sqlBuilders.Jet {
		t.Error("All SQL builders should be enabled by default")
	}
}
