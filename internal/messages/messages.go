// Package messages provides enhanced diagnostic messages with examples and documentation links.
package messages

import (
	"fmt"
	"strings"
)

const (
	// DocsBaseURL is the base URL for documentation
	DocsBaseURL = "https://github.com/MirrexOne/unqueryvet"
)

// MessageType represents the type of SELECT * violation.
type MessageType string

const (
	BasicSelectStar MessageType = "basic_select_star"
	AliasedWildcard MessageType = "aliased_wildcard"
	SQLBuilder      MessageType = "sql_builder"
	EmptySelect     MessageType = "empty_select"
	Subquery        MessageType = "subquery"
	Concatenation   MessageType = "concatenation"
	FormatString    MessageType = "format_string"
	StringBuilder   MessageType = "string_builder"
)

// DiagnosticMessage contains enhanced diagnostic information.
type DiagnosticMessage struct {
	Title       string
	Description string
	Example     string
	Suggestion  string
	Impact      string
	LearnMore   string
}

// GetEnhancedMessage returns an enhanced diagnostic message with context.
func GetEnhancedMessage(msgType MessageType, verbose bool) string {
	msg := getMessage(msgType)

	if !verbose {
		// Simple mode: just the title
		return msg.Title
	}

	// Verbose mode: full details
	var parts []string

	parts = append(parts, msg.Title)

	if msg.Description != "" {
		parts = append(parts, "\n"+msg.Description)
	}

	if msg.Example != "" {
		parts = append(parts, "\n\nExample fix:")
		parts = append(parts, msg.Example)
	}

	if msg.Suggestion != "" {
		parts = append(parts, "\n"+msg.Suggestion)
	}

	if msg.Impact != "" {
		parts = append(parts, "\n\n💡 Impact: "+msg.Impact)
	}

	if msg.LearnMore != "" {
		parts = append(parts, "\n📖 Learn more: "+msg.LearnMore)
	}

	return strings.Join(parts, "")
}

