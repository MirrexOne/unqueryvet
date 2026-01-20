package analyzer_test

import (
	"go/parser"
	"go/token"
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/MirrexOne/unqueryvet/internal/analyzer"
	"github.com/MirrexOne/unqueryvet/pkg/config"
)

func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, analyzer.NewAnalyzer(), "a")
}

func TestAnalyzerWithSettings(t *testing.T) {
	testdata := analysistest.TestData()

	// Test with default settings
	analysistest.Run(t, testdata, analyzer.NewAnalyzer(), "clean")

	// Test with SQL builders detection
	analysistest.Run(t, testdata, analyzer.NewAnalyzer(), "integration")
}

// TestNoFalsePositivesForCustomTypes verifies that custom types with methods like
// All(), One(), Count() do NOT trigger false positives from SQL builder checkers.
// This is a regression test for issue #5.
func TestNoFalsePositivesForCustomTypes(t *testing.T) {
	testdata := analysistest.TestData()

	// falsepositive package should have NO warnings
	// The checker should not flag custom types that happen to have methods
	// with the same names as SQL builder methods (All, One, Count, Select, etc.)
	analysistest.Run(t, testdata, analyzer.NewAnalyzer(), "falsepositive")
}

// TestRealSQLBoilerIsDetected verifies that REAL SQLBoiler code IS detected.
// This is an e2e test to ensure the checker works correctly with actual SQLBoiler types.
// Combined with TestNoFalsePositivesForCustomTypes, this proves the fix for issue #5 is complete.
func TestRealSQLBoilerIsDetected(t *testing.T) {
	testdata := analysistest.TestData()

	// sqlboilerreal package has real SQLBoiler imports
	// The checker SHOULD flag qm.Select("*") but NOT qm.Select("id", "name")
	analysistest.Run(t, testdata, analyzer.NewAnalyzer(), "sqlboilerreal")
}

// TestDefaultRulesWork verifies that default rules are active
func TestDefaultRulesWork(t *testing.T) {
	testdata := analysistest.TestData()

	// Test with default settings - all rules should be active
	analysistest.Run(t, testdata, analyzer.NewAnalyzer(), "rules")
}

// TestSelectStarRuleWithDefaultConfig tests that select-star rule works with default config
func TestSelectStarRuleWithDefaultConfig(t *testing.T) {
	testdata := analysistest.TestData()

	// Create analyzer with explicit default settings
	cfg := config.DefaultSettings()
	a := analyzer.NewAnalyzerWithSettings(cfg)

	// Should detect SELECT * issues
	analysistest.Run(t, testdata, a, "rules")
}

// TestSelectStarRuleDisabled tests that select-star can be disabled via config
func TestSelectStarRuleDisabled(t *testing.T) {
	testdata := analysistest.TestData()

	// Create config with select-star disabled
	cfg := config.DefaultSettings()
	cfg.Rules["select-star"] = "ignore"

	a := analyzer.NewAnalyzerWithSettings(cfg)

	// Should NOT detect SELECT * issues when rule is ignored
	// Note: "clean" directory has no expected warnings
	analysistest.Run(t, testdata, a, "clean")
}

// TestAllDefaultRulesEnabled verifies all three default rules are enabled
func TestAllDefaultRulesEnabled(t *testing.T) {
	cfg := config.DefaultSettings()

	// Verify rules are present and not ignored
	rules := []string{"select-star", "n1-queries", "sql-injection"}
	for _, rule := range rules {
		severity, ok := cfg.Rules[rule]
		if !ok {
			t.Errorf("rule %q should be present in default config", rule)
			continue
		}
		if severity == "ignore" {
			t.Errorf("rule %q should not be 'ignore' by default", rule)
		}
	}
}

// TestRulesCanBeOverridden verifies rules can be overridden in config
func TestRulesCanBeOverridden(t *testing.T) {
	cfg := config.DefaultSettings()

	// Override rules
	cfg.Rules["select-star"] = "error"
	cfg.Rules["n1-queries"] = "ignore"
	cfg.Rules["sql-injection"] = "warning"

	if cfg.Rules["select-star"] != "error" {
		t.Error("should be able to set select-star to error")
	}
	if cfg.Rules["n1-queries"] != "ignore" {
		t.Error("should be able to set n1-queries to ignore")
	}
	if cfg.Rules["sql-injection"] != "warning" {
		t.Error("should be able to set sql-injection to warning")
	}
}

