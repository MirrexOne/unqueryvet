package analyzer

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/MirrexOne/unqueryvet/pkg/config"
)

func TestNewFormatStringAnalyzer(t *testing.T) {
	defaults := config.DefaultSettings()
	cfg := &defaults

	fsa := NewFormatStringAnalyzer(nil, cfg)

	if fsa == nil {
		t.Fatal("NewFormatStringAnalyzer() returned nil")
	}

	if fsa.cfg != cfg {
		t.Error("NewFormatStringAnalyzer() did not set config correctly")
	}
}

func TestFormatStringAnalyzer_AnalyzeFormatCall(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected bool
	}{
		{
			name:     "fmt.Sprintf with SELECT *",
			code:     `package test; import "fmt"; func f() { fmt.Sprintf("SELECT * FROM users") }`,
			expected: true,
		},
		{
			name:     "fmt.Printf with SELECT *",
			code:     `package test; import "fmt"; func f() { fmt.Printf("SELECT * FROM %s", "users") }`,
			expected: true,
		},
		{
			name:     "fmt.Errorf with SELECT *",
			code:     `package test; import "fmt"; func f() { fmt.Errorf("query: SELECT * FROM users") }`,
			expected: true,
		},
		{
			name:     "log.Printf with SELECT *",
			code:     `package test; import "log"; func f() { log.Printf("SELECT * FROM users") }`,
			expected: true,
		},
		{
			name:     "fmt.Sprintf with explicit columns",
			code:     `package test; import "fmt"; func f() { fmt.Sprintf("SELECT id, name FROM users") }`,
			expected: false,
		},
		{
			name:     "fmt.Sprintf with COUNT(*)",
			code:     `package test; import "fmt"; func f() { fmt.Sprintf("SELECT COUNT(*) FROM users") }`,
			expected: false,
		},
		{
			name:     "non-format function",
			code:     `package test; func f() { someFunc("SELECT * FROM users") }`,
			expected: false,
		},
		{
			name:     "fmt.Sprintf non-SQL string",
			code:     `package test; import "fmt"; func f() { fmt.Sprintf("hello %s", "world") }`,
			expected: false,
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
			fsa := NewFormatStringAnalyzer(nil, cfg)

			var result bool
			ast.Inspect(f, func(n ast.Node) bool {
				if call, ok := n.(*ast.CallExpr); ok {
					if fsa.AnalyzeFormatCall(call) {
						result = true
						return false
					}
				}
				return true
			})

			if result != tt.expected {
				t.Errorf("AnalyzeFormatCall() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestFormatStringAnalyzer_getFunctionName(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected string
	}{
		{
			name:     "fmt.Sprintf",
			code:     `package test; import "fmt"; func f() { fmt.Sprintf("test") }`,
			expected: "fmt.Sprintf",
		},
		{
			name:     "log.Printf",
			code:     `package test; import "log"; func f() { log.Printf("test") }`,
			expected: "log.Printf",
		},
		{
			name:     "direct Sprintf (dot import)",
			code:     `package test; func f() { Sprintf("test") }`,
			expected: "Sprintf",
		},
		{
			name:     "nested selector",
			code:     `package test; func f() { pkg.sub.Func("test") }`,
			expected: "pkg.sub.Func",
		},
		{
			name:     "receiver method",
			code:     `package test; func f() { logger.Infof("test") }`,
			expected: "logger.Infof",
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
			fsa := NewFormatStringAnalyzer(nil, cfg)

			var result string
			ast.Inspect(f, func(n ast.Node) bool {
				if call, ok := n.(*ast.CallExpr); ok {
					result = fsa.getFunctionName(call)
					return false
				}
				return true
			})

			if result != tt.expected {
				t.Errorf("getFunctionName() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestFormatStringAnalyzer_extractFormatString(t *testing.T) {
	tests := []struct {
		name        string
		code        string
		argIndex    int
		expectedStr string
		expectedOK  bool
	}{
		{
			name:        "direct string literal index 0",
			code:        `package test; func f() { fn("SELECT * FROM users") }`,
			argIndex:    0,
			expectedStr: "SELECT * FROM users",
			expectedOK:  true,
		},
		{
			name:        "direct string literal index 1",
			code:        `package test; func f() { fn(arg1, "SELECT * FROM users") }`,
			argIndex:    1,
			expectedStr: "SELECT * FROM users",
			expectedOK:  true,
		},
		{
			name:        "out of bounds index",
			code:        `package test; func f() { fn("test") }`,
			argIndex:    5,
			expectedStr: "",
			expectedOK:  false,
		},
		{
			name:        "negative index",
			code:        `package test; func f() { fn("test") }`,
			argIndex:    -1,
			expectedStr: "",
			expectedOK:  false,
		},
		{
			name:        "non-string argument",
			code:        `package test; func f() { fn(123) }`,
			argIndex:    0,
			expectedStr: "",
			expectedOK:  false,
		},
		{
			name:        "raw string literal",
			code:        "package test; func f() { fn(`SELECT * FROM users`) }",
			argIndex:    0,
			expectedStr: "SELECT * FROM users",
			expectedOK:  true,
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
			fsa := NewFormatStringAnalyzer(nil, cfg)

			var resultStr string
			var resultOK bool
			ast.Inspect(f, func(n ast.Node) bool {
				if call, ok := n.(*ast.CallExpr); ok {
					resultStr, resultOK = fsa.extractFormatString(call, tt.argIndex)
					return false
				}
				return true
			})

			if resultOK != tt.expectedOK {
				t.Errorf("extractFormatString() ok = %v, want %v", resultOK, tt.expectedOK)
			}

			if resultStr != tt.expectedStr {
				t.Errorf("extractFormatString() = %q, want %q", resultStr, tt.expectedStr)
			}
		})
	}
}

func TestCheckFormatFunction(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected bool
	}{
		{
			name:     "SELECT * in format function",
			code:     `package test; import "fmt"; func f() { fmt.Sprintf("SELECT * FROM users") }`,
			expected: true,
		},
		{
			name:     "explicit columns in format function",
			code:     `package test; import "fmt"; func f() { fmt.Sprintf("SELECT id FROM users") }`,
			expected: false,
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

			var result bool
			ast.Inspect(f, func(n ast.Node) bool {
				if call, ok := n.(*ast.CallExpr); ok {
					if CheckFormatFunction(nil, call, cfg) {
						result = true
						return false
					}
				}
				return true
			})

			if result != tt.expected {
				t.Errorf("CheckFormatFunction() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsFormatFunction(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected bool
	}{
		{
			name:     "fmt.Sprintf",
			code:     `package test; import "fmt"; func f() { fmt.Sprintf("test") }`,
			expected: true,
		},
		{
			name:     "fmt.Printf",
			code:     `package test; import "fmt"; func f() { fmt.Printf("test") }`,
			expected: true,
		},
		{
			name:     "fmt.Errorf",
			code:     `package test; import "fmt"; func f() { fmt.Errorf("test") }`,
			expected: true,
		},
		{
			name:     "log.Printf",
			code:     `package test; import "log"; func f() { log.Printf("test") }`,
			expected: true,
		},
		{
			name:     "log.Fatalf",
			code:     `package test; import "log"; func f() { log.Fatalf("test") }`,
			expected: true,
		},
		{
			name:     "unknown function",
			code:     `package test; func f() { someFunc("test") }`,
			expected: false,
		},
		{
			name:     "db.Query (not a format function)",
			code:     `package test; func f() { db.Query("test") }`,
			expected: false,
		},
		{
			name:     "direct Sprintf",
			code:     `package test; func f() { Sprintf("test") }`,
			expected: true,
		},
		{
			name:     "direct Printf",
			code:     `package test; func f() { Printf("test") }`,
			expected: true,
		},
		{
			name:     "logrus.Infof",
			code:     `package test; import "logrus"; func f() { logrus.Infof("test") }`,
			expected: true,
		},
		{
			name:     "logrus.Errorf",
			code:     `package test; import "logrus"; func f() { logrus.Errorf("test") }`,
			expected: true,
		},
	}

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
					result = IsFormatFunction(call)
					return false
				}
				return true
			})

			if result != tt.expected {
				t.Errorf("IsFormatFunction() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestFormatFunctions_Map(t *testing.T) {
	// Test that the formatFunctions map contains expected entries
	expectedFunctions := []struct {
		name     string
		argIndex int
	}{
		{"fmt.Sprintf", 0},
		{"fmt.Printf", 0},
		{"fmt.Fprintf", 1},
		{"fmt.Errorf", 0},
		{"log.Printf", 0},
		{"log.Fatalf", 0},
		{"Sprintf", 0},
		{"Printf", 0},
		{"logrus.Infof", 0},
		{"logrus.Errorf", 0},
	}

	for _, expected := range expectedFunctions {
		t.Run(expected.name, func(t *testing.T) {
			argIndex, ok := formatFunctions[expected.name]
			if !ok {
				t.Errorf("formatFunctions does not contain %q", expected.name)
				return
			}
			if argIndex != expected.argIndex {
				t.Errorf("formatFunctions[%q] = %d, want %d", expected.name, argIndex, expected.argIndex)
			}
		})
	}
}

func TestFormatStringAnalyzer_AnalyzeFormatCall_Fprintf(t *testing.T) {
	// fmt.Fprintf has format string at index 1 (first arg is io.Writer)
	code := `package test
import "fmt"
import "os"
func f() { fmt.Fprintf(os.Stdout, "SELECT * FROM users") }`

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", code, 0)
	if err != nil {
		t.Fatalf("Failed to parse code: %v", err)
	}

	defaults := config.DefaultSettings()
	cfg := &defaults
	fsa := NewFormatStringAnalyzer(nil, cfg)

	var result bool
	ast.Inspect(f, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
				if sel.Sel.Name == "Fprintf" {
					result = fsa.AnalyzeFormatCall(call)
					return false
				}
			}
		}
		return true
	})

	if !result {
		t.Error("AnalyzeFormatCall() should detect SELECT * in Fprintf")
	}
}

func TestFormatStringAnalyzer_AnalyzeFormatCall_EmptyArgs(t *testing.T) {
	code := `package test; import "fmt"; func f() { fmt.Sprintf() }`

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", code, 0)
	if err != nil {
		t.Fatalf("Failed to parse code: %v", err)
	}

	defaults := config.DefaultSettings()
	cfg := &defaults
	fsa := NewFormatStringAnalyzer(nil, cfg)

	var result bool
	ast.Inspect(f, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
				if sel.Sel.Name == "Sprintf" {
					result = fsa.AnalyzeFormatCall(call)
					return false
				}
			}
		}
		return true
	})

	if result {
		t.Error("AnalyzeFormatCall() should return false for empty args")
	}
}
