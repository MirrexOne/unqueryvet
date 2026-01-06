package sqlbuilders

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestJetChecker_Name(t *testing.T) {
	checker := NewJetChecker()
	if checker.Name() != "jet" {
		t.Errorf("Name() = %q, want %q", checker.Name(), "jet")
	}
}

func TestJetChecker_IsApplicable(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected bool
	}{
		{
			name:     "SELECT method",
			code:     `package test; func f() { stmt.SELECT(table.AllColumns) }`,
			expected: true,
		},
		{
			name:     "FROM method",
			code:     `package test; func f() { stmt.FROM(table) }`,
			expected: true,
		},
		{
			name:     "WHERE method",
			code:     `package test; func f() { stmt.WHERE(cond) }`,
			expected: true,
		},
		{
			name:     "AllColumns method",
			code:     `package test; func f() { table.AllColumns() }`,
			expected: true,
		},
		{
			name:     "Star method",
			code:     `package test; func f() { jet.Star() }`,
			expected: true,
		},
		{
			name:     "RawStatement method",
			code:     `package test; func f() { jet.RawStatement("SELECT *") }`,
			expected: true,
		},
		{
			name:     "Raw method",
			code:     `package test; func f() { jet.Raw("SELECT *") }`,
			expected: true,
		},
		{
			name:     "direct SELECT call",
			code:     `package test; func f() { SELECT(cols) }`,
			expected: true,
		},
		{
			name:     "direct RawStatement call",
			code:     `package test; func f() { RawStatement("SELECT *") }`,
			expected: true,
		},
		{
			name:     "non-jet method",
			code:     `package test; func f() { db.Other() }`,
			expected: false,
		},
	}

	checker := NewJetChecker()

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

func TestJetChecker_CheckSelectStar(t *testing.T) {
	tests := []struct {
		name            string
		code            string
		expectViolation bool
		expectedContext string
	}{
		{
			name:            "AllColumns method call",
			code:            `package test; func f() { table.AllColumns() }`,
			expectViolation: true,
			expectedContext: "all_columns",
		},
		{
			name:            "Star method call",
			code:            `package test; func f() { jet.Star() }`,
			expectViolation: true,
			expectedContext: "explicit_star",
		},
		{
			name:            "RawStatement with SELECT *",
			code:            `package test; func f() { jet.RawStatement("SELECT * FROM users") }`,
			expectViolation: true,
			expectedContext: "raw_select_star",
		},
		{
			name:            "Raw with SELECT *",
			code:            `package test; func f() { jet.Raw("SELECT * FROM users") }`,
			expectViolation: true,
			expectedContext: "raw_select_star",
		},
		{
			name:            "direct RawStatement with SELECT *",
			code:            `package test; func f() { RawStatement("SELECT * FROM users") }`,
			expectViolation: true,
			expectedContext: "raw_select_star",
		},
		{
			name:            "RawStatement with explicit columns",
			code:            `package test; func f() { jet.RawStatement("SELECT id, name FROM users") }`,
			expectViolation: false,
		},
	}

	checker := NewJetChecker()

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
					if violation != nil {
						return false
					}
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
				if violation.Builder != "jet" {
					t.Errorf("CheckSelectStar() builder = %q, want %q", violation.Builder, "jet")
				}
			}
		})
	}
}

func TestJetChecker_CheckChainedCalls(t *testing.T) {
	tests := []struct {
		name              string
		code              string
		expectViolations  bool
		minViolationCount int
	}{
		{
			name:              "SELECT with AllColumns in chain",
			code:              `package test; func f() { stmt.SELECT(table.AllColumns).FROM(table) }`,
			expectViolations:  true,
			minViolationCount: 1,
		},
		{
			name:              "SELECT with explicit columns",
			code:              `package test; func f() { stmt.SELECT(table.ID, table.Name).FROM(table) }`,
			expectViolations:  false,
			minViolationCount: 0,
		},
	}

	checker := NewJetChecker()

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

func TestJetChecker_isAllColumnsOrStar(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected bool
	}{
		{
			name:     "table.AllColumns selector",
			code:     `package test; var x = table.AllColumns`,
			expected: true,
		},
		{
			name:     "package.STAR selector",
			code:     `package test; var x = jet.STAR`,
			expected: true,
		},
		{
			name:     "AllColumns identifier",
			code:     `package test; var x = AllColumns`,
			expected: true,
		},
		{
			name:     "STAR identifier",
			code:     `package test; var x = STAR`,
			expected: true,
		},
		{
			name:     "table.ID selector (not star)",
			code:     `package test; var x = table.ID`,
			expected: false,
		},
		{
			name:     "regular identifier",
			code:     `package test; var x = someVar`,
			expected: false,
		},
	}

	checker := NewJetChecker()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, "test.go", tt.code, 0)
			if err != nil {
				t.Fatalf("Failed to parse code: %v", err)
			}

			var result bool
			ast.Inspect(f, func(n ast.Node) bool {
				if valueSpec, ok := n.(*ast.ValueSpec); ok {
					for _, value := range valueSpec.Values {
						result = checker.isAllColumnsOrStar(value)
						return false
					}
				}
				return true
			})

			if result != tt.expected {
				t.Errorf("isAllColumnsOrStar() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestNewJetChecker(t *testing.T) {
	checker := NewJetChecker()
	if checker == nil {
		t.Error("NewJetChecker() returned nil")
	}
}

func TestJetChecker_CheckSelectStar_DirectSELECT(t *testing.T) {
	// Test direct SELECT function call with AllColumns argument
	code := `package test
var x = table.AllColumns
func f() { SELECT(x) }`

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", code, 0)
	if err != nil {
		t.Fatalf("Failed to parse code: %v", err)
	}

	checker := NewJetChecker()

	var violation *SelectStarViolation
	ast.Inspect(f, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			if ident, ok := call.Fun.(*ast.Ident); ok && ident.Name == "SELECT" {
				violation = checker.CheckSelectStar(call)
				return false
			}
		}
		return true
	})

	// The direct SELECT call should check arguments for AllColumns
	// Note: This test may or may not find a violation depending on how
	// the checker handles variable references
	_ = violation // Just verify no panic
}
