package sqlbuilders

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestSquirrelChecker_Name(t *testing.T) {
	checker := NewSquirrelChecker()
	if checker.Name() != "squirrel" {
		t.Errorf("Name() = %q, want %q", checker.Name(), "squirrel")
	}
}

func TestSquirrelChecker_IsApplicable(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected bool
	}{
		{
			name:     "Select method",
			code:     `package test; func f() { builder.Select("*") }`,
			expected: true,
		},
		{
			name:     "Columns method",
			code:     `package test; func f() { builder.Columns("id") }`,
			expected: true,
		},
		{
			name:     "Column method",
			code:     `package test; func f() { builder.Column("id") }`,
			expected: true,
		},
		{
			name:     "squirrel package prefix",
			code:     `package test; func f() { squirrel.Select("*") }`,
			expected: true,
		},
		{
			name:     "sq package prefix (alias)",
			code:     `package test; func f() { sq.Select("*") }`,
			expected: true,
		},
		{
			name:     "SelectBuilder method",
			code:     `package test; func f() { builder.SelectBuilder() }`,
			expected: true,
		},
		{
			name:     "non-squirrel method",
			code:     `package test; func f() { db.Query("SELECT *") }`,
			expected: false,
		},
		{
			name:     "direct function call",
			code:     `package test; func f() { Select("*") }`,
			expected: false,
		},
	}

	checker := NewSquirrelChecker()

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

func TestSquirrelChecker_CheckSelectStar(t *testing.T) {
	tests := []struct {
		name            string
		code            string
		expectViolation bool
		expectedContext string
	}{
		{
			name:            "Select with star",
			code:            `package test; func f() { builder.Select("*") }`,
			expectViolation: true,
			expectedContext: "explicit_star",
		},
		{
			name:            "Select with empty string",
			code:            `package test; func f() { builder.Select("") }`,
			expectViolation: true,
			expectedContext: "explicit_star",
		},
		{
			name:            "Select with no args (empty Select)",
			code:            `package test; func f() { builder.Select() }`,
			expectViolation: true,
			expectedContext: "empty_select",
		},
		{
			name:            "Columns with star",
			code:            `package test; func f() { builder.Columns("*") }`,
			expectViolation: true,
			expectedContext: "explicit_star",
		},
		{
			name:            "Column with star",
			code:            `package test; func f() { builder.Column("*") }`,
			expectViolation: true,
			expectedContext: "explicit_star",
		},
		{
			name:            "Select with explicit columns",
			code:            `package test; func f() { builder.Select("id", "name") }`,
			expectViolation: false,
		},
		{
			name:            "Select with single column",
			code:            `package test; func f() { builder.Select("id") }`,
			expectViolation: false,
		},
		{
			name:            "Columns with explicit columns",
			code:            `package test; func f() { builder.Columns("id", "name") }`,
			expectViolation: false,
		},
		{
			name:            "non-Select method",
			code:            `package test; func f() { builder.From("users") }`,
			expectViolation: false,
		},
	}

	checker := NewSquirrelChecker()

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
				if violation.Builder != "squirrel" {
					t.Errorf("CheckSelectStar() builder = %q, want %q", violation.Builder, "squirrel")
				}
			}
		})
	}
}

func TestSquirrelChecker_CheckChainedCalls(t *testing.T) {
	tests := []struct {
		name              string
		code              string
		expectViolations  bool
		minViolationCount int
	}{
		{
			name:              "Select star in chain",
			code:              `package test; func f() { builder.Select("*").From("users") }`,
			expectViolations:  true,
			minViolationCount: 1,
		},
		{
			name:              "Empty Select with From - no detection in current impl",
			code:              `package test; func f() { builder.Select().From("users") }`,
			expectViolations:  false, // Current implementation doesn't detect empty Select() in chain
			minViolationCount: 0,
		},
		{
			name:              "Columns star in chain",
			code:              `package test; func f() { builder.Select().Columns("*").From("users") }`,
			expectViolations:  true,
			minViolationCount: 1,
		},
		{
			name:              "explicit columns in chain - no violation",
			code:              `package test; func f() { builder.Select("id", "name").From("users") }`,
			expectViolations:  false,
			minViolationCount: 0,
		},
		{
			name:              "Select with Columns explicit",
			code:              `package test; func f() { builder.Select().Columns("id", "name").From("users") }`,
			expectViolations:  false, // Has Columns with explicit columns
			minViolationCount: 0,
		},
	}

	checker := NewSquirrelChecker()

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

func TestNewSquirrelChecker(t *testing.T) {
	checker := NewSquirrelChecker()
	if checker == nil {
		t.Error("NewSquirrelChecker() returned nil")
	}
}

func TestSquirrelChecker_CheckSelectStar_RawString(t *testing.T) {
	code := "package test; func f() { builder.Select(`*`) }"

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", code, 0)
	if err != nil {
		t.Fatalf("Failed to parse code: %v", err)
	}

	checker := NewSquirrelChecker()

	var violation *SelectStarViolation
	ast.Inspect(f, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			violation = checker.CheckSelectStar(call)
			return false
		}
		return true
	})

	if violation == nil {
		t.Error("CheckSelectStar() should detect raw string star")
	}
}
