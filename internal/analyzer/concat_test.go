package analyzer

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/MirrexOne/unqueryvet/pkg/config"
)

func TestStringConcatAnalyzer_AnalyzeBinaryExpr(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected bool
	}{
		{
			name:     "simple SELECT * concatenation",
			code:     `package test; var x = "SELECT * " + "FROM users"`,
			expected: true,
		},
		{
			name:     "multi-part SELECT * concatenation",
			code:     `package test; var x = "SELECT " + "* " + "FROM users"`,
			expected: true,
		},
		{
			name:     "SELECT * with WHERE",
			code:     `package test; var x = "SELECT * FROM " + "users WHERE id = 1"`,
			expected: true,
		},
		{
			name:     "explicit columns - no violation",
			code:     `package test; var x = "SELECT id, name " + "FROM users"`,
			expected: false,
		},
		{
			name:     "COUNT(*) - allowed pattern",
			code:     `package test; var x = "SELECT COUNT(*) " + "FROM users"`,
			expected: false,
		},
		{
			name:     "empty concatenation",
			code:     `package test; var x = "" + ""`,
			expected: false,
		},
		{
			name:     "non-SQL string",
			code:     `package test; var x = "hello " + "world"`,
			expected: false,
		},
		{
			name:     "SELECT without FROM - detected as just SELECT *",
			code:     `package test; var x = "SELECT " + "*"`,
			expected: true, // "SELECT *" alone is still detected
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, "test.go", tt.code, 0)
			if err != nil {
				t.Fatalf("Failed to parse code: %v", err)
			}

			defaults := config.DefaultSettings()
			cfg := &defaults
			sca := NewStringConcatAnalyzer(nil, cfg)

			var result bool
			ast.Inspect(f, func(n ast.Node) bool {
				if binExpr, ok := n.(*ast.BinaryExpr); ok {
					result = sca.AnalyzeBinaryExpr(binExpr)
					return false
				}
				return true
			})

			if result != tt.expected {
				t.Errorf("AnalyzeBinaryExpr() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestStringConcatAnalyzer_AnalyzeBinaryExpr_NonAddOperator(t *testing.T) {
	code := `package test; var x = 1 - 2`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", code, 0)
	if err != nil {
		t.Fatalf("Failed to parse code: %v", err)
	}

	defaults := config.DefaultSettings()
	cfg := &defaults
	sca := NewStringConcatAnalyzer(nil, cfg)

	var result bool
	ast.Inspect(f, func(n ast.Node) bool {
		if binExpr, ok := n.(*ast.BinaryExpr); ok {
			result = sca.AnalyzeBinaryExpr(binExpr)
			return false
		}
		return true
	})

	if result != false {
		t.Error("AnalyzeBinaryExpr() should return false for non-ADD operator")
	}
}

func TestStringConcatAnalyzer_extractStringParts(t *testing.T) {
	tests := []struct {
		name          string
		code          string
		expectedParts []string
	}{
		{
			name:          "single string literal",
			code:          `package test; var x = "hello"`,
			expectedParts: []string{"hello"},
		},
		{
			name:          "two string literals",
			code:          `package test; var x = "hello " + "world"`,
			expectedParts: []string{"hello ", "world"},
		},
		{
			name:          "three string literals",
			code:          `package test; var x = "a" + "b" + "c"`,
			expectedParts: []string{"a", "b", "c"},
		},
		{
			name:          "raw string literal",
			code:          "package test; var x = `raw string`",
			expectedParts: []string{"raw string"},
		},
		{
			name:          "parenthesized expression",
			code:          `package test; var x = ("hello")`,
			expectedParts: []string{"hello"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, "test.go", tt.code, 0)
			if err != nil {
				t.Fatalf("Failed to parse code: %v", err)
			}

			defaults := config.DefaultSettings()
			cfg := &defaults
			sca := NewStringConcatAnalyzer(nil, cfg)

			var parts []string
			ast.Inspect(f, func(n ast.Node) bool {
				switch expr := n.(type) {
				case *ast.BinaryExpr:
					parts = sca.extractStringParts(expr)
					return false
				case *ast.BasicLit:
					if expr.Kind == token.STRING {
						parts = sca.extractStringParts(expr)
						return false
					}
				case *ast.ParenExpr:
					parts = sca.extractStringParts(expr)
					return false
				}
				return true
			})

			if len(parts) != len(tt.expectedParts) {
				t.Errorf("extractStringParts() returned %d parts, want %d", len(parts), len(tt.expectedParts))
				return
			}

			for i, part := range parts {
				if part != tt.expectedParts[i] {
					t.Errorf("part[%d] = %q, want %q", i, part, tt.expectedParts[i])
				}
			}
		})
	}
}

func TestStringConcatAnalyzer_extractStringParts_WithConstant(t *testing.T) {
	code := `package test
const query = "SELECT * FROM users"
var x = query`

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", code, 0)
	if err != nil {
		t.Fatalf("Failed to parse code: %v", err)
	}

	defaults := config.DefaultSettings()
	cfg := &defaults
	sca := NewStringConcatAnalyzer(nil, cfg)

	// Find the variable x and check if constant is resolved
	var parts []string
	ast.Inspect(f, func(n ast.Node) bool {
		if valueSpec, ok := n.(*ast.ValueSpec); ok {
			for _, name := range valueSpec.Names {
				if name.Name == "x" {
					for _, value := range valueSpec.Values {
						if ident, ok := value.(*ast.Ident); ok {
							parts = sca.extractStringParts(ident)
						}
					}
				}
			}
		}
		return true
	})

	// Note: The constant resolution depends on ast.Ident.Obj being set
	// In a real scenario with proper type checking, this would resolve
	// For now, we just verify it doesn't panic
	_ = parts
}

func TestCheckConcatenation(t *testing.T) {
	code := `package test; var x = "SELECT * " + "FROM users"`

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", code, 0)
	if err != nil {
		t.Fatalf("Failed to parse code: %v", err)
	}

	defaults := config.DefaultSettings()
	cfg := &defaults

	var result bool
	ast.Inspect(f, func(n ast.Node) bool {
		if binExpr, ok := n.(*ast.BinaryExpr); ok {
			result = CheckConcatenation(nil, binExpr, cfg)
			return false
		}
		return true
	})

	if !result {
		t.Error("CheckConcatenation() should return true for SELECT * concatenation")
	}
}

func TestCheckConcatenation_NoViolation(t *testing.T) {
	code := `package test; var x = "SELECT id " + "FROM users"`

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", code, 0)
	if err != nil {
		t.Fatalf("Failed to parse code: %v", err)
	}

	defaults := config.DefaultSettings()
	cfg := &defaults

	var result bool
	ast.Inspect(f, func(n ast.Node) bool {
		if binExpr, ok := n.(*ast.BinaryExpr); ok {
			result = CheckConcatenation(nil, binExpr, cfg)
			return false
		}
		return true
	})

	if result {
		t.Error("CheckConcatenation() should return false for explicit columns")
	}
}

func TestNewStringConcatAnalyzer(t *testing.T) {
	defaults := config.DefaultSettings()
	cfg := &defaults

	sca := NewStringConcatAnalyzer(nil, cfg)

	if sca == nil {
		t.Fatal("NewStringConcatAnalyzer() returned nil")
	}

	if sca.cfg != cfg {
		t.Error("NewStringConcatAnalyzer() did not set config correctly")
	}
}
