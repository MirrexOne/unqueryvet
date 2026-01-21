package lsp

import (
	"testing"

	"github.com/MirrexOne/unqueryvet/internal/lsp/protocol"
	"github.com/MirrexOne/unqueryvet/pkg/config"
)

func TestAnalyzer_Analyze(t *testing.T) {
	analyzer := NewAnalyzer()

	tests := []struct {
		name      string
		content   string
		uri       string
		wantCount int
		wantCodes []string
	}{
		{
			name:      "simple SELECT *",
			content:   `query := "SELECT * FROM users"`,
			uri:       "file:///test.go",
			wantCount: 1,
			wantCodes: []string{"select_star"},
		},
		{
			name:      "no issues",
			content:   `query := "SELECT id, name FROM users"`,
			uri:       "file:///test.go",
			wantCount: 0,
			wantCodes: nil,
		},
		{
			name:      "COUNT(*) is allowed",
			content:   `query := "SELECT COUNT(*) FROM users"`,
			uri:       "file:///test.go",
			wantCount: 0,
			wantCodes: nil,
		},
		{
			name:      "aliased wildcard",
			content:   `query := "SELECT u.* FROM users u"`,
			uri:       "file:///test.go",
			wantCount: 1,
			wantCodes: []string{"aliased_wildcard"},
		},
		{
			name:      "subquery SELECT *",
			content:   `query := "SELECT id FROM (SELECT * FROM users)"`,
			uri:       "file:///test.go",
			wantCount: 2, // Both SELECT * FROM and (SELECT * patterns match
			wantCodes: []string{"select_star", "subquery_select_star"},
		},
		{
			name:      "SQL builder Select(*)",
			content:   `query := squirrel.Select("*").From("users")`,
			uri:       "file:///test.go",
			wantCount: 1,
			wantCodes: []string{"sql_builder_select_star"},
		},
		{
			name:      "information_schema is allowed",
			content:   `query := "SELECT * FROM information_schema.tables"`,
			uri:       "file:///test.go",
			wantCount: 0,
			wantCodes: nil,
		},
		{
			name:      "multiple issues",
			content:   "query1 := \"SELECT * FROM users\"\nquery2 := \"SELECT * FROM orders\"",
			uri:       "file:///test.go",
			wantCount: 2,
			wantCodes: []string{"select_star", "select_star"},
		},
		{
			name:      "comment line is skipped",
			content:   "// SELECT * FROM users",
			uri:       "file:///test.go",
			wantCount: 0,
			wantCodes: nil,
		},
		{
			name:      "non-go file is skipped",
			content:   `query := "SELECT * FROM users"`,
			uri:       "file:///test.txt",
			wantCount: 0,
			wantCodes: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &Document{
				URI:        tt.uri,
				LanguageID: "go",
				Content:    tt.content,
			}

			diagnostics := analyzer.Analyze(doc)

			if len(diagnostics) != tt.wantCount {
				t.Errorf("got %d diagnostics, want %d", len(diagnostics), tt.wantCount)
			}

			for i, diag := range diagnostics {
				if i < len(tt.wantCodes) {
					if diag.Code != tt.wantCodes[i] {
						t.Errorf("diagnostic[%d] code = %v, want %v", i, diag.Code, tt.wantCodes[i])
					}
				}
				if diag.Source != "unqueryvet" {
					t.Errorf("diagnostic[%d] source = %v, want unqueryvet", i, diag.Source)
				}
			}
		})
	}
}

func TestAnalyzer_GetHover(t *testing.T) {
	analyzer := NewAnalyzer()

	tests := []struct {
		name      string
		content   string
		line      int
		char      int
		wantHover bool
	}{
		{
			name:      "hover over SELECT *",
			content:   `query := "SELECT * FROM users"`,
			line:      0,
			char:      12,
			wantHover: true,
		},
		{
			name:      "hover outside SELECT *",
			content:   `query := "SELECT * FROM users"`,
			line:      0,
			char:      0,
			wantHover: false,
		},
		{
			name:      "hover on clean query",
			content:   `query := "SELECT id FROM users"`,
			line:      0,
			char:      12,
			wantHover: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &Document{
				URI:     "file:///test.go",
				Content: tt.content,
			}

			hover := analyzer.GetHover(doc, protocol.Position{
				Line:      tt.line,
				Character: tt.char,
			})

			if tt.wantHover && hover == nil {
				t.Error("expected hover, got nil")
			}
			if !tt.wantHover && hover != nil {
				t.Errorf("expected no hover, got %v", hover)
			}
		})
	}
}