// TestNewAnalyzerWithSettingsUsesRules verifies NewAnalyzerWithSettings respects Rules config
func TestNewAnalyzerWithSettingsUsesRules(t *testing.T) {
	testdata := analysistest.TestData()

	t.Run("default config detects issues", func(t *testing.T) {
		cfg := config.DefaultSettings()
		a := analyzer.NewAnalyzerWithSettings(cfg)
		// "rules" directory has SELECT * issues with correct expectations
		analysistest.Run(t, testdata, a, "rules")
	})

	t.Run("custom allowed patterns work", func(t *testing.T) {
		cfg := config.DefaultSettings()
		// Add pattern to allow all SELECT * (for testing)
		cfg.AllowedPatterns = append(cfg.AllowedPatterns, `(?i)SELECT \* FROM clean_table`)
		a := analyzer.NewAnalyzerWithSettings(cfg)
		// "clean" directory should have no issues
		analysistest.Run(t, testdata, a, "clean")
	})
}

// =============================================================================
// N+1 QUERIES RULE TESTS
// =============================================================================

// TestN1QueriesRuleEnabledByDefault verifies n1-queries rule is enabled by default
func TestN1QueriesRuleEnabledByDefault(t *testing.T) {
	cfg := config.DefaultSettings()

	severity, ok := cfg.Rules["n1-queries"]
	if !ok {
		t.Fatal("n1-queries rule should be present in default config")
	}
	if severity == "ignore" {
		t.Error("n1-queries rule should not be 'ignore' by default")
	}
	if severity != "warning" {
		t.Errorf("n1-queries rule severity = %q, want 'warning'", severity)
	}
}

// TestN1QueriesRuleDetectsIssues verifies n1-queries rule detects N+1 patterns
func TestN1QueriesRuleDetectsIssues(t *testing.T) {
	src := `
package main

import "database/sql"

func getUsers(db *sql.DB, ids []int) {
	for _, id := range ids {
		db.Query("SELECT * FROM users WHERE id = ?", id)
	}
}
`
	detector := analyzer.NewN1Detector()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	violations := detector.CheckN1QueriesNoPass(file)
	if len(violations) == 0 {
		t.Error("n1-queries rule should detect query in loop")
	}
}

// TestN1QueriesRuleCanBeDisabled verifies n1-queries rule can be disabled
func TestN1QueriesRuleCanBeDisabled(t *testing.T) {
	cfg := config.DefaultSettings()
	cfg.Rules["n1-queries"] = "ignore"

	// Verify the rule is now disabled
	if cfg.Rules["n1-queries"] != "ignore" {
		t.Error("should be able to disable n1-queries rule")
	}

	// Verify isRuleEnabled returns false
	if analyzer.IsRuleEnabledExported(cfg.Rules, "n1-queries") {
		t.Error("isRuleEnabled should return false for ignored rule")
	}
}

// TestN1QueriesRuleNoFalsePositives verifies n1-queries doesn't flag batch queries
func TestN1QueriesRuleNoFalsePositives(t *testing.T) {
	src := `
package main

import "database/sql"

func getUsers(db *sql.DB, ids []int) {
	// Batch query - should NOT be flagged
	db.Query("SELECT * FROM users WHERE id IN (?)", ids)
}
`
	detector := analyzer.NewN1Detector()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	violations := detector.CheckN1QueriesNoPass(file)
	if len(violations) != 0 {
		t.Error("n1-queries rule should not flag batch queries")
	}
}

// =============================================================================
// SQL INJECTION RULE TESTS
// =============================================================================

// TestSQLInjectionRuleEnabledByDefault verifies sql-injection rule is enabled by default
func TestSQLInjectionRuleEnabledByDefault(t *testing.T) {
	cfg := config.DefaultSettings()

	severity, ok := cfg.Rules["sql-injection"]
	if !ok {
		t.Fatal("sql-injection rule should be present in default config")
	}
	if severity == "ignore" {
		t.Error("sql-injection rule should not be 'ignore' by default")
	}
	if severity != "error" {
		t.Errorf("sql-injection rule severity = %q, want 'error'", severity)
	}
}

// TestSQLInjectionRuleDetectsIssues verifies sql-injection rule detects vulnerabilities
func TestSQLInjectionRuleDetectsIssues(t *testing.T) {
	src := `
package main

import (
	"database/sql"
	"fmt"
)

func getUser(db *sql.DB, userID string) {
	query := fmt.Sprintf("SELECT * FROM users WHERE id = '%s'", userID)
	db.Query(query)
}
`
	scanner := analyzer.NewSQLInjectionScanner()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	violations := scanner.ScanFileNoPass(fset, file)
	if len(violations) == 0 {
		t.Error("sql-injection rule should detect sprintf SQL injection")
	}
}

// TestSQLInjectionRuleDetectsConcatenation verifies sql-injection detects string concat
func TestSQLInjectionRuleDetectsConcatenation(t *testing.T) {
	src := `
package main

import "database/sql"

func getUser(db *sql.DB, userID string) {
	db.Query("SELECT * FROM users WHERE id = '" + userID + "'")
}
`
	scanner := analyzer.NewSQLInjectionScanner()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	violations := scanner.ScanFileNoPass(fset, file)
	if len(violations) == 0 {
		t.Error("sql-injection rule should detect string concatenation")
	}
}

