package config

import "testing"

func TestDefaultSettings(t *testing.T) {
	defaults := DefaultSettings()

	t.Run("all detection features enabled by default", func(t *testing.T) {
		if !defaults.CheckSQLBuilders {
			t.Error("CheckSQLBuilders should be true by default")
		}
		if !defaults.CheckAliasedWildcard {
			t.Error("CheckAliasedWildcard should be true by default")
		}
		if !defaults.CheckStringConcat {
			t.Error("CheckStringConcat should be true by default")
		}
		if !defaults.CheckFormatStrings {
			t.Error("CheckFormatStrings should be true by default")
		}
		if !defaults.CheckStringBuilder {
			t.Error("CheckStringBuilder should be true by default")
		}
		if !defaults.CheckSubqueries {
			t.Error("CheckSubqueries should be true by default")
		}
	})

	t.Run("default severity is warning", func(t *testing.T) {
		if defaults.Severity != "warning" {
			t.Errorf("Severity = %q, want %q", defaults.Severity, "warning")
		}
	})

	t.Run("allowed patterns include COUNT(*)", func(t *testing.T) {
		found := false
		for _, p := range defaults.AllowedPatterns {
			if contains(p, "COUNT") {
				found = true
				break
			}
		}
		if !found {
			t.Error("AllowedPatterns should include COUNT(*) pattern")
		}
	})
}

func TestDefaultRules(t *testing.T) {
	defaults := DefaultSettings()

	t.Run("Rules map is not nil", func(t *testing.T) {
		if defaults.Rules == nil {
			t.Fatal("Rules should not be nil")
		}
	})

	t.Run("select-star rule enabled with warning severity", func(t *testing.T) {
		severity, ok := defaults.Rules["select-star"]
		if !ok {
			t.Fatal("select-star rule should be present")
		}
		if severity != "warning" {
			t.Errorf("select-star severity = %q, want %q", severity, "warning")
		}
	})

	t.Run("n1-queries rule enabled with warning severity", func(t *testing.T) {
		severity, ok := defaults.Rules["n1-queries"]
		if !ok {
			t.Fatal("n1-queries rule should be present")
		}
		if severity != "warning" {
			t.Errorf("n1-queries severity = %q, want %q", severity, "warning")
		}
	})

	t.Run("sql-injection rule enabled with error severity", func(t *testing.T) {
		severity, ok := defaults.Rules["sql-injection"]
		if !ok {
			t.Fatal("sql-injection rule should be present")
		}
		if severity != "error" {
			t.Errorf("sql-injection severity = %q, want %q", severity, "error")
		}
	})

	t.Run("all four default rules are present", func(t *testing.T) {
		expectedRules := []string{"select-star", "n1-queries", "sql-injection", "tx-leak"}
		for _, rule := range expectedRules {
			if _, ok := defaults.Rules[rule]; !ok {
				t.Errorf("rule %q should be present in defaults", rule)
			}
		}
		if len(defaults.Rules) != 4 {
			t.Errorf("expected 4 default rules, got %d", len(defaults.Rules))
		}
	})
}

func TestDefaultSQLBuildersConfig(t *testing.T) {
	defaults := DefaultSQLBuildersConfig()

	builders := map[string]bool{
		"Squirrel":  defaults.Squirrel,
		"GORM":      defaults.GORM,
		"SQLx":      defaults.SQLx,
		"Ent":       defaults.Ent,
		"PGX":       defaults.PGX,
		"Bun":       defaults.Bun,
		"SQLBoiler": defaults.SQLBoiler,
		"Jet":       defaults.Jet,
	}

	for name, enabled := range builders {
		if !enabled {
			t.Errorf("%s should be enabled by default", name)
		}
	}
}

func TestRuleSeverityCanBeModified(t *testing.T) {
	defaults := DefaultSettings()

	// Modify a rule severity
	defaults.Rules["select-star"] = "error"
	defaults.Rules["n1-queries"] = "ignore"

	if defaults.Rules["select-star"] != "error" {
		t.Error("should be able to change select-star severity to error")
	}
	if defaults.Rules["n1-queries"] != "ignore" {
		t.Error("should be able to change n1-queries severity to ignore")
	}
	// sql-injection should remain unchanged
	if defaults.Rules["sql-injection"] != "error" {
		t.Error("sql-injection should remain error")
	}
}

func TestRuleSeverityCanAddNewRules(t *testing.T) {
	defaults := DefaultSettings()

	// Add a custom rule
	defaults.Rules["custom-rule"] = "info"

	if defaults.Rules["custom-rule"] != "info" {
		t.Error("should be able to add custom rules")
	}
	if len(defaults.Rules) != 5 {
		t.Errorf("expected 5 rules after adding custom, got %d", len(defaults.Rules))
	}
}

func TestDefaultSettingsAreIndependent(t *testing.T) {
	// Ensure that modifying one instance doesn't affect another
	defaults1 := DefaultSettings()
	defaults2 := DefaultSettings()

	defaults1.Rules["select-star"] = "error"
	defaults1.CheckSQLBuilders = false

	// defaults2 should be unaffected
	if defaults2.Rules["select-star"] != "warning" {
		t.Error("modifying defaults1 should not affect defaults2 Rules")
	}
	if !defaults2.CheckSQLBuilders {
		t.Error("modifying defaults1 should not affect defaults2 CheckSQLBuilders")
	}
}

func TestDefaultAllowedPatternsComprehensive(t *testing.T) {
	defaults := DefaultSettings()

	expectedPatterns := []string{
		"COUNT",
		"MAX",
		"MIN",
		"information_schema",
		"pg_catalog",
		"sys",
	}

	for _, expected := range expectedPatterns {
		found := false
		for _, pattern := range defaults.AllowedPatterns {
			if contains(pattern, expected) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("AllowedPatterns should include pattern for %s", expected)
		}
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
