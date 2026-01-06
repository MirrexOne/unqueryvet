package analyzer

import (
	"go/token"
	"testing"
)

func TestNewSuggestedFixGenerator(t *testing.T) {
	fset := token.NewFileSet()
	sfg := NewSuggestedFixGenerator(fset)

	if sfg == nil {
		t.Fatal("NewSuggestedFixGenerator() returned nil")
	}

	if sfg.fset != fset {
		t.Error("NewSuggestedFixGenerator() did not set fset correctly")
	}
}

func TestSuggestedFixGenerator_GenerateFix(t *testing.T) {
	tests := []struct {
		name          string
		originalText  string
		violationType string
		expectedText  string
		expectNil     bool
	}{
		{
			name:          "select_star type",
			originalText:  `SELECT * FROM users`,
			violationType: "select_star",
			expectedText:  `SELECT id, /* TODO: specify columns */  FROM users`,
			expectNil:     false,
		},
		{
			name:          "aliased_wildcard type",
			originalText:  `SELECT t.* FROM users t`,
			violationType: "aliased_wildcard",
			expectedText:  `SELECT t.id, t./* TODO: specify columns */ FROM users t`,
			expectNil:     false,
		},
		{
			name:          "sql_builder_star type",
			originalText:  `"*"`,
			violationType: "sql_builder_star",
			expectedText:  `"id", /* TODO: specify columns */`,
			expectNil:     false,
		},
		{
			name:          "empty_select type",
			originalText:  `Select()`,
			violationType: "empty_select",
			expectedText:  `Select("id", /* TODO: specify columns */)`,
			expectNil:     false,
		},
		{
			name:          "unknown type - uses default",
			originalText:  `SELECT * FROM users`,
			violationType: "unknown",
			expectedText:  `SELECT id, /* TODO: specify columns */  FROM users`,
			expectNil:     false,
		},
		{
			name:          "no change needed - returns nil",
			originalText:  `SELECT id FROM users`,
			violationType: "select_star",
			expectedText:  "",
			expectNil:     true,
		},
		{
			name:          "aliased_wildcard no match",
			originalText:  `SELECT id FROM users`,
			violationType: "aliased_wildcard",
			expectedText:  "",
			expectNil:     true,
		},
	}

	fset := token.NewFileSet()
	sfg := NewSuggestedFixGenerator(fset)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fix := sfg.GenerateFix(1, 10, tt.originalText, tt.violationType)

			if tt.expectNil {
				if fix != nil {
					t.Errorf("GenerateFix() should return nil, got fix with message: %s", fix.Message)
				}
				return
			}

			if fix == nil {
				t.Fatal("GenerateFix() returned nil, expected a fix")
			}

			if len(fix.TextEdits) != 1 {
				t.Errorf("Expected 1 text edit, got %d", len(fix.TextEdits))
				return
			}

			newText := string(fix.TextEdits[0].NewText)
			if newText != tt.expectedText {
				t.Errorf("GenerateFix() text = %q, want %q", newText, tt.expectedText)
			}
		})
	}
}

func TestSuggestedFixGenerator_GenerateFix_Messages(t *testing.T) {
	tests := []struct {
		name            string
		violationType   string
		expectedMessage string
	}{
		{
			name:            "select_star message",
			violationType:   "select_star",
			expectedMessage: "Replace SELECT * with explicit columns",
		},
		{
			name:            "aliased_wildcard message",
			violationType:   "aliased_wildcard",
			expectedMessage: "Replace SELECT alias.* with explicit columns",
		},
		{
			name:            "sql_builder_star message",
			violationType:   "sql_builder_star",
			expectedMessage: `Replace "*" with explicit column names`,
		},
		{
			name:            "empty_select message",
			violationType:   "empty_select",
			expectedMessage: "Add column names to Select()",
		},
	}

	fset := token.NewFileSet()
	sfg := NewSuggestedFixGenerator(fset)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var originalText string
			switch tt.violationType {
			case "select_star":
				originalText = "SELECT * FROM users"
			case "aliased_wildcard":
				originalText = "SELECT t.* FROM users t"
			case "sql_builder_star":
				originalText = `"*"`
			case "empty_select":
				originalText = "Select()"
			}

			fix := sfg.GenerateFix(1, 10, originalText, tt.violationType)

			if fix == nil {
				t.Fatal("GenerateFix() returned nil")
			}

			if fix.Message != tt.expectedMessage {
				t.Errorf("GenerateFix() message = %q, want %q", fix.Message, tt.expectedMessage)
			}
		})
	}
}

func TestSuggestedFixGenerator_GenerateColumnPlaceholder(t *testing.T) {
	fset := token.NewFileSet()
	sfg := NewSuggestedFixGenerator(fset)

	expected := "id, /* TODO: specify columns */"
	result := sfg.GenerateColumnPlaceholder()

	if result != expected {
		t.Errorf("GenerateColumnPlaceholder() = %q, want %q", result, expected)
	}
}