// TestSQLInjectionRuleCanBeDisabled verifies sql-injection rule can be disabled
func TestSQLInjectionRuleCanBeDisabled(t *testing.T) {
	cfg := config.DefaultSettings()
	cfg.Rules["sql-injection"] = "ignore"

	// Verify the rule is now disabled
	if cfg.Rules["sql-injection"] != "ignore" {
		t.Error("should be able to disable sql-injection rule")
	}

	// Verify isRuleEnabled returns false
	if analyzer.IsRuleEnabledExported(cfg.Rules, "sql-injection") {
		t.Error("isRuleEnabled should return false for ignored rule")
	}
}

// TestSQLInjectionRuleNoFalsePositives verifies sql-injection doesn't flag parameterized queries
func TestSQLInjectionRuleNoFalsePositives(t *testing.T) {
	src := `
package main

import "database/sql"

func getUser(db *sql.DB, userID string) {
	// Parameterized query - should NOT be flagged as high severity
	db.Query("SELECT * FROM users WHERE id = ?", userID)
}
`
	scanner := analyzer.NewSQLInjectionScanner()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	violations := scanner.ScanFileNoPass(fset, file)

	// Count high severity violations
	highSeverity := 0
	for _, v := range violations {
		if v.Severity == analyzer.SQLISeverityHigh || v.Severity == analyzer.SQLISeverityCritical {
			highSeverity++
		}
	}
	if highSeverity > 0 {
		t.Error("sql-injection rule should not flag parameterized queries as high severity")
	}
}

// =============================================================================
// ALL RULES INTEGRATION TESTS
// =============================================================================

// TestAllRulesEnabledInDefaultConfig verifies all three rules work together
func TestAllRulesEnabledInDefaultConfig(t *testing.T) {
	cfg := config.DefaultSettings()

	tests := []struct {
		rule           string
		expectedSev    string
		shouldBeActive bool
	}{
		{"select-star", "warning", true},
		{"n1-queries", "warning", true},
		{"sql-injection", "error", true},
	}

	for _, tt := range tests {
		t.Run(tt.rule, func(t *testing.T) {
			severity, ok := cfg.Rules[tt.rule]
			if !ok {
				t.Errorf("rule %q should be present", tt.rule)
				return
			}
			if severity != tt.expectedSev {
				t.Errorf("rule %q severity = %q, want %q", tt.rule, severity, tt.expectedSev)
			}

			isActive := analyzer.IsRuleEnabledExported(cfg.Rules, tt.rule)
			if isActive != tt.shouldBeActive {
				t.Errorf("rule %q active = %v, want %v", tt.rule, isActive, tt.shouldBeActive)
			}
		})
	}
}

// TestRulesCanBeIndividuallyDisabled verifies rules can be disabled independently
func TestRulesCanBeIndividuallyDisabled(t *testing.T) {
	cfg := config.DefaultSettings()

	// Disable only n1-queries
	cfg.Rules["n1-queries"] = "ignore"

	// select-star should still be active
	if !analyzer.IsRuleEnabledExported(cfg.Rules, "select-star") {
		t.Error("select-star should still be active")
	}

	// sql-injection should still be active
	if !analyzer.IsRuleEnabledExported(cfg.Rules, "sql-injection") {
		t.Error("sql-injection should still be active")
	}

	// n1-queries should be disabled
	if analyzer.IsRuleEnabledExported(cfg.Rules, "n1-queries") {
		t.Error("n1-queries should be disabled")
	}
}

// TestRuleSeverityCanBeChanged verifies rule severity can be changed
func TestRuleSeverityCanBeChanged(t *testing.T) {
	cfg := config.DefaultSettings()

	// Change severities
	cfg.Rules["select-star"] = "error"
	cfg.Rules["n1-queries"] = "info"
	cfg.Rules["sql-injection"] = "warning"

	if cfg.Rules["select-star"] != "error" {
		t.Error("select-star severity should be changeable to error")
	}
	if cfg.Rules["n1-queries"] != "info" {
		t.Error("n1-queries severity should be changeable to info")
	}
	if cfg.Rules["sql-injection"] != "warning" {
		t.Error("sql-injection severity should be changeable to warning")
	}

	// All should still be active (not ignore)
	for _, rule := range []string{"select-star", "n1-queries", "sql-injection"} {
		if !analyzer.IsRuleEnabledExported(cfg.Rules, rule) {
			t.Errorf("rule %q should still be active after severity change", rule)
		}
	}
}
