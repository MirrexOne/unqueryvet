package analyzer

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/MirrexOne/unqueryvet/pkg/config"
)

func TestNewFilterContext(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *config.UnqueryvetSettings
		expectError bool
	}{
		{
			name: "valid patterns",
			cfg: &config.UnqueryvetSettings{
				IgnoredFunctions: []string{"debug.*", "test.Query"},
				IgnoredFiles:     []string{"*_test.go", "testdata/**"},
				AllowedPatterns:  []string{`(?i)COUNT\(\s*\*\s*\)`},
			},
			expectError: false,
		},
		{
			name: "invalid function pattern",
			cfg: &config.UnqueryvetSettings{
				IgnoredFunctions: []string{"[invalid"},
				IgnoredFiles:     []string{},
				AllowedPatterns:  []string{},
			},
			expectError: true,
		},
		{
			name: "invalid allowed pattern",
			cfg: &config.UnqueryvetSettings{
				IgnoredFunctions: []string{},
				IgnoredFiles:     []string{},
				AllowedPatterns:  []string{"[invalid"},
			},
			expectError: true,
		},
		{
			name: "empty patterns",
			cfg: &config.UnqueryvetSettings{
				IgnoredFunctions: []string{},
				IgnoredFiles:     []string{},
				AllowedPatterns:  []string{},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fc, err := NewFilterContext(tt.cfg)
			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if fc == nil {
					t.Error("expected FilterContext, got nil")
				}
			}
		})
	}
}

