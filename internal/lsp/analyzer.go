package lsp

import (
	"go/ast"
	"go/parser"
	"go/token"
	"regexp"
	"strings"
	"unicode"

	"github.com/MirrexOne/unqueryvet/internal/analyzer"
	"github.com/MirrexOne/unqueryvet/internal/lsp/protocol"
)

// StructInfo contains information about a Go struct and its database columns.
type StructInfo struct {
	Name    string
	Columns []string // Database column names extracted from struct tags or field names
}

// Analyzer performs SQL analysis on document content.
type Analyzer struct {
	// selectStarPattern matches SELECT * patterns
	selectStarPattern *regexp.Regexp
	// aliasedWildcardPattern matches SELECT alias.* patterns
	aliasedWildcardPattern *regexp.Regexp
	// subqueryPattern matches subquery SELECT *
	subqueryPattern *regexp.Regexp
	// allowedPatterns lists patterns that should not trigger warnings
	allowedPatterns []*regexp.Regexp

	// Feature flags for additional analyzers
	n1Enabled   bool
	sqliEnabled bool
}

// AnalyzerConfig contains configuration options for the analyzer.
type AnalyzerConfig struct {
	CheckN1Queries     bool
	CheckSQLInjection  bool
	CheckSelectStar    bool
	CheckAliasWildcard bool
	CheckSubqueries    bool
}

// Issue represents a detected SQL issue.
type Issue struct {
	Line       int
	Column     int
	EndColumn  int
	Message    string
	Severity   protocol.DiagnosticSeverity
	Code       string
	Query      string
	Suggestion string
}

