package sqlbuilders

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/MirrexOne/unqueryvet/pkg/config"
)

func TestNewRegistry(t *testing.T) {
	tests := []struct {
		name          string
		cfg           *config.SQLBuildersConfig
		expectedCount int
		expectedHas   bool
	}{
		{
			name: "all checkers enabled",
			cfg: &config.SQLBuildersConfig{
				Squirrel:  true,
				GORM:      true,
				SQLx:      true,
				Ent:       true,
				PGX:       true,
				Bun:       true,
				SQLBoiler: true,
				Jet:       true,
			},
			expectedCount: 8,
			expectedHas:   true,
		},
		{
			name: "some checkers enabled",
			cfg: &config.SQLBuildersConfig{
				Squirrel: true,
				GORM:     true,
				SQLx:     false,
				Ent:      false,
				PGX:      false,
				Bun:      false,
			},
			expectedCount: 2,
			expectedHas:   true,
		},
		{
			name: "all checkers disabled",
			cfg: &config.SQLBuildersConfig{
				Squirrel:  false,
				GORM:      false,
				SQLx:      false,
				Ent:       false,
				PGX:       false,
				Bun:       false,
				SQLBoiler: false,
				Jet:       false,
			},
			expectedCount: 0,
			expectedHas:   false,
		},
		{
			name: "only squirrel enabled",
			cfg: &config.SQLBuildersConfig{
				Squirrel: true,
			},
			expectedCount: 1,
			expectedHas:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := NewRegistry(tt.cfg)

			if registry == nil {
				t.Fatal("NewRegistry() returned nil")
			}

			if len(registry.checkers) != tt.expectedCount {
				t.Errorf("NewRegistry() checkers count = %d, want %d", len(registry.checkers), tt.expectedCount)
			}

			if registry.HasCheckers() != tt.expectedHas {
				t.Errorf("HasCheckers() = %v, want %v", registry.HasCheckers(), tt.expectedHas)
			}
		})
	}
}

func TestRegistry_Check(t *testing.T) {
	tests := []struct {
		name              string
		code              string
		cfg               *config.SQLBuildersConfig
		expectViolations  bool
		minViolationCount int
	}{
		{
			// Without types.Info, IsApplicable returns false, so no violations are detected
			name: "squirrel Select star",
			code: `package test; func f() { squirrel.Select("*") }`,
			cfg: &config.SQLBuildersConfig{
				Squirrel: true,
			},
			expectViolations:  false, // Without types.Info, returns false
			minViolationCount: 0,
		},
		{
			// Without types.Info, IsApplicable returns false, so no violations are detected
			name: "gorm Select star",
			code: `package test; func f() { db.Select("*") }`,
			cfg: &config.SQLBuildersConfig{
				GORM: true,
			},
			expectViolations:  false, // Without types.Info, returns false
			minViolationCount: 0,
		},
		{
			name: "no violation - explicit columns",
			code: `package test; func f() { squirrel.Select("id", "name") }`,
			cfg: &config.SQLBuildersConfig{
				Squirrel: true,
			},
			expectViolations:  false,
			minViolationCount: 0,
		},
		{
			name: "no checkers enabled",
			code: `package test; func f() { squirrel.Select("*") }`,
			cfg: &config.SQLBuildersConfig{
				Squirrel: false,
			},
			expectViolations:  false,
			minViolationCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := NewRegistry(tt.cfg)

			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, "test.go", tt.code, 0)
			if err != nil {
				t.Fatalf("Failed to parse code: %v", err)
			}

			var violations []*SelectStarViolation
			ast.Inspect(f, func(n ast.Node) bool {
				if call, ok := n.(*ast.CallExpr); ok {
					violations = append(violations, registry.Check(nil, call)...)
				}
				return true
			})

			hasViolations := len(violations) > 0
			if hasViolations != tt.expectViolations {
				t.Errorf("Check() hasViolations = %v, want %v", hasViolations, tt.expectViolations)
			}

			if len(violations) < tt.minViolationCount {
				t.Errorf("Check() violations = %d, want at least %d", len(violations), tt.minViolationCount)
			}
		})
	}
}

func TestRegistry_HasCheckers(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *config.SQLBuildersConfig
		expected bool
	}{
		{
			name: "has checkers",
			cfg: &config.SQLBuildersConfig{
				Squirrel: true,
			},
			expected: true,
		},
		{
			name: "no checkers",
			cfg: &config.SQLBuildersConfig{
				Squirrel: false,
				GORM:     false,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := NewRegistry(tt.cfg)
			if registry.HasCheckers() != tt.expected {
				t.Errorf("HasCheckers() = %v, want %v", registry.HasCheckers(), tt.expected)
			}
		})
	}
}

func TestSelectStarViolation(t *testing.T) {
	violation := &SelectStarViolation{
		Pos:     100,
		End:     200,
		Message: "test message",
		Builder: "test_builder",
		Context: "test_context",
	}

	if violation.Pos != 100 {
		t.Errorf("Pos = %d, want 100", violation.Pos)
	}

	if violation.End != 200 {
		t.Errorf("End = %d, want 200", violation.End)
	}

	if violation.Message != "test message" {
		t.Errorf("Message = %q, want %q", violation.Message, "test message")
	}

	if violation.Builder != "test_builder" {
		t.Errorf("Builder = %q, want %q", violation.Builder, "test_builder")
	}

	if violation.Context != "test_context" {
		t.Errorf("Context = %q, want %q", violation.Context, "test_context")
	}
}

func TestRegistry_Check_MultipleCheckers(t *testing.T) {
	// Enable multiple checkers and verify they all work
	cfg := &config.SQLBuildersConfig{
		Squirrel: true,
		GORM:     true,
		SQLx:     true,
	}

	registry := NewRegistry(cfg)

	if len(registry.checkers) != 3 {
		t.Errorf("Expected 3 checkers, got %d", len(registry.checkers))
	}

	// Verify each checker is of the correct type
	checkerNames := make(map[string]bool)
	for _, checker := range registry.checkers {
		checkerNames[checker.Name()] = true
	}

	expectedCheckers := []string{"squirrel", "gorm", "sqlx"}
	for _, name := range expectedCheckers {
		if !checkerNames[name] {
			t.Errorf("Expected checker %q not found", name)
		}
	}
}

func TestRegistry_Check_NonApplicableCall(t *testing.T) {
	cfg := &config.SQLBuildersConfig{
		Squirrel: true,
	}

	registry := NewRegistry(cfg)

	// Code with a call that doesn't match any checker
	code := `package test; func f() { someOtherFunc("test") }`

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", code, 0)
	if err != nil {
		t.Fatalf("Failed to parse code: %v", err)
	}

	var violations []*SelectStarViolation
	ast.Inspect(f, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			violations = append(violations, registry.Check(nil, call)...)
		}
		return true
	})

	if len(violations) != 0 {
		t.Errorf("Expected no violations for non-applicable call, got %d", len(violations))
	}
}