func TestSuggestedFixGenerator_GenerateAliasedColumnPlaceholder(t *testing.T) {
	tests := []struct {
		alias    string
		expected string
	}{
		{
			alias:    "t",
			expected: "t.id, t./* TODO: specify columns */",
		},
		{
			alias:    "users",
			expected: "users.id, users./* TODO: specify columns */",
		},
		{
			alias:    "u",
			expected: "u.id, u./* TODO: specify columns */",
		},
	}

	fset := token.NewFileSet()
	sfg := NewSuggestedFixGenerator(fset)

	for _, tt := range tests {
		t.Run(tt.alias, func(t *testing.T) {
			result := sfg.GenerateAliasedColumnPlaceholder(tt.alias)
			if result != tt.expected {
				t.Errorf("GenerateAliasedColumnPlaceholder(%q) = %q, want %q", tt.alias, result, tt.expected)
			}
		})
	}
}

func TestCreateDiagnosticWithFix(t *testing.T) {
	tests := []struct {
		name          string
		message       string
		originalText  string
		violationType string
		expectFix     bool
	}{
		{
			name:          "with fix",
			message:       "SELECT * detected",
			originalText:  "SELECT * FROM users",
			violationType: "select_star",
			expectFix:     true,
		},
		{
			name:          "without original text",
			message:       "SELECT * detected",
			originalText:  "",
			violationType: "select_star",
			expectFix:     false,
		},
		{
			name:          "without fset - no fix",
			message:       "SELECT * detected",
			originalText:  "SELECT * FROM users",
			violationType: "select_star",
			expectFix:     false, // fset is nil, so no fix is generated
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			if tt.name == "without fset - no fix" {
				fset = nil
			}

			diag := CreateDiagnosticWithFix(1, 10, tt.message, tt.originalText, tt.violationType, fset)

			if diag.Message != tt.message {
				t.Errorf("CreateDiagnosticWithFix() message = %q, want %q", diag.Message, tt.message)
			}

			if diag.Pos != 1 {
				t.Errorf("CreateDiagnosticWithFix() pos = %d, want 1", diag.Pos)
			}

			if diag.End != 10 {
				t.Errorf("CreateDiagnosticWithFix() end = %d, want 10", diag.End)
			}

			hasFix := len(diag.SuggestedFixes) > 0
			if hasFix != tt.expectFix {
				t.Errorf("CreateDiagnosticWithFix() hasFix = %v, want %v", hasFix, tt.expectFix)
			}
		})
	}
}

func TestSuggestedFixGenerator_GenerateFix_Positions(t *testing.T) {
	fset := token.NewFileSet()
	sfg := NewSuggestedFixGenerator(fset)

	pos := token.Pos(100)
	end := token.Pos(200)

	fix := sfg.GenerateFix(pos, end, "SELECT * FROM users", "select_star")

	if fix == nil {
		t.Fatal("GenerateFix() returned nil")
	}

	if len(fix.TextEdits) != 1 {
		t.Fatalf("Expected 1 text edit, got %d", len(fix.TextEdits))
	}

	if fix.TextEdits[0].Pos != pos {
		t.Errorf("TextEdit.Pos = %d, want %d", fix.TextEdits[0].Pos, pos)
	}

	if fix.TextEdits[0].End != end {
		t.Errorf("TextEdit.End = %d, want %d", fix.TextEdits[0].End, end)
	}
}

func TestSelectStarFixPattern(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "SELECT * FROM users",
			expected: "SELECT id, /* TODO: specify columns */  FROM users",
		},
		{
			input:    "select * from users",
			expected: "select id, /* TODO: specify columns */  from users",
		},
		{
			input:    "SELECT  *  FROM users",
			expected: "SELECT  id, /* TODO: specify columns */   FROM users", // Extra space preserved
		},
		{
			input:    "SELECT * FROM",
			expected: "SELECT id, /* TODO: specify columns */  FROM",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := selectStarFixPattern.ReplaceAllString(tt.input, "${1}id, /* TODO: specify columns */ ${2}")
			if result != tt.expected {
				t.Errorf("Pattern replacement = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestAliasedStarFixPattern(t *testing.T) {
	tests := []struct {
		input   string
		matches bool
	}{
		{
			input:   "SELECT t.* FROM users t",
			matches: true,
		},
		{
			input:   "SELECT users.* FROM users",
			matches: true,
		},
		{
			input:   "SELECT  t . * FROM users",
			matches: true,
		},
		{
			input:   "SELECT * FROM users",
			matches: false,
		},
		{
			input:   "SELECT id FROM users",
			matches: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := aliasedStarFixPattern.MatchString(tt.input)
			if result != tt.matches {
				t.Errorf("Pattern match = %v, want %v", result, tt.matches)
			}
		})
	}
}
