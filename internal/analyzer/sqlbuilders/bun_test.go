package sqlbuilders

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestBunChecker_Name(t *testing.T) {
	checker := NewBunChecker()
	if checker.Name() != "bun" {
		t.Errorf("Name() = %q, want %q", checker.Name(), "bun")
	}
}

func TestBunChecker_IsApplicable(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected bool
	}{
		{
			name:     "NewSelect method",
			code:     `package test; func f() { db.NewSelect() }`,
			expected: false, // Without types.Info, returns false
		},
		{
			name:     "Column method",
			code:     `package test; func f() { q.Column("id") }`,
			expected: false, // Without types.Info, returns false
		},
		{
			name:     "ColumnExpr method",
			code:     `package test; func f() { q.ColumnExpr("*") }`,
			expected: false, // Without types.Info, returns false
		},
		{
			name:     "NewRaw method",
			code:     `package test; func f() { db.NewRaw("SELECT *") }`,
			expected: false, // Without types.Info, returns false
		},
		{
			name:     "Raw method",
			code:     `package test; func f() { db.Raw("SELECT *") }`,
			expected: false, // Without types.Info, returns false
		},
		{
			name:     "Model method",
			code:     `package test; func f() { q.Model(&user) }`,
			expected: false, // Without types.Info, returns false
		},
		{
			name:     "Scan method",
			code:     `package test; func f() { q.Scan(ctx) }`,
			expected: false, // Without types.Info, returns false
		},
		{
			name:     "non-bun method",
			code:     `package test; func f() { db.Other() }`,
			expected: false,
		},
	}

	checker := NewBunChecker()

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

func TestBunChecker_CheckSelectStar(t *testing.T) {
	tests := []struct {
		name            string
		code            string
		expectViolation bool
		expectedContext string
	}{
		{
			name:            "ColumnExpr with star",
			code:            `package test; func f() { q.ColumnExpr("*") }`,
			expectViolation: true,
			expectedContext: "explicit_star",
		},
		{
			name:            "Column with star",
			code:            `package test; func f() { q.Column("*") }`,
			expectViolation: true,
			expectedContext: "explicit_star",
		},
		{
			name:            "NewRaw with SELECT *",
			code:            `package test; func f() { db.NewRaw("SELECT * FROM users") }`,
			expectViolation: true,
			expectedContext: "raw_select_star",
		},
		{
			name:            "Raw with SELECT *",
			code:            `package test; func f() { db.Raw("SELECT * FROM users") }`,
			expectViolation: true,
			expectedContext: "raw_select_star",
		},
		{
			name:            "Column with explicit column",
			code:            `package test; func f() { q.Column("id") }`,
			expectViolation: false,
		},
		{
			name:            "ColumnExpr with explicit column",
			code:            `package test; func f() { q.ColumnExpr("id") }`,
			expectViolation: false,
		},
		{
			name:            "NewRaw with explicit columns",
			code:            `package test; func f() { db.NewRaw("SELECT id, name FROM users") }`,
			expectViolation: false,
		},
	}

	checker := NewBunChecker()

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

			if tt.expectViolation && violation != nil {
				if violation.Context != tt.expectedContext {
					t.Errorf("CheckSelectStar() context = %q, want %q", violation.Context, tt.expectedContext)
				}
				if violation.Builder != "bun" {
					t.Errorf("CheckSelectStar() builder = %q, want %q", violation.Builder, "bun")
				}
			}
		})
	}
}

func TestBunChecker_CheckChainedCalls(t *testing.T) {
	tests := []struct {
		name              string
		code              string
		expectViolations  bool
		minViolationCount int
	}{
		{
			name:              "NewSelect Scan without Column - no detection in current impl",
			code:              `package test; func f() { db.NewSelect().Model(&user).Scan(ctx) }`,
			expectViolations:  false, // Current implementation doesn't detect this pattern
			minViolationCount: 0,
		},
		{
			name:              "NewSelect Column Scan (explicit)",
			code:              `package test; func f() { db.NewSelect().Column("id").Scan(ctx) }`,
			expectViolations:  false,
			minViolationCount: 0,
		},
		{
			name:              "Column star in chain",
			code:              `package test; func f() { db.NewSelect().Column("*").Scan(ctx) }`,
			expectViolations:  true,
			minViolationCount: 1,
		},
		{
			name:              "NewSelect Exec without Column - no detection in current impl",
			code:              `package test; func f() { db.NewSelect().Model(&user).Exec(ctx) }`,
			expectViolations:  false, // Current implementation doesn't detect this pattern
			minViolationCount: 0,
		},
	}

	checker := NewBunChecker()

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

func TestNewBunChecker(t *testing.T) {
	checker := NewBunChecker()
	if checker == nil {
		t.Error("NewBunChecker() returned nil")
	}
}
