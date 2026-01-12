package lsp

import (
	"strings"

	"github.com/MirrexOne/unqueryvet/internal/dsl"
	"github.com/MirrexOne/unqueryvet/internal/lsp/protocol"
)

// DSLSupport provides LSP support for DSL configuration files.
type DSLSupport struct{}

// NewDSLSupport creates a new DSL support instance.
func NewDSLSupport() *DSLSupport {
	return &DSLSupport{}
}

// GetDSLCompletions returns completions for .unqueryvet.yaml files.
func (d *DSLSupport) GetDSLCompletions(doc *Document, pos protocol.Position) []protocol.CompletionItem {
	// Only for YAML config files
	if !strings.HasSuffix(doc.URI, ".unqueryvet.yaml") &&
		!strings.HasSuffix(doc.URI, ".unqueryvet.yml") &&
		!strings.HasSuffix(doc.URI, "unqueryvet.yaml") &&
		!strings.HasSuffix(doc.URI, "unqueryvet.yml") {
		return nil
	}

	lines := strings.Split(doc.Content, "\n")
	if pos.Line >= len(lines) {
		return nil
	}

	line := lines[pos.Line]
	trimmed := strings.TrimSpace(line)

	var items []protocol.CompletionItem

	// Detect context based on line content
	switch {
	case strings.HasPrefix(trimmed, "when:"):
		// Completions for condition expressions
		items = append(items, d.getConditionCompletions()...)

	case strings.HasPrefix(trimmed, "pattern:"):
		// Completions for patterns
		items = append(items, d.getPatternCompletions()...)

	case strings.HasPrefix(trimmed, "severity:"):
		items = append(items, d.getSeverityCompletions()...)

	case strings.HasPrefix(trimmed, "action:"):
		items = append(items, d.getActionCompletions()...)

	case strings.Contains(line, "rules:"):
		items = append(items, d.getBuiltinRuleCompletions()...)

	case trimmed == "" || strings.HasPrefix(trimmed, "-"):
		// Top-level or list context
		items = append(items, d.getTopLevelCompletions()...)
	}

	return items
}

// GetDSLHover returns hover information for DSL config files.
func (d *DSLSupport) GetDSLHover(doc *Document, pos protocol.Position) *protocol.Hover {
	if !strings.HasSuffix(doc.URI, ".unqueryvet.yaml") &&
		!strings.HasSuffix(doc.URI, ".unqueryvet.yml") {
		return nil
	}

	lines := strings.Split(doc.Content, "\n")
	if pos.Line >= len(lines) {
		return nil
	}

	line := lines[pos.Line]
	word := d.getWordAtPosition(line, pos.Character)

	// Check for built-in variables
	if desc, ok := dsl.BuiltinVariableDescriptions()[word]; ok {
		return &protocol.Hover{
			Contents: protocol.MarkupContent{
				Kind:  protocol.MarkupKindMarkdown,
				Value: "### Variable: `" + word + "`\n\n" + desc,
			},
		}
	}

	// Check for built-in functions
	if desc, ok := dsl.BuiltinFunctionDescriptions()[word]; ok {
		return &protocol.Hover{
			Contents: protocol.MarkupContent{
				Kind:  protocol.MarkupKindMarkdown,
				Value: "### Function: `" + word + "`\n\n" + desc,
			},
		}
	}

	// Check for metavariables
	if strings.HasPrefix(word, "$") {
		if desc, ok := dsl.MetavariableDescriptions()[word]; ok {
			return &protocol.Hover{
				Contents: protocol.MarkupContent{
					Kind:  protocol.MarkupKindMarkdown,
					Value: "### Metavariable: `" + word + "`\n\n" + desc,
				},
			}
		}
	}

	// Check for built-in rules
	if desc, ok := dsl.BuiltinRuleDescriptions()[word]; ok {
		return &protocol.Hover{
			Contents: protocol.MarkupContent{
				Kind:  protocol.MarkupKindMarkdown,
				Value: "### Rule: `" + word + "`\n\n" + desc,
			},
		}
	}

	// Check for keywords
	keywordDocs := map[string]string{
		"rules":        "Map of built-in rule IDs to their severity (error, warning, info, ignore)",
		"ignore":       "List of file glob patterns to ignore from analysis",
		"allow":        "List of SQL patterns to whitelist (won't trigger warnings)",
		"custom-rules": "List of user-defined custom rules with patterns and conditions",
		"pattern":      "SQL or code pattern to match. Supports metavariables like $TABLE, $VAR",
		"patterns":     "Multiple patterns for a single rule",
		"when":         "Condition expression evaluated with expr-lang",
		"message":      "Diagnostic message shown when the rule triggers",
		"severity":     "Severity level: error, warning, info, or ignore",
		"action":       "Action when pattern matches: report, allow, or ignore",
		"fix":          "Suggested fix message for the user",
	}

	if desc, ok := keywordDocs[word]; ok {
		return &protocol.Hover{
			Contents: protocol.MarkupContent{
				Kind:  protocol.MarkupKindMarkdown,
				Value: "### `" + word + "`\n\n" + desc,
			},
		}
	}

	return nil
}