// NewAnalyzer creates a new SQL analyzer.
func NewAnalyzer() *Analyzer {
	return &Analyzer{
		selectStarPattern:      regexp.MustCompile(`(?i)SELECT\s+\*\s+FROM`),
		aliasedWildcardPattern: regexp.MustCompile(`(?i)SELECT\s+([a-zA-Z_][a-zA-Z0-9_]*)\.\*`),
		subqueryPattern:        regexp.MustCompile(`(?i)\(\s*SELECT\s+\*`),
		allowedPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)COUNT\s*\(\s*\*\s*\)`),
			regexp.MustCompile(`(?i)SELECT\s+\*\s+FROM\s+information_schema`),
			regexp.MustCompile(`(?i)SELECT\s+\*\s+FROM\s+pg_catalog`),
			regexp.MustCompile(`(?i)SELECT\s+\*\s+FROM\s+sys\.`),
		},
		n1Enabled:   false,
		sqliEnabled: false,
	}
}

// SetN1Detection enables or disables N+1 query detection.
func (a *Analyzer) SetN1Detection(enabled bool) {
	a.n1Enabled = enabled
}

// SetSQLInjectionDetection enables or disables SQL injection detection.
func (a *Analyzer) SetSQLInjectionDetection(enabled bool) {
	a.sqliEnabled = enabled
}

// Configure applies configuration to the analyzer.
func (a *Analyzer) Configure(cfg AnalyzerConfig) {
	a.n1Enabled = cfg.CheckN1Queries
	a.sqliEnabled = cfg.CheckSQLInjection
}

// Analyze analyzes a document and returns diagnostics.
func (a *Analyzer) Analyze(doc *Document) []protocol.Diagnostic {
	// Only analyze Go files
	if doc.LanguageID != "" && doc.LanguageID != "go" {
		return nil
	}
	if !strings.HasSuffix(doc.URI, ".go") {
		return nil
	}

	var diagnostics []protocol.Diagnostic

	// 1. SELECT * detection (regex-based, always works)
	selectStarIssues := a.findIssues(doc.Content)
	for _, issue := range selectStarIssues {
		diag := protocol.Diagnostic{
			Range: protocol.Range{
				Start: protocol.Position{
					Line:      issue.Line,
					Character: issue.Column,
				},
				End: protocol.Position{
					Line:      issue.Line,
					Character: issue.EndColumn,
				},
			},
			Severity: issue.Severity,
			Code:     issue.Code,
			Source:   "unqueryvet",
			Message:  issue.Message,
			CodeDescription: &protocol.CodeDescription{
				Href: "https://github.com/MirrexOne/unqueryvet#why-avoid-select-",
			},
		}
		diagnostics = append(diagnostics, diag)
	}

	// 2. AST-based detection (N+1 and SQLI)
	if a.n1Enabled || a.sqliEnabled {
		astDiags := a.analyzeAST(doc.Content)
		diagnostics = append(diagnostics, astDiags...)
	}

	return diagnostics
}

// analyzeAST parses Go code and runs AST-based analyzers.
func (a *Analyzer) analyzeAST(content string) []protocol.Diagnostic {
	var diagnostics []protocol.Diagnostic

	// Parse Go code to AST
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "source.go", content, parser.AllErrors)
	if err != nil {
		// Syntax error - skip AST-based analysis
		// This is normal during editing
		return nil
	}

	// N+1 query detection
	if a.n1Enabled {
		n1Violations := analyzer.DetectN1InAST(fset, file)
		for _, v := range n1Violations {
			pos := fset.Position(v.Pos)
			endPos := fset.Position(v.End)

			diag := protocol.Diagnostic{
				Range: protocol.Range{
					Start: protocol.Position{
						Line:      pos.Line - 1, // LSP is 0-indexed
						Character: pos.Column - 1,
					},
					End: protocol.Position{
						Line:      endPos.Line - 1,
						Character: endPos.Column - 1,
					},
				},
				Severity: n1SeverityToLSP(v.Severity),
				Code:     "n1_query",
				Source:   "unqueryvet",
				Message:  v.Message,
				CodeDescription: &protocol.CodeDescription{
					Href: "https://github.com/MirrexOne/unqueryvet#n1-query-detection",
				},
			}
			diagnostics = append(diagnostics, diag)
		}
	}

	// SQL injection detection
	if a.sqliEnabled {
		sqliViolations := analyzer.ScanFileAST(fset, file)
		for _, v := range sqliViolations {
			pos := fset.Position(v.Pos)
			endPos := fset.Position(v.End)

			diag := protocol.Diagnostic{
				Range: protocol.Range{
					Start: protocol.Position{
						Line:      pos.Line - 1,
						Character: pos.Column - 1,
					},
					End: protocol.Position{
						Line:      endPos.Line - 1,
						Character: endPos.Column - 1,
					},
				},
				Severity: sqliSeverityToLSP(v.Severity),
				Code:     "sql_injection",
				Source:   "unqueryvet",
				Message:  v.Message,
				CodeDescription: &protocol.CodeDescription{
					Href: "https://github.com/MirrexOne/unqueryvet#sql-injection-detection",
				},
			}
			diagnostics = append(diagnostics, diag)
		}
	}

	return diagnostics
}

// n1SeverityToLSP converts N+1 severity to LSP diagnostic severity.
func n1SeverityToLSP(severity analyzer.N1Severity) protocol.DiagnosticSeverity {
	switch severity {
	case analyzer.N1SeverityCritical:
		return protocol.DiagnosticSeverityError
	case analyzer.N1SeverityHigh:
		return protocol.DiagnosticSeverityError
	case analyzer.N1SeverityMedium:
		return protocol.DiagnosticSeverityWarning
	case analyzer.N1SeverityLow:
		return protocol.DiagnosticSeverityInformation
	default:
		return protocol.DiagnosticSeverityWarning
	}
}

// sqliSeverityToLSP converts SQL injection severity to LSP diagnostic severity.
func sqliSeverityToLSP(severity analyzer.SQLISeverity) protocol.DiagnosticSeverity {
	switch severity {
	case analyzer.SQLISeverityCritical:
		return protocol.DiagnosticSeverityError
	case analyzer.SQLISeverityHigh:
		return protocol.DiagnosticSeverityError
	case analyzer.SQLISeverityMedium:
		return protocol.DiagnosticSeverityWarning
	case analyzer.SQLISeverityLow:
		return protocol.DiagnosticSeverityInformation
	default:
		return protocol.DiagnosticSeverityWarning
	}
}

// findIssues scans content for SQL issues.
func (a *Analyzer) findIssues(content string) []Issue {
	var issues []Issue
	lines := strings.Split(content, "\n")

	for lineNum, line := range lines {
		// Skip comments
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") {
			continue
		}

		// Check for SELECT * patterns
		if a.selectStarPattern.MatchString(line) {
			if !a.isAllowed(line) {
				loc := a.selectStarPattern.FindStringIndex(line)
				if loc != nil {
					issues = append(issues, Issue{
						Line:       lineNum,
						Column:     loc[0],
						EndColumn:  loc[1],
						Message:    "avoid SELECT * - explicitly specify needed columns for better performance, maintainability and stability",
						Severity:   protocol.DiagnosticSeverityWarning,
						Code:       "select_star",
						Query:      line,
						Suggestion: "Replace SELECT * with specific column names like: SELECT id, name, email",
					})
				}
			}
		}

		// Check for aliased wildcard patterns (SELECT t.*)
		matches := a.aliasedWildcardPattern.FindAllStringSubmatchIndex(line, -1)
		for _, match := range matches {
			if len(match) >= 2 && !a.isAllowed(line) {
				issues = append(issues, Issue{
					Line:       lineNum,
					Column:     match[0],
					EndColumn:  match[1],
					Message:    "avoid SELECT alias.* - explicitly specify columns like alias.id, alias.name for better maintainability",
					Severity:   protocol.DiagnosticSeverityWarning,
					Code:       "aliased_wildcard",
					Query:      line,
					Suggestion: "Replace alias.* with specific columns prefixed with the alias",
				})
			}
		}

		// Check for subquery SELECT *
		if a.subqueryPattern.MatchString(line) {
			loc := a.subqueryPattern.FindStringIndex(line)
			if loc != nil {
				issues = append(issues, Issue{
					Line:       lineNum,
					Column:     loc[0],
					EndColumn:  loc[1],
					Message:    "avoid SELECT * in subquery - explicitly specify needed columns",
					Severity:   protocol.DiagnosticSeverityWarning,
					Code:       "subquery_select_star",
					Query:      line,
					Suggestion: "Specify only the columns needed in both the subquery and outer query",
				})
			}
		}

		// Check for SQL builder patterns
		if strings.Contains(line, `Select("*")`) || strings.Contains(line, `Select('*')`) {
			idx := strings.Index(line, `Select("*")`)
			if idx == -1 {
				idx = strings.Index(line, `Select('*')`)
			}
			if idx >= 0 {
				issues = append(issues, Issue{
					Line:       lineNum,
					Column:     idx,
					EndColumn:  idx + 11,
					Message:    "avoid SELECT * in SQL builder - explicitly specify columns to prevent unnecessary data transfer",
					Severity:   protocol.DiagnosticSeverityWarning,
					Code:       "sql_builder_select_star",
					Query:      line,
					Suggestion: `Replace Select("*") with Select("id", "name", "email")`,
				})
			}
		}

		// Check for Columns("*")
		if strings.Contains(line, `Columns("*")`) {
			idx := strings.Index(line, `Columns("*")`)
			if idx >= 0 {
				issues = append(issues, Issue{
					Line:       lineNum,
					Column:     idx,
					EndColumn:  idx + 12,
					Message:    "avoid Columns(\"*\") - explicitly specify column names",
					Severity:   protocol.DiagnosticSeverityWarning,
					Code:       "sql_builder_columns_star",
					Query:      line,
					Suggestion: `Replace Columns("*") with Columns("id", "name", "email")`,
				})
			}
		}
	}

	return issues
}

// isAllowed checks if a query matches allowed patterns.
func (a *Analyzer) isAllowed(query string) bool {
	for _, pattern := range a.allowedPatterns {
		if pattern.MatchString(query) {
			return true
		}
	}
	return false
}

// extractStructs parses Go code and extracts struct definitions with their database columns.
func (a *Analyzer) extractStructs(content string) map[string]*StructInfo {
	structs := make(map[string]*StructInfo)

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "source.go", content, parser.ParseComments)
	if err != nil {
		return structs
	}

	ast.Inspect(file, func(n ast.Node) bool {
		typeSpec, ok := n.(*ast.TypeSpec)
		if !ok {
			return true
		}

		structType, ok := typeSpec.Type.(*ast.StructType)
		if !ok {
			return true
		}

		structName := typeSpec.Name.Name
		var columns []string

		for _, field := range structType.Fields.List {
			if len(field.Names) == 0 {
				continue // embedded field
			}

			fieldName := field.Names[0].Name
			// Skip unexported fields
			if !unicode.IsUpper(rune(fieldName[0])) {
				continue
			}

			// Try to get column name from struct tag
			columnName := a.getColumnFromTag(field.Tag, fieldName)
			if columnName != "" && columnName != "-" {
				columns = append(columns, columnName)
			}
		}

		if len(columns) > 0 {
			structs[structName] = &StructInfo{
				Name:    structName,
				Columns: columns,
			}
			// Also store by lowercase for easier lookup
			structs[strings.ToLower(structName)] = structs[structName]
		}

		return true
	})

	return structs
}

// getColumnFromTag extracts the database column name from a struct field tag.
// It checks for db, gorm, json, bun, sqlx tags in that order.
func (a *Analyzer) getColumnFromTag(tag *ast.BasicLit, fieldName string) string {
	if tag == nil {
		return toSnakeCase(fieldName)
	}

	tagValue := strings.Trim(tag.Value, "`")

	// Check common database tags in order of preference
	tagNames := []string{"db", "gorm", "bun", "sqlx", "json"}
	for _, tagName := range tagNames {
		if value := getTagValue(tagValue, tagName); value != "" {
			// Handle gorm's special syntax like "column:name"
			if tagName == "gorm" {
				if strings.Contains(value, "column:") {
					parts := strings.Split(value, ";")
					for _, part := range parts {
						if strings.HasPrefix(part, "column:") {
							return strings.TrimPrefix(part, "column:")
						}
					}
				}
				continue
			}
			// For other tags, take the first part before comma
			parts := strings.Split(value, ",")
			if parts[0] != "" && parts[0] != "-" {
				return parts[0]
			}
		}
	}

	return toSnakeCase(fieldName)
}

