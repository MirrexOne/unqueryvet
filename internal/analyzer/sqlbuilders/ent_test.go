package sqlbuilders

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestEntChecker_Name(t *testing.T) {
	checker := NewEntChecker()
	if checker.Name() != "ent" {
		t.Errorf("Name() = %q, want %q", checker.Name(), "ent")
	}
}

func TestEntChecker_IsApplicable(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected bool
	}{
		{
			name:     "Query method",
			code:     `package test; func f() { client.User.Query() }`,
			expected: true,
		},
		{
			name:     "All method",
			code:     `package test; func f() { q.All(ctx) }`,
			expected: true,
		},
		{
			name:     "Only method",
			code:     `package test; func f() { q.Only(ctx) }`,
			expected: true,
		},
		{
			name:     "OnlyX method",
			code:     `package test; func f() { q.OnlyX(ctx) }`,
			expected: true,
		},
		{
			name:     "First method",
			code:     `package test; func f() { q.First(ctx) }`,
			expected: true,
		},
		{
			name:     "FirstX method",
			code:     `package test; func f() { q.FirstX(ctx) }`,
			expected: true,
		},
		{
			name:     "Select method",
			code:     `package test; func f() { q.Select("id") }`,
			expected: true,
		},
		{
			name:     "non-ent method",
			code:     `package test; func f() { db.Other() }`,
			expected: false,
		},
	}

	checker := NewEntChecker()

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

func TestEntChecker_CheckSelectStar(t *testing.T) {
	// ent doesn't have explicit SELECT * patterns
	// The implicit SELECT * happens in chain calls
	checker := NewEntChecker()

	code := `package test; func f() { client.User.Query() }`

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", code, 0)
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

	// CheckSelectStar should always return nil for ent
	if violation != nil {
		t.Error("CheckSelectStar() should return nil for ent")
	}
}

func TestEntChecker_CheckChainedCalls(t *testing.T) {
	tests := []struct {
		name              string
		code              string
		expectViolations  bool
		minViolationCount int
	}{
		{
			name:              "Query All without Select - no detection in current impl",
			code:              `package test; func f() { client.User.Query().All(ctx) }`,
			expectViolations:  false, // Current implementation doesn't detect this pattern
			minViolationCount: 0,
		},
		{
			name:              "Query Only without Select - no detection in current impl",
			code:              `package test; func f() { client.User.Query().Only(ctx) }`,
			expectViolations:  false, // Current implementation doesn't detect this pattern
			minViolationCount: 0,
		},
		{
			name:              "Query OnlyX without Select - no detection in current impl",
			code:              `package test; func f() { client.User.Query().OnlyX(ctx) }`,
			expectViolations:  false, // Current implementation doesn't detect this pattern
			minViolationCount: 0,
		},
		{
			name:              "Query First without Select - no detection in current impl",
			code:              `package test; func f() { client.User.Query().First(ctx) }`,
			expectViolations:  false, // Current implementation doesn't detect this pattern
			minViolationCount: 0,
		},
		{
			name:              "Query FirstX without Select - no detection in current impl",
			code:              `package test; func f() { client.User.Query().FirstX(ctx) }`,
			expectViolations:  false, // Current implementation doesn't detect this pattern
			minViolationCount: 0,
		},
		{
			name:              "Query Select All (explicit columns)",
			code:              `package test; func f() { client.User.Query().Select("id", "name").All(ctx) }`,
			expectViolations:  false,
			minViolationCount: 0,
		},
		{
			name:              "Query Select Only (explicit columns)",
			code:              `package test; func f() { client.User.Query().Select("id").Only(ctx) }`,
			expectViolations:  false,
			minViolationCount: 0,
		},
	}

	checker := NewEntChecker()

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

			// Check that violations have correct builder name
			for _, v := range allViolations {
				if v.Builder != "ent" {
					t.Errorf("Violation builder = %q, want %q", v.Builder, "ent")
				}
			}
		})
	}
}

func TestNewEntChecker(t *testing.T) {
	checker := NewEntChecker()
	if checker == nil {
		t.Error("NewEntChecker() returned nil")
	}
}
