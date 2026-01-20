// Package combined contains test cases demonstrating all default checks together.
package combined

import (
	"database/sql"
	"fmt"
)

// =============================================================================
// SELECT * Detection
// =============================================================================

// BadSelectStar demonstrates SELECT * usage
func BadSelectStar(db *sql.DB) {
	// Should warn: avoid SELECT *
	db.Query("SELECT * FROM users")
}

// GoodExplicitColumns demonstrates proper column selection
func GoodExplicitColumns(db *sql.DB) {
	// OK: explicit columns
	db.Query("SELECT id, name, email FROM users")
}

// =============================================================================
// N+1 Query Detection
// =============================================================================

// BadN1RangeLoop demonstrates N+1 in range loop
func BadN1RangeLoop(db *sql.DB, ids []int) {
	// Should warn: N+1 query detected
	for _, id := range ids {
		db.Query("SELECT name FROM users WHERE id = ?", id)
	}
}

// BadN1ForLoop demonstrates N+1 in for loop
func BadN1ForLoop(db *sql.DB, count int) {
	// Should warn: N+1 query detected
	for i := 0; i < count; i++ {
		db.QueryRow("SELECT name FROM users WHERE id = ?", i)
	}
}

// GoodBatchQuery demonstrates proper batch querying
func GoodBatchQuery(db *sql.DB, ids []int) {
	// OK: single query with IN clause
	db.Query("SELECT name FROM users WHERE id IN (?)", ids)
}

// =============================================================================
// SQL Injection Detection
// =============================================================================

// BadSQLInjectionSprintf demonstrates SQL injection via Sprintf
func BadSQLInjectionSprintf(db *sql.DB, userInput string) {
	// Should warn: potential SQL injection
	query := fmt.Sprintf("SELECT name FROM users WHERE id = '%s'", userInput)
	db.Query(query)
}

// BadSQLInjectionConcat demonstrates SQL injection via concatenation
func BadSQLInjectionConcat(db *sql.DB, userInput string) {
	// Should warn: potential SQL injection
	db.Query("SELECT name FROM users WHERE id = '" + userInput + "'")
}

// GoodParameterizedQuery demonstrates proper parameterized queries
func GoodParameterizedQuery(db *sql.DB, userInput string) {
	// OK: parameterized query
	db.Query("SELECT name FROM users WHERE id = ?", userInput)
}

// =============================================================================
// Combined Bad Patterns - Multiple issues in one function
// =============================================================================

// TerribleCode demonstrates multiple anti-patterns at once
func TerribleCode(db *sql.DB, userIDs []string) {
	for _, id := range userIDs {
		// N+1 query + SELECT * + SQL injection - triple threat!
		query := fmt.Sprintf("SELECT * FROM users WHERE id = '%s'", id)
		db.Query(query)
	}
}

// =============================================================================
// All Good Patterns
// =============================================================================

// GoodCode demonstrates all best practices
func GoodCode(db *sql.DB, userIDs []int) {
	// Single parameterized query with explicit columns
	db.Query("SELECT id, name, email FROM users WHERE id IN (?)", userIDs)
}
