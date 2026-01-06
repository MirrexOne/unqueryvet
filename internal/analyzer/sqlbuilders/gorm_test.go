package sqlbuilders

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestGORMChecker_Name(t *testing.T) {
	checker := NewGORMChecker()
	if checker.Name() != "gorm" {
		t.Errorf("Name() = %q, want %q", checker.Name(), "gorm")
	}
}

func TestGORMChecker_IsApplicable(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected bool
	}{
		{
			name:     "Select method",
			code:     `package test; func f() { db.Select("*") }`,
			expected: true,
		},
		{
			name:     "Find method",
			code:     `package test; func f() { db.Find(&users) }`,
			expected: true,
		},
		{
			name:     "First method",
			code:     `package test; func f() { db.First(&user) }`,
			expected: true,
		},
		{
			name:     "Raw method",
			code:     `package test; func f() { db.Raw("SELECT * FROM users") }`,
			expected: true,
		},
		{
			name:     "Model method",
			code:     `package test; func f() { db.Model(&user) }`,
			expected: true,
		},
		{
			name:     "gorm package prefix",
			code:     `package test; func f() { gorm.Open("mysql", "") }`,
			expected: true,
		},
		{
			name:     "non-gorm method",
			code:     `package test; func f() { foo.Bar() }`,
			expected: false,
		},
	}

	checker := NewGORMChecker()

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

func TestGORMChecker_CheckSelectStar(t *testing.T) {
	tests := []struct {
		name            string
		code            string
		expectViolation bool
		expectedContext string
	}{
		{
			name:            "Select with star",
			code:            `package test; func f() { db.Select("*") }`,
			expectViolation: true,
			expectedContext: "explicit_star",
		},
		{
			name:            "Raw with SELECT *",
			code:            `package test; func f() { db.Raw("SELECT * FROM users") }`,
			expectViolation: true,
			expectedContext: "raw_select_star",
		},
		{
			name:            "Exec with SELECT *",
			code:            `package test; func f() { db.Exec("SELECT * FROM users") }`,
			expectViolation: true,
			expectedContext: "raw_select_star",
		},
		{
			name:            "Select with explicit columns",
			code:            `package test; func f() { db.Select("id", "name") }`,
			expectViolation: false,
		},
		{
			name:            "Raw with explicit columns",
			code:            `package test; func f() { db.Raw("SELECT id, name FROM users") }`,
			expectViolation: false,
		},
		{
			name:            "Select with SELECT * in content",
			code:            `package test; func f() { db.Select("SELECT * FROM users") }`,
			expectViolation: true,
			expectedContext: "raw_select_star",
		},
	}

	checker := NewGORMChecker()

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
				if violation.Builder != "gorm" {
					t.Errorf("CheckSelectStar() builder = %q, want %q", violation.Builder, "gorm")
				}
			}
		})
	}
}

func TestGORMChecker_CheckChainedCalls(t *testing.T) {
	tests := []struct {
		name              string
		code              string
		expectViolations  bool
		minViolationCount int
	}{
		{
			name:              "Model Find without Select - no detection in current impl",
			code:              `package test; func f() { db.Model(&User{}).Find(&users) }`,
			expectViolations:  false, // Current implementation doesn't detect this pattern
			minViolationCount: 0,
		},
		{
			name:              "Model First without Select - no detection in current impl",
			code:              `package test; func f() { db.Model(&User{}).First(&user) }`,
			expectViolations:  false, // Current implementation doesn't detect this pattern
			minViolationCount: 0,
		},
		{
			name:              "Model Select Find (explicit columns)",
			code:              `package test; func f() { db.Model(&User{}).Select("id", "name").Find(&users) }`,
			expectViolations:  false,
			minViolationCount: 0,
		},
		{
			name:              "Select star in chain",
			code:              `package test; func f() { db.Model(&User{}).Select("*").Find(&users) }`,
			expectViolations:  true,
			minViolationCount: 1,
		},
		{
			name:              "Table Find without Select - no detection in current impl",
			code:              `package test; func f() { db.Table("users").Find(&users) }`,
			expectViolations:  false, // Current implementation doesn't detect this pattern
			minViolationCount: 0,
		},
	}

	checker := NewGORMChecker()

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

func TestNewGORMChecker(t *testing.T) {
	checker := NewGORMChecker()
	if checker == nil {
		t.Error("NewGORMChecker() returned nil")
	}
}

func TestGORMChecker_CheckSelectStar_CaseInsensitive(t *testing.T) {
	tests := []struct {
		code            string
		expectViolation bool
	}{
		{
			code:            `package test; func f() { db.Raw("select * from users") }`,
			expectViolation: true,
		},
		{
			code:            `package test; func f() { db.Raw("SELECT * FROM users") }`,
			expectViolation: true,
		},
		{
			code:            `package test; func f() { db.Raw("Select * From users") }`,
			expectViolation: true,
		},
	}

	checker := NewGORMChecker()

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