// getTagValue extracts a specific tag value from a struct tag string.
func getTagValue(tag, key string) string {
	// Find the key in the tag
	keyStart := strings.Index(tag, key+`:"`)
	if keyStart == -1 {
		return ""
	}

	valueStart := keyStart + len(key) + 2 // +2 for `:"
	valueEnd := strings.Index(tag[valueStart:], `"`)
	if valueEnd == -1 {
		return ""
	}

	return tag[valueStart : valueStart+valueEnd]
}

// toSnakeCase converts a CamelCase string to snake_case.
func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				result.WriteRune('_')
			}
			result.WriteRune(unicode.ToLower(r))
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// tableToStructName converts a SQL table name to a Go struct name.
// Examples: users -> User, user_profiles -> UserProfile
func tableToStructName(tableName string) string {
	// Remove schema prefix if present (e.g., "public.users" -> "users")
	if idx := strings.LastIndex(tableName, "."); idx != -1 {
		tableName = tableName[idx+1:]
	}

	// Convert snake_case to PascalCase and singularize
	parts := strings.Split(tableName, "_")
	var result strings.Builder
	for _, part := range parts {
		if len(part) > 0 {
			// Capitalize first letter
			result.WriteString(strings.ToUpper(part[:1]))
			if len(part) > 1 {
				result.WriteString(part[1:])
			}
		}
	}

	name := result.String()
	// Simple singularization: remove trailing 's' if present
	if len(name) > 1 && strings.HasSuffix(name, "s") && !strings.HasSuffix(name, "ss") {
		name = name[:len(name)-1]
	}

	return name
}