// getMessage returns the diagnostic message for a given type.
func getMessage(msgType MessageType) DiagnosticMessage {
	messages := map[MessageType]DiagnosticMessage{
		BasicSelectStar: {
			Title:       "avoid SELECT * - explicitly specify needed columns for better performance, maintainability and stability",
			Description: "Using SELECT * can lead to unexpected behavior when schema changes and wastes network bandwidth by transferring unnecessary data.",
			Example: `  - query := "SELECT * FROM users"
  + query := "SELECT id, name, email FROM users"`,
			Suggestion: "Specify only the columns you actually need in your application.",
			Impact:     "30-70% performance improvement for tables with many columns. Prevents breaking changes when schema is modified.",
			LearnMore:  DocsBaseURL + "#why-avoid-select-",
		},

		AliasedWildcard: {
			Title:       "avoid SELECT alias.* - explicitly specify columns like alias.id, alias.name for better maintainability",
			Description: "Aliased wildcards (e.g., t.*, u.*) make code harder to maintain and can break when table schemas change.",
			Example: `  - query := "SELECT t.* FROM users t"
  + query := "SELECT t.id, t.name, t.email FROM users t"`,
			Suggestion: "Replace alias.* with explicit column names prefixed with the alias.",
			Impact:     "Improves code clarity and prevents schema change issues in JOINs.",
			LearnMore:  DocsBaseURL + "#aliased-wildcards",
		},

		SQLBuilder: {
			Title:       "avoid SELECT * in SQL builder - explicitly specify columns to prevent unnecessary data transfer and schema change issues",
			Description: "SQL builders should use explicit column lists for type safety and performance.",
			Example: `  - query := squirrel.Select("*").From("users")
  + query := squirrel.Select("id", "name", "email").From("users")`,
			Suggestion: "Pass column names as separate arguments to Select().",
			Impact:     "Better type safety, easier refactoring, and improved performance.",
			LearnMore:  DocsBaseURL + "#sql-builders",
		},

		EmptySelect: {
			Title:       "SQL builder Select() without columns defaults to SELECT * - add specific columns",
			Description: "An empty Select() call implicitly selects all columns, which can cause issues.",
			Example: `  - query := squirrel.Select().From("users")
  + query := squirrel.Select("id", "name", "email").From("users")`,
			Suggestion: "Always specify columns in Select() or use Columns() method.",
			Impact:     "Prevents accidental SELECT * queries in production.",
			LearnMore:  DocsBaseURL + "#empty-select",
		},

		Subquery: {
			Title:       "avoid SELECT * in subquery - explicitly specify needed columns",
			Description: "SELECT * in subqueries can cause performance issues and makes queries harder to optimize.",
			Example: `  - query := "SELECT * FROM (SELECT * FROM users) AS u"
  + query := "SELECT u.id, u.name FROM (SELECT id, name FROM users) AS u"`,
			Suggestion: "Specify only the columns needed in both the subquery and outer query.",
			Impact:     "Significantly improves query performance and database optimizer efficiency.",
			LearnMore:  DocsBaseURL + "#subqueries",
		},

		Concatenation: {
			Title:       "avoid SELECT * in concatenated string - explicitly specify needed columns",
			Description: "String concatenation can hide SELECT * usage and make code harder to maintain.",
			Example: `  - query := "SELECT * " + "FROM users"
  + query := "SELECT id, name, email " + "FROM users"`,
			Suggestion: "Use explicit column names in concatenated query strings.",
			Impact:     "Improves code readability and query performance.",
			LearnMore:  DocsBaseURL + "#string-concatenation",
		},

		FormatString: {
			Title:       "avoid SELECT * in format string - explicitly specify needed columns",
			Description: "Using SELECT * in fmt.Sprintf or similar functions makes queries harder to review and maintain.",
			Example: `  - query := fmt.Sprintf("SELECT * FROM %s", table)
  + query := fmt.Sprintf("SELECT id, name, email FROM %s", table)`,
			Suggestion: "Always use explicit column lists, even in formatted strings.",
			Impact:     "Better code maintainability and performance.",
			LearnMore:  DocsBaseURL + "#format-strings",
		},

		StringBuilder: {
			Title:       "avoid SELECT * when building strings - explicitly specify needed columns",
			Description: "When using strings.Builder to construct queries, use explicit column names.",
			Example: `  - var b strings.Builder
  - b.WriteString("SELECT * FROM users")
  + var b strings.Builder
  + b.WriteString("SELECT id, name, email FROM users")`,
			Suggestion: "Build queries with explicit column lists for better performance.",
			Impact:     "Improves query performance and code maintainability.",
			LearnMore:  DocsBaseURL + "#string-builder",
		},
	}

	msg, exists := messages[msgType]
	if !exists {
		// Fallback message
		return DiagnosticMessage{
			Title: "avoid SELECT * - explicitly specify needed columns",
		}
	}

	return msg
}

// FormatDiagnostic formats a diagnostic message with file location.
func FormatDiagnostic(file string, line int, col int, msgType MessageType, verbose bool) string {
	location := fmt.Sprintf("%s:%d:%d", file, line, col)
	message := GetEnhancedMessage(msgType, verbose)

	return fmt.Sprintf("%s: %s", location, message)
}

// GetQuickFix returns a suggested quick fix for the given message type.
func GetQuickFix(msgType MessageType, originalQuery string) string {
	// This can be expanded to provide more intelligent suggestions
	// based on the actual query content
	switch msgType {
	case BasicSelectStar:
		return strings.Replace(originalQuery, "SELECT *", "SELECT id, /* TODO: specify columns */", 1)
	case AliasedWildcard:
		// More complex logic needed for aliased wildcards
		return originalQuery + " /* TODO: replace alias.* with explicit columns */"
	case SQLBuilder:
		return strings.Replace(originalQuery, `"*"`, `"id", /* TODO: specify columns */`, 1)
	case EmptySelect:
		return strings.Replace(originalQuery, "Select()", `Select("id", /* TODO: specify columns */)`, 1)
	default:
		return originalQuery
	}
}
