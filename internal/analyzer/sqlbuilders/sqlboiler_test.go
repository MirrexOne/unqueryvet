package sqlbuilders

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestSQLBoilerChecker_Name(t *testing.T) {
	checker := NewSQLBoilerChecker()
	if checker.Name() != "sqlboiler" {
		t.Errorf("Name() = %q, want %q", checker.Name(), "sqlboiler")
	}
}

func TestSQLBoilerChecker_IsApplicable(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected bool
	}{
		{
			name:     "All method",
			code:     `package test; func f() { models.Users().All(ctx, db) }`,
			expected: false, // Without types.Info, returns false
		},
		{
			name:     "One method",
			code:     `package test; func f() { models.Users().One(ctx, db) }`,
			expected: false, // Without types.Info, returns false
		},
		{
			name:     "Count method",
			code:     `package test; func f() { models.Users().Count(ctx, db) }`,
			expected: false, // Without types.Info, returns false
		},
		{
			name:     "Exists method",
			code:     `package test; func f() { models.Users().Exists(ctx, db) }`,
			expected: false, // Without types.Info, returns false
		},
		{
			name:     "Select method",
			code:     `package test; func f() { qm.Select("id") }`,
			expected: false, // Without types.Info, returns false
		},
		{
			name:     "qm package",
			code:     `package test; func f() { qm.Where("id = ?", 1) }`,
			expected: false, // Without types.Info, returns false
		},
		{
			name:     "non-sqlboiler method",
			code:     `package test; func f() { db.Other() }`,
			expected: false,
		},
	}

	checker := NewSQLBoilerChecker()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, "test.go", tt.code, 0)
			if err != nil {
				t.Fatalf("Failed to parse code: %v", err)
			}

			var result bool
			ast.Inspect(f, func(n ast.Node) bool {
				if call, ok := n.(*ast.CallExpr); ok {
					result = checker.IsApplicable(nil, call)
					return false
				}
				return true
			})

			if result != tt.expected {
				t.Errorf("IsApplicable() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSQLBoilerChecker_CheckSelectStar(t *testing.T) {
	tests := []struct {
		name            string
		code            string
		expectViolation bool
	}{
		{
			name:            "qm.Select with star",
			code:            `package test; func f() { qm.Select("*") }`,
			expectViolation: true,
		},
		{
			name:            "qm.Select with explicit column",
			code:            `package test; func f() { qm.Select("id") }`,
			expectViolation: false,
		},
		{
			name:            "qm.Select with multiple columns",
			code:            `package test; func f() { qm.Select("id", "name") }`,
			expectViolation: false,
		},
		{
			name:            "non-qm Select",
			code:            `package test; func f() { other.Select("*") }`,
			expectViolation: false,
		},
	}

	checker := NewSQLBoilerChecker()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, "test.go", tt.code, 0)
			if err != nil {
				t.Fatalf("Failed to parse code: %v", err)
			}

			var violation *SelectStarViolation
			ast.Inspect(f, func(n ast.Node) bool {
				if call, ok := n.(*ast.CallExpr); ok {
					violation = checker.CheckSelectStar(call)
					return false
				}
				return true
			})

			hasViolation := violation != nil
			if hasViolation != tt.expectViolation {
				t.Errorf("CheckSelectStar() hasViolation = %v, want %v", hasViolation, tt.expectViolation)
			}

			if violation != nil && violation.Builder != "sqlboiler" {
				t.Errorf("CheckSelectStar() builder = %q, want %q", violation.Builder, "sqlboiler")
			}
		})
	}
}

func TestSQLBoilerChecker_CheckChainedCalls(t *testing.T) {
	tests := []struct {
		name              string
		code              string
		expectViolations  bool
		minViolationCount int
	}{
		{
			name:              "model All without query mods (implicit star)",
			code:              `package test; func f() { models.Users().All(ctx, db) }`,
			expectViolations:  true,
			minViolationCount: 1,
		},
		{
			name:              "model One without query mods",
			code:              `package test; func f() { models.Users().One(ctx, db) }`,
			expectViolations:  true,
			minViolationCount: 1,
		},
		{
			// Note: qm.Select("*") is detected by CheckSelectStar, not CheckChainedCalls
			// CheckChainedCalls only detects model().All() without any Select
			name:              "model with qm.Select star - detected by CheckSelectStar not CheckChainedCalls",
			code:              `package test; func f() { models.Users(qm.Select("*")).All(ctx, db) }`,
			expectViolations:  false, // hasSelect=true, so no implicit SELECT * violation
			minViolationCount: 0,
		},
		{
			name:              "model with qm.Select explicit columns",
			code:              `package test; func f() { models.Users(qm.Select("id", "name")).All(ctx, db) }`,
			expectViolations:  false,
			minViolationCount: 0,
		},
	}

	checker := NewSQLBoilerChecker()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, "test.go", tt.code, 0)
			if err != nil {
				t.Fatalf("Failed to parse code: %v", err)
			}

			var allViolations []*SelectStarViolation
			ast.Inspect(f, func(n ast.Node) bool {
				if call, ok := n.(*ast.CallExpr); ok {
					violations := checker.CheckChainedCalls(call)
					allViolations = append(allViolations, violations...)
				}
				return true
			})

			hasViolations := len(allViolations) > 0
			if hasViolations != tt.expectViolations {
				t.Errorf("CheckChainedCalls() hasViolations = %v, want %v", hasViolations, tt.expectViolations)
			}

			if len(allViolations) < tt.minViolationCount {
				t.Errorf("CheckChainedCalls() violations = %d, want at least %d", len(allViolations), tt.minViolationCount)
			}
		})
	}
}

func TestNewSQLBoilerChecker(t *testing.T) {
	checker := NewSQLBoilerChecker()
	if checker == nil {
		t.Error("NewSQLBoilerChecker() returned nil")
	}
}