// extractTableName extracts the table name from a SELECT query.
func extractTableName(query string) string {
	// Pattern to match: SELECT ... FROM table_name
	pattern := regexp.MustCompile(`(?i)FROM\s+([a-zA-Z_][a-zA-Z0-9_]*(?:\.[a-zA-Z_][a-zA-Z0-9_]*)?)`)
	matches := pattern.FindStringSubmatch(query)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

// getColumnsForTable finds the struct corresponding to a table and returns its columns.
func (a *Analyzer) getColumnsForTable(structs map[string]*StructInfo, tableName string) []string {
	if tableName == "" {
		return nil
	}

	// Try direct struct name lookup
	structName := tableToStructName(tableName)
	if info, ok := structs[structName]; ok {
		return info.Columns
	}

	// Try lowercase lookup
	if info, ok := structs[strings.ToLower(structName)]; ok {
		return info.Columns
	}

	// Try the table name directly (for cases like "User" table with "User" struct)
	if info, ok := structs[tableName]; ok {
		return info.Columns
	}

	return nil
}

// GetHover returns hover information for a position.
func (a *Analyzer) GetHover(doc *Document, pos protocol.Position) *protocol.Hover {
	lines := strings.Split(doc.Content, "\n")
	if pos.Line >= len(lines) {
		return nil
	}

	line := lines[pos.Line]

	// Check if hovering over SELECT *
	if a.selectStarPattern.MatchString(line) {
		loc := a.selectStarPattern.FindStringIndex(line)
		if loc != nil && pos.Character >= loc[0] && pos.Character <= loc[1] {
			return &protocol.Hover{
				Contents: protocol.MarkupContent{
					Kind: protocol.MarkupKindMarkdown,
					Value: `### SELECT * Detected

**Why avoid SELECT \*?**

1. **Performance**: Selecting unnecessary columns wastes network bandwidth and memory
2. **Maintainability**: Schema changes can break your application unexpectedly
3. **Security**: May expose sensitive data that shouldn't be returned
4. **API Stability**: Adding new columns can break clients that depend on column order

**Recommendation**: Replace with explicit column selection:
` + "```sql\n" + `SELECT id, name, email FROM users
` + "```" + `

[Learn more](https://github.com/MirrexOne/unqueryvet#why-avoid-select-)`,
				},
				Range: &protocol.Range{
					Start: protocol.Position{Line: pos.Line, Character: loc[0]},
					End:   protocol.Position{Line: pos.Line, Character: loc[1]},
				},
			}
		}
	}

	return nil
}

// GetCodeActions returns code actions for diagnostics.
func (a *Analyzer) GetCodeActions(doc *Document, params protocol.CodeActionParams) []protocol.CodeAction {
	var actions []protocol.CodeAction

	// Extract structs from the document for column suggestions
	structs := a.extractStructs(doc.Content)

	for _, diag := range params.Context.Diagnostics {
		if diag.Source != "unqueryvet" {
			continue
		}

		// Get the line content
		lines := strings.Split(doc.Content, "\n")
		if diag.Range.Start.Line >= len(lines) {
			continue
		}
		line := lines[diag.Range.Start.Line]

		// Generate quick fix based on the issue type
		switch diag.Code {
		case "select_star":
			// Extract table name and find corresponding struct columns
			tableName := extractTableName(line)
			columns := a.getColumnsForTable(structs, tableName)

			var columnList string
			var title string
			if len(columns) > 0 {
				columnList = strings.Join(columns, ", ")
				title = "Replace SELECT * with columns from struct"
			} else {
				columnList = "id, name, email /* TODO: specify columns */"
				title = "Replace SELECT * with explicit columns"
			}

			newLine := a.selectStarPattern.ReplaceAllString(line, "SELECT "+columnList+" FROM")
			action := protocol.CodeAction{
				Title:       title,
				Kind:        protocol.CodeActionKindQuickFix,
				Diagnostics: []protocol.Diagnostic{diag},
				IsPreferred: true,
				Edit: &protocol.WorkspaceEdit{
					Changes: map[string][]protocol.TextEdit{
						doc.URI: {
							{
								Range: protocol.Range{
									Start: protocol.Position{Line: diag.Range.Start.Line, Character: 0},
									End:   protocol.Position{Line: diag.Range.Start.Line, Character: len(line)},
								},
								NewText: newLine,
							},
						},
					},
				},
			}
			actions = append(actions, action)

		case "sql_builder_select_star":
			// For SQL builders, try to find columns from any struct in the file
			var columns []string
			for _, info := range structs {
				if len(info.Columns) > 0 {
					columns = info.Columns
					break
				}
			}

			var replacement string
			var title string
			if len(columns) > 0 {
				// Format as Select("col1", "col2", "col3")
				quoted := make([]string, len(columns))
				for i, col := range columns {
					quoted[i] = `"` + col + `"`
				}
				replacement = "Select(" + strings.Join(quoted, ", ") + ")"
				title = "Replace Select(\"*\") with columns from struct"
			} else {
				replacement = `Select("id", "name", "email" /* TODO: specify columns */)`
				title = "Replace Select(\"*\") with explicit columns"
			}

			newLine := strings.Replace(line, `Select("*")`, replacement, 1)
			newLine = strings.Replace(newLine, `Select('*')`, replacement, 1)
			action := protocol.CodeAction{
				Title:       title,
				Kind:        protocol.CodeActionKindQuickFix,
				Diagnostics: []protocol.Diagnostic{diag},
				IsPreferred: true,
				Edit: &protocol.WorkspaceEdit{
					Changes: map[string][]protocol.TextEdit{
						doc.URI: {
							{
								Range: protocol.Range{
									Start: protocol.Position{Line: diag.Range.Start.Line, Character: 0},
									End:   protocol.Position{Line: diag.Range.Start.Line, Character: len(line)},
								},
								NewText: newLine,
							},
						},
					},
				},
			}
			actions = append(actions, action)

		case "aliased_wildcard":
			// For aliased wildcard (SELECT t.*), extract alias and columns
			tableName := extractTableName(line)
			columns := a.getColumnsForTable(structs, tableName)

			// Extract the alias from the pattern
			aliasMatch := a.aliasedWildcardPattern.FindStringSubmatch(line)
			alias := ""
			if len(aliasMatch) >= 2 {
				alias = aliasMatch[1]
			}

			var columnList string
			var title string
			if len(columns) > 0 && alias != "" {
				// Prefix each column with alias
				prefixed := make([]string, len(columns))
				for i, col := range columns {
					prefixed[i] = alias + "." + col
				}
				columnList = strings.Join(prefixed, ", ")
				title = "Replace " + alias + ".* with columns from struct"
			} else if alias != "" {
				columnList = alias + ".id, " + alias + ".name, " + alias + ".email /* TODO: specify columns */"
				title = "Replace " + alias + ".* with explicit columns"
			} else {
				continue
			}

			newLine := a.aliasedWildcardPattern.ReplaceAllString(line, columnList)
			action := protocol.CodeAction{
				Title:       title,
				Kind:        protocol.CodeActionKindQuickFix,
				Diagnostics: []protocol.Diagnostic{diag},
				IsPreferred: true,
				Edit: &protocol.WorkspaceEdit{
					Changes: map[string][]protocol.TextEdit{
						doc.URI: {
							{
								Range: protocol.Range{
									Start: protocol.Position{Line: diag.Range.Start.Line, Character: 0},
									End:   protocol.Position{Line: diag.Range.Start.Line, Character: len(line)},
								},
								NewText: newLine,
							},
						},
					},
				},
			}
			actions = append(actions, action)
		}
	}

	return actions
}

// GetCompletions returns completion items for a position.
func (a *Analyzer) GetCompletions(doc *Document, pos protocol.Position) []protocol.CompletionItem {
	lines := strings.Split(doc.Content, "\n")
	if pos.Line >= len(lines) {
		return nil
	}

	line := lines[pos.Line]
	prefix := ""
	if pos.Character <= len(line) {
		prefix = line[:pos.Character]
	}

	var items []protocol.CompletionItem

	// If user just typed * after SELECT, suggest replacing it
	if strings.Contains(strings.ToUpper(prefix), "SELECT") && strings.HasSuffix(strings.TrimSpace(prefix), "*") {
		items = append(items, protocol.CompletionItem{
			Label:      "id, name, email",
			Kind:       protocol.CompletionItemKindSnippet,
			Detail:     "Replace SELECT * with common columns",
			InsertText: "id, name, email",
			Documentation: protocol.MarkupContent{
				Kind:  protocol.MarkupKindMarkdown,
				Value: "Replace `SELECT *` with explicit column selection for better performance and maintainability.",
			},
		})

		items = append(items, protocol.CompletionItem{
			Label:      "id",
			Kind:       protocol.CompletionItemKindField,
			Detail:     "Primary key column",
			InsertText: "id",
		})

		items = append(items, protocol.CompletionItem{
			Label:      "created_at, updated_at",
			Kind:       protocol.CompletionItemKindSnippet,
			Detail:     "Timestamp columns",
			InsertText: "created_at, updated_at",
		})
	}

	// Common SQL column suggestions
	commonColumns := []string{"id", "name", "email", "created_at", "updated_at", "status", "type", "user_id"}
	for _, col := range commonColumns {
		items = append(items, protocol.CompletionItem{
			Label:      col,
			Kind:       protocol.CompletionItemKindField,
			Detail:     "Column name",
			InsertText: col,
		})
	}

	return items
}
