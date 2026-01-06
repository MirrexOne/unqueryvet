package sqlbuilders

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestPGXChecker_Name(t *testing.T) {
	checker := NewPGXChecker()
	if checker.Name() != "pgx" {
		t.Errorf("Name() = %q, want %q", checker.Name(), "pgx")
	}
}

func TestPGXChecker_IsApplicable(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected bool
	}{
		{
			name:     "Query method",
			code:     `package test; func f() { conn.Query(ctx, "SELECT *") }`,
			expected: true,
		},
		{
			name:     "QueryRow method",
			code:     `package test; func f() { conn.QueryRow(ctx, "SELECT *") }`,
			expected: true,
		},
		{
			name:     "QueryFunc method",
			code:     `package test; func f() { conn.QueryFunc(ctx, "SELECT *", nil, nil) }`,
			expected: true,
		},
		{
			name:     "Exec method",
			code:     `package test; func f() { conn.Exec(ctx, "SELECT *") }`,
			expected: true,
		},
		{
			name:     "Prepare method",
			code:     `package test; func f() { conn.Prepare(ctx, "stmt", "SELECT *") }`,
			expected: true,
		},
		{
			name:     "non-pgx method",
			code:     `package test; func f() { db.Other() }`,
			expected: false,
		},
	}

	checker := NewPGXChecker()

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
					result = checker.IsApplicable(call)
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

func TestPGXChecker_CheckSelectStar(t *testing.T) {
	tests := []struct {
		name            string
		code            string
		expectViolation bool
	}{
		{
			name:            "Query with SELECT *",
			code:            `package test; func f() { conn.Query(ctx, "SELECT * FROM users") }`,
			expectViolation: true,
		},
		{
			name:            "QueryRow with SELECT *",
			code:            `package test; func f() { conn.QueryRow(ctx, "SELECT * FROM users") }`,
			expectViolation: true,
		},
		{
			name:            "Exec with SELECT *",
			code:            `package test; func f() { conn.Exec(ctx, "SELECT * FROM users") }`,
			expectViolation: true,
		},
		{
			name:            "Prepare with SELECT * - query at index 2 not checked",
			code:            `package test; func f() { conn.Prepare(ctx, "stmt", "SELECT * FROM users") }`,
			expectViolation: false, // Prepare has query at index 2, but checker uses index 1
		},
		{
			name:            "Query with explicit columns",
			code:            `package test; func f() { conn.Query(ctx, "SELECT id, name FROM users") }`,
			expectViolation: false,
		},
		{
			name:            "QueryRow with explicit columns",
			code:            `package test; func f() { conn.QueryRow(ctx, "SELECT id FROM users") }`,
			expectViolation: false,
		},
	}

	checker := NewPGXChecker()

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

			if violation != nil && violation.Builder != "pgx" {
				t.Errorf("CheckSelectStar() builder = %q, want %q", violation.Builder, "pgx")
			}
		})
	}
}

func TestPGXChecker_CheckChainedCalls(t *testing.T) {
	checker := NewPGXChecker()

	// pgx doesn't have chaining patterns, should return nil
	code := `package test; func f() { conn.Query(ctx, "query") }`

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", code, 0)
	if err != nil {
		t.Fatalf("Failed to parse code: %v", err)
	}

	var violations []*SelectStarViolation
	ast.Inspect(f, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			violations = checker.CheckChainedCalls(call)
			return false
		}
		return true
	})

	if len(violations) > 0 {
		t.Error("CheckChainedCalls() should return nil or empty slice for pgx")
	}
}

func TestNewPGXChecker(t *testing.T) {
	checker := NewPGXChecker()
	if checker == nil {
		t.Error("NewPGXChecker() returned nil")
	}
}

func TestPGXChecker_CheckSelectStar_NotEnoughArgs(t *testing.T) {
	// Test case where query arg index is out of bounds
	code := `package test; func f() { conn.Query(ctx) }`

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", code, 0)
	if err != nil {
		t.Fatalf("Failed to parse code: %v", err)
	}

	checker := NewPGXChecker()

	var violation *SelectStarViolation
	ast.Inspect(f, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			violation = checker.CheckSelectStar(call)
			return false
		}
		return true
	})

	if violation != nil {
		t.Error("CheckSelectStar() should return nil when args are insufficient")
	}
}

func TestPGXChecker_CheckSelectStar_CaseInsensitive(t *testing.T) {
	tests := []struct {
		code            string
		expectViolation bool
	}{
		{
			code:            `package test; func f() { conn.Query(ctx, "select * from users") }`,
			expectViolation: true,
		},
		{
			code:            `package test; func f() { conn.Query(ctx, "SELECT * FROM users") }`,
			expectViolation: true,
		},
		{
			code:            `package test; func f() { conn.Query(ctx, "Select * From users") }`,
			expectViolation: true,
		},
	}

	checker := NewPGXChecker()

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
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
		})
	}
}