func TestAnalyzer_GetCodeActions(t *testing.T) {
	analyzer := NewAnalyzer()

	doc := &Document{
		URI:     "file:///test.go",
		Content: `query := "SELECT * FROM users"`,
	}

	diagnostics := analyzer.Analyze(doc)
	if len(diagnostics) == 0 {
		t.Fatal("expected at least one diagnostic")
	}

	params := protocol.CodeActionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: doc.URI},
		Range:        diagnostics[0].Range,
		Context: protocol.CodeActionContext{
			Diagnostics: diagnostics,
		},
	}

	actions := analyzer.GetCodeActions(doc, params)
	if len(actions) == 0 {
		t.Error("expected at least one code action")
	}

	// Check that the action is a quick fix
	if actions[0].Kind != protocol.CodeActionKindQuickFix {
		t.Errorf("expected quickfix, got %v", actions[0].Kind)
	}

	// Check that the edit exists
	if actions[0].Edit == nil {
		t.Error("expected edit in code action")
	}
}

func TestAnalyzer_GetCompletions(t *testing.T) {
	analyzer := NewAnalyzer()

	doc := &Document{
		URI:     "file:///test.go",
		Content: `query := "SELECT *`,
	}

	items := analyzer.GetCompletions(doc, protocol.Position{
		Line:      0,
		Character: 18,
	})

	if len(items) == 0 {
		t.Error("expected at least one completion item")
	}

	// Check that common columns are included
	found := false
	for _, item := range items {
		if item.Label == "id" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'id' in completion items")
	}
}

func BenchmarkAnalyzer_Analyze(b *testing.B) {
	analyzer := NewAnalyzer()
	doc := &Document{
		URI: "file:///test.go",
		Content: `package main

import "database/sql"

func getUsers(db *sql.DB) error {
	query := "SELECT * FROM users"
	rows, err := db.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()
	return nil
}

func getOrders(db *sql.DB) error {
	query := "SELECT * FROM orders WHERE status = ?"
	rows, err := db.Query(query, "active")
	if err != nil {
		return err
	}
	defer rows.Close()
	return nil
}
`,
	}

	for b.Loop() {
		analyzer.Analyze(doc)
	}
}

func TestLSPIsRuleEnabled(t *testing.T) {
	tests := []struct {
		name     string
		rules    map[string]string
		ruleID   string
		expected bool
	}{
		{
			name:     "nil rules returns false",
			rules:    nil,
			ruleID:   "select-star",
			expected: false,
		},
		{
			name:     "rule not in map returns false",
			rules:    map[string]string{"other-rule": "warning"},
			ruleID:   "select-star",
			expected: false,
		},
		{
			name:     "rule with warning severity is enabled",
			rules:    map[string]string{"select-star": "warning"},
			ruleID:   "select-star",
			expected: true,
		},
		{
			name:     "rule with error severity is enabled",
			rules:    map[string]string{"sql-injection": "error"},
			ruleID:   "sql-injection",
			expected: true,
		},
		{
			name:     "rule with ignore severity is disabled",
			rules:    map[string]string{"n1-queries": "ignore"},
			ruleID:   "n1-queries",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRuleEnabled(tt.rules, tt.ruleID)
			if result != tt.expected {
				t.Errorf("isRuleEnabled(%v, %q) = %v, want %v", tt.rules, tt.ruleID, result, tt.expected)
			}
		})
	}
}

func TestNewAnalyzerWithConfigEnablesRules(t *testing.T) {
	// Import config package is needed
	cfg := config.DefaultSettings()

	analyzer := NewAnalyzerWithConfig(cfg)

	// With default config, both n1 and sqli should be enabled
	if !analyzer.n1Enabled {
		t.Error("n1 detection should be enabled with default config")
	}
	if !analyzer.sqliEnabled {
		t.Error("sqli detection should be enabled with default config")
	}
}

func TestNewAnalyzerWithConfigDisabledRules(t *testing.T) {
	cfg := config.DefaultSettings()
	// Disable n1-queries
	cfg.Rules["n1-queries"] = "ignore"

	analyzer := NewAnalyzerWithConfig(cfg)

	// n1 should be disabled
	if analyzer.n1Enabled {
		t.Error("n1 detection should be disabled when rule is ignore")
	}
	// sqli should still be enabled
	if !analyzer.sqliEnabled {
		t.Error("sqli detection should still be enabled")
	}
}

func TestNewAnalyzerUsesDefaultConfig(t *testing.T) {
	// NewAnalyzer should use default config which enables all rules
	analyzer := NewAnalyzer()

	if !analyzer.n1Enabled {
		t.Error("n1 detection should be enabled by default")
	}
	if !analyzer.sqliEnabled {
		t.Error("sqli detection should be enabled by default")
	}
}