func TestFilterContext_IsIgnoredFunction(t *testing.T) {
	tests := []struct {
		name             string
		ignoredFunctions []string
		code             string
		expected         bool
	}{
		{
			name:             "match debug pattern",
			ignoredFunctions: []string{"debug.*"},
			code:             `package test; func f() { debug.Log("test") }`,
			expected:         true,
		},
		{
			name:             "match exact function",
			ignoredFunctions: []string{"test.Query"},
			code:             `package test; func f() { test.Query("SELECT *") }`,
			expected:         true,
		},
		{
			name:             "no match",
			ignoredFunctions: []string{"debug.*"},
			code:             `package test; func f() { db.Query("SELECT *") }`,
			expected:         false,
		},
		{
			name:             "empty patterns",
			ignoredFunctions: []string{},
			code:             `package test; func f() { debug.Log("test") }`,
			expected:         false,
		},
		{
			name:             "match receiver method",
			ignoredFunctions: []string{"Logger.Debug"},
			code:             `package test; func f() { Logger.Debug("test") }`,
			expected:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.UnqueryvetSettings{
				IgnoredFunctions: tt.ignoredFunctions,
				IgnoredFiles:     []string{},
				AllowedPatterns:  []string{},
			}

			fc, err := NewFilterContext(cfg)
			if err != nil {
				t.Fatalf("Failed to create FilterContext: %v", err)
			}

			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, "test.go", tt.code, 0)
			if err != nil {
				t.Fatalf("Failed to parse code: %v", err)
			}

			var result bool
			ast.Inspect(f, func(n ast.Node) bool {
				if call, ok := n.(*ast.CallExpr); ok {
					result = fc.IsIgnoredFunction(call)
					return false
				}
				return true
			})

			if result != tt.expected {
				t.Errorf("IsIgnoredFunction() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestFilterContext_IsIgnoredFile(t *testing.T) {
	tests := []struct {
		name         string
		ignoredFiles []string
		filePath     string
		expected     bool
	}{
		{
			name:         "match test file pattern",
			ignoredFiles: []string{"*_test.go"},
			filePath:     "foo_test.go",
			expected:     true,
		},
		{
			name:         "match test file full path",
			ignoredFiles: []string{"*_test.go"},
			filePath:     "/path/to/foo_test.go",
			expected:     true,
		},
		{
			name:         "no match regular file",
			ignoredFiles: []string{"*_test.go"},
			filePath:     "foo.go",
			expected:     false,
		},
		{
			name:         "double star pattern",
			ignoredFiles: []string{"testdata/**"},
			filePath:     "testdata/foo/bar.go",
			expected:     true,
		},
		{
			name:         "double star pattern root",
			ignoredFiles: []string{"testdata/**"},
			filePath:     "testdata/file.go",
			expected:     true,
		},
		{
			name:         "mock file pattern",
			ignoredFiles: []string{"mock_*.go"},
			filePath:     "mock_user.go",
			expected:     true,
		},
		{
			name:         "empty patterns",
			ignoredFiles: []string{},
			filePath:     "foo_test.go",
			expected:     false,
		},
		{
			name:         "windows path",
			ignoredFiles: []string{"*_test.go"},
			filePath:     "C:\\path\\to\\foo_test.go",
			expected:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.UnqueryvetSettings{
				IgnoredFunctions: []string{},
				IgnoredFiles:     tt.ignoredFiles,
				AllowedPatterns:  []string{},
			}

			fc, err := NewFilterContext(cfg)
			if err != nil {
				t.Fatalf("Failed to create FilterContext: %v", err)
			}

			result := fc.IsIgnoredFile(tt.filePath)

			if result != tt.expected {
				t.Errorf("IsIgnoredFile(%q) = %v, want %v", tt.filePath, result, tt.expected)
			}
		})
	}
}

func TestFilterContext_IsAllowedPattern(t *testing.T) {
	tests := []struct {
		name            string
		allowedPatterns []string
		query           string
		expected        bool
	}{
		{
			name:            "match COUNT(*)",
			allowedPatterns: []string{`(?i)COUNT\(\s*\*\s*\)`},
			query:           "SELECT COUNT(*) FROM users",
			expected:        true,
		},
		{
			name:            "match COUNT with spaces",
			allowedPatterns: []string{`(?i)COUNT\(\s*\*\s*\)`},
			query:           "SELECT COUNT( * ) FROM users",
			expected:        true,
		},
		{
			name:            "no match SELECT *",
			allowedPatterns: []string{`(?i)COUNT\(\s*\*\s*\)`},
			query:           "SELECT * FROM users",
			expected:        false,
		},
		{
			name:            "match information_schema",
			allowedPatterns: []string{`(?i)INFORMATION_SCHEMA`},
			query:           "SELECT * FROM INFORMATION_SCHEMA.TABLES",
			expected:        true,
		},
		{
			name:            "empty patterns",
			allowedPatterns: []string{},
			query:           "SELECT COUNT(*) FROM users",
			expected:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.UnqueryvetSettings{
				IgnoredFunctions: []string{},
				IgnoredFiles:     []string{},
				AllowedPatterns:  tt.allowedPatterns,
			}

			fc, err := NewFilterContext(cfg)
			if err != nil {
				t.Fatalf("Failed to create FilterContext: %v", err)
			}

			result := fc.IsAllowedPattern(tt.query)

			if result != tt.expected {
				t.Errorf("IsAllowedPattern(%q) = %v, want %v", tt.query, result, tt.expected)
			}
		})
	}
}

func TestExtractFunctionName(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected string
	}{
		{
			name:     "package.Function",
			code:     `package test; func f() { fmt.Println("test") }`,
			expected: "fmt.Println",
		},
		{
			name:     "direct function call",
			code:     `package test; func f() { println("test") }`,
			expected: "println",
		},
		{
			name:     "receiver.Method",
			code:     `package test; func f() { obj.Method("test") }`,
			expected: "obj.Method",
		},
		{
			name:     "nested pkg.subpkg.Function",
			code:     `package test; func f() { pkg.sub.Func("test") }`,
			expected: "pkg.sub.Func",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, "test.go", tt.code, 0)
			if err != nil {
				t.Fatalf("Failed to parse code: %v", err)
			}

			var result string
			ast.Inspect(f, func(n ast.Node) bool {
				if call, ok := n.(*ast.CallExpr); ok {
					result = extractFunctionName(call)
					return false
				}
				return true
			})

			if result != tt.expected {
				t.Errorf("extractFunctionName() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestMatchDoubleStarPattern(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		path     string
		expected bool
	}{
		{
			name:     "testdata/** matches nested path",
			pattern:  "testdata/**",
			path:     "testdata/foo/bar.go",
			expected: true,
		},
		{
			name:     "testdata/** matches direct child",
			pattern:  "testdata/**",
			path:     "testdata/file.go",
			expected: true,
		},
		{
			name:     "testdata/** does not match other dir",
			pattern:  "testdata/**",
			path:     "src/foo/bar.go",
			expected: false,
		},
		{
			name:     "vendor/** matches vendor path",
			pattern:  "vendor/**",
			path:     "vendor/github.com/pkg/file.go",
			expected: true,
		},
		{
			name:     "pattern with suffix",
			pattern:  "testdata/**/*.go",
			path:     "testdata/foo/bar.go",
			expected: true,
		},
		{
			name:     "pattern without **",
			pattern:  "testdata/file.go",
			path:     "testdata/file.go",
			expected: false, // This function only handles ** patterns
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchDoubleStarPattern(tt.pattern, tt.path)
			if result != tt.expected {
				t.Errorf("matchDoubleStarPattern(%q, %q) = %v, want %v", tt.pattern, tt.path, result, tt.expected)
			}
		})
	}
}

func TestFilterContext_IsIgnoredFunction_ChainedCall(t *testing.T) {
	code := `package test; func f() { obj.Method1().Method2("test") }`

	cfg := &config.UnqueryvetSettings{
		IgnoredFunctions: []string{"Method2"},
		IgnoredFiles:     []string{},
		AllowedPatterns:  []string{},
	}

	fc, err := NewFilterContext(cfg)
	if err != nil {
		t.Fatalf("Failed to create FilterContext: %v", err)
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", code, 0)
	if err != nil {
		t.Fatalf("Failed to parse code: %v", err)
	}

	var result bool
	ast.Inspect(f, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			// Only check the outer call (Method2)
			if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
				if sel.Sel.Name == "Method2" {
					result = fc.IsIgnoredFunction(call)
					return false
				}
			}
		}
		return true
	})

	if !result {
		t.Error("IsIgnoredFunction() should match chained call Method2")
	}
}