// getConditionCompletions returns completions for "when:" conditions.
func (d *DSLSupport) getConditionCompletions() []protocol.CompletionItem {
	var items []protocol.CompletionItem

	// Variables
	for name, desc := range dsl.BuiltinVariableDescriptions() {
		items = append(items, protocol.CompletionItem{
			Label:      name,
			Kind:       protocol.CompletionItemKindVariable,
			Detail:     desc,
			InsertText: name,
		})
	}

	// Functions
	for name, desc := range dsl.BuiltinFunctionDescriptions() {
		items = append(items, protocol.CompletionItem{
			Label:      name,
			Kind:       protocol.CompletionItemKindFunction,
			Detail:     desc,
			InsertText: name + "(",
		})
	}

	// Common expressions
	commonExprs := []struct {
		label  string
		insert string
		detail string
	}{
		{"in_loop check", "in_loop", "True when code is inside a loop"},
		{"file regex match", `file =~ "_test.go$"`, "Match test files"},
		{"not test file", `!matches(file, "_test.go$")`, "Exclude test files"},
		{"system table check", "!isSystemTable(table)", "Exclude system tables"},
		{"temp table check", "isTempTable(table)", "Match temporary tables"},
		{"has WHERE clause", "has_where", "True when query has WHERE"},
	}

	for _, expr := range commonExprs {
		items = append(items, protocol.CompletionItem{
			Label:      expr.label,
			Kind:       protocol.CompletionItemKindSnippet,
			Detail:     expr.detail,
			InsertText: expr.insert,
		})
	}

	return items
}

// getPatternCompletions returns completions for "pattern:" values.
func (d *DSLSupport) getPatternCompletions() []protocol.CompletionItem {
	var items []protocol.CompletionItem

	// Metavariables
	for name, desc := range dsl.MetavariableDescriptions() {
		items = append(items, protocol.CompletionItem{
			Label:      name,
			Kind:       protocol.CompletionItemKindVariable,
			Detail:     desc,
			InsertText: name,
		})
	}

	// Common patterns
	patterns := []struct {
		label  string
		insert string
		detail string
	}{
		{"SELECT * FROM table", "SELECT * FROM $TABLE", "Match SELECT * queries"},
		{"DELETE without WHERE", "DELETE FROM $TABLE", "Match DELETE queries"},
		{"UPDATE without WHERE", "UPDATE $TABLE SET $COLS", "Match UPDATE queries"},
		{"DB.Query call", "$DB.Query($QUERY)", "Match database Query calls"},
		{"N+1 in loop", "$DB.Query($QUERY)", "Match queries (combine with in_loop condition)"},
	}

	for _, p := range patterns {
		items = append(items, protocol.CompletionItem{
			Label:      p.label,
			Kind:       protocol.CompletionItemKindSnippet,
			Detail:     p.detail,
			InsertText: p.insert,
		})
	}

	return items
}

