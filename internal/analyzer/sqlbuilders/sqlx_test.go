package sqlbuilders

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestSQLxChecker_Name(t *testing.T) {
	checker := NewSQLxChecker()
	if checker.Name() != "sqlx" {
		t.Errorf("Name() = %q, want %q", checker.Name(), "sqlx")
	}
}

func TestSQLxChecker_IsApplicable(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected bool
	}{
		{
			name:     "Select method",
			code:     `package test; func f() { db.Select(&dest, "query") }`,
			expected: true,
		},
		{
			name:     "Get method",
			code:     `package test; func f() { db.Get(&dest, "query") }`,
			expected: true,
		},
		{
			name:     "Queryx method",
			code:     `package test; func f() { db.Queryx("query") }`,
			expected: true,
		},
		{
			name:     "QueryRowx method",
			code:     `package test; func f() { db.QueryRowx("query") }`,
			expected: true,
		},
		{
			name:     "NamedQuery method",
			code:     `package test; func f() { db.NamedQuery("query", arg) }`,
			expected: true,
		},
		{
			name:     "NamedExec method",
			code:     `package test; func f() { db.NamedExec("query", arg) }`,
			expected: true,
		},
		{
			name:     "MustExec method",
			code:     `package test; func f() { db.MustExec("query") }`,
			expected: true,
		},
		{
			name:     "non-sqlx method",
			code:     `package test; func f() { db.SomeOther() }`,
			expected: false,
		},
	}

	checker := NewSQLxChecker()

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

func TestSQLxChecker_CheckSelectStar(t *testing.T) {
	tests := []struct {
		name            string
		code            string
		expectViolation bool
	}{
		{
			name:            "Select with SELECT *",
			code:            `package test; func f() { db.Select(&dest, "SELECT * FROM users") }`,
			expectViolation: true,
		},
		{
			name:            "Get with SELECT *",
			code:            `package test; func f() { db.Get(&dest, "SELECT * FROM users") }`,
			expectViolation: true,
		},
		{
			name:            "Queryx with SELECT *",
			code:            `package test; func f() { db.Queryx("SELECT * FROM users") }`,
			expectViolation: true,
		},
		{
			name:            "QueryRowx with SELECT *",
			code:            `package test; func f() { db.QueryRowx("SELECT * FROM users") }`,
			expectViolation: true,
		},
		{
			name:            "NamedQuery with SELECT *",
			code:            `package test; func f() { db.NamedQuery("SELECT * FROM users WHERE id = :id", arg) }`,
			expectViolation: true,
		},
		{
			name:            "MustExec with SELECT *",
			code:            `package test; func f() { db.MustExec("SELECT * FROM users") }`,
			expectViolation: true,
		},
		{
			name:            "Select with explicit columns",
			code:            `package test; func f() { db.Select(&dest, "SELECT id, name FROM users") }`,
			expectViolation: false,
		},
		{
			name:            "Queryx with explicit columns",
			code:            `package test; func f() { db.Queryx("SELECT id FROM users") }`,
			expectViolation: false,
		},
	}

	checker := NewSQLxChecker()

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

			if violation != nil && violation.Builder != "sqlx" {
				t.Errorf("CheckSelectStar() builder = %q, want %q", violation.Builder, "sqlx")
			}
		})
	}
}

func TestSQLxChecker_CheckChainedCalls(t *testing.T) {
	checker := NewSQLxChecker()

	// sqlx doesn't have chaining patterns, should return nil
	code := `package test; func f() { db.Select(&dest, "query") }`

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
		t.Error("CheckChainedCalls() should return nil or empty slice for sqlx")
	}
}

func TestNewSQLxChecker(t *testing.T) {
	checker := NewSQLxChecker()
	if checker == nil {
		t.Error("NewSQLxChecker() returned nil")
	}
}

func TestSQLxChecker_CheckSelectStar_NotEnoughArgs(t *testing.T) {
	// Test case where query arg index is out of bounds
	code := `package test; func f() { db.Select() }`

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", code, 0)
	if err != nil {
		t.Fatalf("Failed to parse code: %v", err)
	}

	checker := NewSQLxChecker()

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
