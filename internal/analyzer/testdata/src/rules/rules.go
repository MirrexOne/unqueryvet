// Package rules contains test cases for default rules behavior
package rules

import (
	"database/sql"
)

// =============================================================================
// SELECT-STAR RULE TESTS
// =============================================================================

// Basic SELECT * should trigger warning
func selectStarBasic() {
	query := "SELECT * FROM users" // want "avoid SELECT \\* - explicitly specify needed columns for better performance, maintainability and stability"
	_ = query
}

// SELECT * with WHERE clause
func selectStarWithWhere() {
	query := "SELECT * FROM orders WHERE status = 'active'" // want "avoid SELECT \\* - explicitly specify needed columns for better performance, maintainability and stability"
	_ = query
}

// SELECT * in db.Query
func selectStarInQuery() {
	db, _ := sql.Open("postgres", "")
	rows, _ := db.Query("SELECT * FROM products") // want "avoid SELECT \\* - explicitly specify needed columns for better performance, maintainability and stability"
	_ = rows
}

// Case insensitive SELECT *
func selectStarCaseInsensitive() {
	query := "select * from users" // want "avoid SELECT \\* - explicitly specify needed columns for better performance, maintainability and stability"
	_ = query
}

// Multiline SELECT * (single line for testing)
func selectStarMultiline() {
	query := `SELECT * FROM users WHERE active = true` // want "avoid SELECT \\* - explicitly specify needed columns for better performance, maintainability and stability"
	_ = query
}

// =============================================================================
// ALLOWED PATTERNS (should NOT trigger warnings)
// =============================================================================

// COUNT(*) is allowed
func countStarAllowed() {
	query := "SELECT COUNT(*) FROM users" // OK
	_ = query
}

// MAX(*) is allowed
func maxStarAllowed() {
	query := "SELECT MAX(*) FROM orders" // OK
	_ = query
}

// MIN(*) is allowed
func minStarAllowed() {
	query := "SELECT MIN(*) FROM products" // OK
	_ = query
}

// information_schema queries are allowed
func informationSchemaAllowed() {
	query := "SELECT * FROM information_schema.tables" // OK
	_ = query
}

// pg_catalog queries are allowed
func pgCatalogAllowed() {
	query := "SELECT * FROM pg_catalog.pg_tables" // OK
	_ = query
}

// sys queries are allowed
func sysSchemaAllowed() {
	query := "SELECT * FROM sys.tables" // OK
	_ = query
}

// =============================================================================
// EXPLICIT COLUMNS (should NOT trigger warnings)
// =============================================================================

// Explicit columns are good
func explicitColumns() {
	query := "SELECT id, name, email FROM users" // OK
	_ = query
}

// Explicit columns with table alias
func explicitColumnsWithAlias() {
	query := "SELECT u.id, u.name FROM users u" // OK
	_ = query
}

// =============================================================================
// EDGE CASES
// =============================================================================

// Not a SQL query (no FROM keyword)
func notSQLQuery() {
	text := "SELECT * something random" // OK - no SQL keywords
	_ = text
}

// Asterisk in non-SQL context
func asteriskInText() {
	text := "Use * for wildcards" // OK - not SQL
	_ = text
}

// Empty query
func emptyQuery() {
	query := "" // OK
	_ = query
}
