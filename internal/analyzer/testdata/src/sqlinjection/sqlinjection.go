// Package sqlinjection contains test cases for SQL injection detection.
package sqlinjection

import (
	"database/sql"
	"fmt"
)

// SQL injection via fmt.Sprintf - should trigger warning
func getUserByIDSprintf(db *sql.DB, userID string) {
	query := fmt.Sprintf("SELECT * FROM users WHERE id = '%s'", userID) // want "potential SQL injection"
	db.Query(query)
}

// SQL injection via string concatenation - should trigger warning
func getUserByIDConcat(db *sql.DB, userID string) {
	db.Query("SELECT name FROM users WHERE id = '" + userID + "'") // want "potential SQL injection"
}

// SQL injection via Sprintf with multiple params - should trigger warning
func searchUsers(db *sql.DB, name, email string) {
	query := fmt.Sprintf("SELECT * FROM users WHERE name = '%s' AND email = '%s'", name, email) // want "potential SQL injection"
	db.Query(query)
}

// SQL injection in Exec - should trigger warning
func deleteUser(db *sql.DB, userID string) {
	query := fmt.Sprintf("DELETE FROM users WHERE id = '%s'", userID) // want "potential SQL injection"
	db.Exec(query)
}

// SQL injection via variable concatenation - should trigger warning
func getOrdersByUser(db *sql.DB, userID string) {
	baseQuery := "SELECT * FROM orders WHERE user_id = '"
	db.Query(baseQuery + userID + "'") // want "potential SQL injection"
}

// Acceptable: parameterized query
func getUserByIDSafe(db *sql.DB, userID string) {
	db.Query("SELECT name FROM users WHERE id = ?", userID)
}

// Acceptable: parameterized query with named params
func getUserByEmailSafe(db *sql.DB, email string) {
	db.Query("SELECT name FROM users WHERE email = $1", email)
}

// Acceptable: constant query
func getAllUsers(db *sql.DB) {
	db.Query("SELECT name, email FROM users")
}