// getSeverityCompletions returns completions for severity values.
func (d *DSLSupport) getSeverityCompletions() []protocol.CompletionItem {
	return []protocol.CompletionItem{
		{Label: "error", Kind: protocol.CompletionItemKindEnumMember, Detail: "Report as error"},
		{Label: "warning", Kind: protocol.CompletionItemKindEnumMember, Detail: "Report as warning (default)"},
		{Label: "info", Kind: protocol.CompletionItemKindEnumMember, Detail: "Report as information"},
		{Label: "ignore", Kind: protocol.CompletionItemKindEnumMember, Detail: "Ignore this rule"},
	}
}

// getActionCompletions returns completions for action values.
func (d *DSLSupport) getActionCompletions() []protocol.CompletionItem {
	return []protocol.CompletionItem{
		{Label: "report", Kind: protocol.CompletionItemKindEnumMember, Detail: "Report as violation (default)"},
		{Label: "allow", Kind: protocol.CompletionItemKindEnumMember, Detail: "Allow/whitelist this pattern"},
		{Label: "ignore", Kind: protocol.CompletionItemKindEnumMember, Detail: "Silently ignore"},
	}
}

// getBuiltinRuleCompletions returns completions for built-in rule IDs.
func (d *DSLSupport) getBuiltinRuleCompletions() []protocol.CompletionItem {
	var items []protocol.CompletionItem

	for id, desc := range dsl.BuiltinRuleDescriptions() {
		items = append(items, protocol.CompletionItem{
			Label:      id,
			Kind:       protocol.CompletionItemKindEnumMember,
			Detail:     desc,
			InsertText: id + ": warning",
		})
	}

	return items
}

// getTopLevelCompletions returns completions for top-level keys.
func (d *DSLSupport) getTopLevelCompletions() []protocol.CompletionItem {
	return []protocol.CompletionItem{
		{
			Label:            "rules",
			Kind:             protocol.CompletionItemKindProperty,
			Detail:           "Configure built-in rule severities",
			InsertText:       "rules:\n  select-star: warning\n  n1-queries: warning",
			InsertTextFormat: protocol.InsertTextFormatSnippet,
		},
		{
			Label:            "ignore",
			Kind:             protocol.CompletionItemKindProperty,
			Detail:           "File patterns to ignore",
			InsertText:       "ignore:\n  - \"*_test.go\"\n  - \"testdata/**\"",
			InsertTextFormat: protocol.InsertTextFormatSnippet,
		},
		{
			Label:            "allow",
			Kind:             protocol.CompletionItemKindProperty,
			Detail:           "SQL patterns to allow",
			InsertText:       "allow:\n  - \"COUNT(*)\"",
			InsertTextFormat: protocol.InsertTextFormatSnippet,
		},
		{
			Label:            "custom-rules",
			Kind:             protocol.CompletionItemKindProperty,
			Detail:           "Define custom analysis rules",
			InsertText:       "custom-rules:\n  - id: my-rule\n    pattern: SELECT * FROM $TABLE\n    message: \"Avoid SELECT *\"\n    severity: warning",
			InsertTextFormat: protocol.InsertTextFormatSnippet,
		},
		{
			Label:            "custom rule (full)",
			Kind:             protocol.CompletionItemKindSnippet,
			Detail:           "Full custom rule template",
			InsertText:       "- id: ${1:rule-id}\n  pattern: ${2:SELECT * FROM \\$TABLE}\n  when: ${3:!matches(file, \"_test.go$\")}\n  message: \"${4:Description of the issue}\"\n  severity: ${5|error,warning,info|}\n  fix: \"${6:Suggested fix}\"",
			InsertTextFormat: protocol.InsertTextFormatSnippet,
		},
	}
}

// getWordAtPosition extracts the word at the given position.
func (d *DSLSupport) getWordAtPosition(line string, char int) string {
	if char > len(line) {
		char = len(line)
	}

	// Find word boundaries
	start := char
	for start > 0 && isWordChar(line[start-1]) {
		start--
	}

	end := char
	for end < len(line) && isWordChar(line[end]) {
		end++
	}

	if start >= end {
		return ""
	}

	word := line[start:end]

	// Check for $ prefix (metavariables)
	if start > 0 && line[start-1] == '$' {
		word = "$" + word
	}

	return word
}

// isWordChar returns true if c is a valid word character.
func isWordChar(c byte) bool {
	return (c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') ||
		c == '_' || c == '-'
}
