// Package a contains test cases for unqueryvet analyzer
package a

import (
	"database/sql"
	"fmt"
	"log"
	"os"
)

// Package-level constants with SELECT * (should trigger warnings)
const (
	// Should trigger warning - SELECT * in const
	BadQuery1 = "SELECT * FROM users" // want "avoid SELECT \\* - explicitly specify needed columns for better performance, maintainability and stability"

	// Should trigger warning - SELECT * with WHERE
	BadQuery2 = "SELECT * FROM orders WHERE status = 'active'" // want "avoid SELECT \\* - explicitly specify needed columns for better performance, maintainability and stability"

	// Good query - explicit columns
	GoodQuery1 = "SELECT id, name, email FROM users"

	// Good query - COUNT(*)
	GoodQuery2 = "SELECT COUNT(*) FROM users"

	// Good query - system tables
	GoodQuery3 = "SELECT * FROM information_schema.tables"
)

// Single package-level const declaration
const SingleBadQuery = "SELECT * FROM products" // want "avoid SELECT \\* - explicitly specify needed columns for better performance, maintainability and stability"

const SingleGoodQuery = "SELECT id, name FROM products"

// Multiline query constant
const MultilineQuery = `SELECT * FROM inventory WHERE quantity > 0` // want "avoid SELECT \\* - explicitly specify needed columns for better performance, maintainability and stability"

// Package-level variables (should also be checked)
var (
	VarBadQuery  = "SELECT * FROM categories" // want "avoid SELECT \\* - explicitly specify needed columns for better performance, maintainability and stability"
	VarGoodQuery = "SELECT id, name FROM categories"
	VarCount     = "SELECT COUNT(*) FROM categories"
)

// Non-SQL constants (should not trigger warnings)
const (
	NotSQL      = "This is not * a SQL query"
	AlsoNotSQL  = "Use asterisk * in documentation"
	JustText    = "Some random * text"
	NumberValue = 42
	BoolValue   = true
)

// Basic SELECT * detection in string literals
func basicSelectStar() {
	query := "SELECT * FROM users"           // want "avoid SELECT \\* - explicitly specify needed columns for better performance, maintainability and stability"
	anotherQuery := "select * from products" // want "avoid SELECT \\* - explicitly specify needed columns for better performance, maintainability and stability"
	_ = query
	_ = anotherQuery
}

// Acceptable patterns that should NOT trigger warnings
func acceptablePatterns() {
	// COUNT(*) is acceptable
	countQuery := "SELECT COUNT(*) FROM users"
	_ = countQuery

	// Information schema queries are acceptable (system tables)
	sysQuery := "SELECT * FROM information_schema.tables"
	_ = sysQuery

	// Catalog queries are acceptable
	pgQuery := "SELECT * FROM pg_catalog.pg_tables"
	_ = pgQuery
}

// SQL in function calls
func sqlInFunctionCalls() {
	db, _ := sql.Open("postgres", "")

	// This should trigger warning
	rows, _ := db.Query("SELECT * FROM orders") // want "avoid SELECT \\* - explicitly specify needed columns for better performance, maintainability and stability"
	_ = rows

	// This should also trigger warning
	stmt, _ := db.Prepare("SELECT * FROM customers") // want "avoid SELECT \\* - explicitly specify needed columns for better performance, maintainability and stability"
	_ = stmt
}

// Multiline SQL queries
func multilineSQL() {
	query := `SELECT * FROM users WHERE active = true` // want "avoid SELECT \\* - explicitly specify needed columns for better performance, maintainability and stability"
	_ = query
}

// Functions that will trigger warnings with default config
func defaultBehavior() {
	// These will trigger warnings with minimal default config
	fmt.Printf("SELECT * FROM debug_table")      // want "avoid SELECT \\* - explicitly specify needed columns for better performance, maintainability and stability"
	fmt.Sprintf("SELECT * FROM temp_%s", "data") // want "avoid SELECT \\* - explicitly specify needed columns for better performance, maintainability and stability"
	log.Printf("Executing: SELECT * FROM logs")  // want "avoid SELECT \\* - explicitly specify needed columns for better performance, maintainability and stability"

	// File operations
	_, _ = os.Open("file.sql")
}

// Test data for removed nolint functionality
func removedNolintDirectives() {
	// This now triggers - nolint support removed
	query := "SELECT * FROM temp_table" // want "avoid SELECT \\* - explicitly specify needed columns for better performance, maintainability and stability"
	_ = query

	// This also triggers - nolint support removed
	anotherQuery := "SELECT * FROM backup" // want "avoid SELECT \\* - explicitly specify needed columns for better performance, maintainability and stability"
	_ = anotherQuery
}
