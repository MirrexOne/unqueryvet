// Package selectstar contains test cases for SELECT * detection.
package selectstar

import (
	"database/sql"
)

// SELECT * in Query - should trigger warning
func getAllUsers(db *sql.DB) {
	db.Query("SELECT * FROM users") // want "avoid SELECT \\*"
}

// SELECT * in QueryRow - should trigger warning
func getUser(db *sql.DB, id int) {
	db.QueryRow("SELECT * FROM users WHERE id = ?", id) // want "avoid SELECT \\*"
}

// SELECT * with lowercase - should trigger warning
func getProducts(db *sql.DB) {
	db.Query("select * from products") // want "avoid SELECT \\*"
}

// SELECT * with mixed case - should trigger warning
func getOrders(db *sql.DB) {
	db.Query("Select * From orders") // want "avoid SELECT \\*"
}

// SELECT * with newlines - should trigger warning
func getCustomers(db *sql.DB) {
	db.Query(`
		SELECT *
		FROM customers
		WHERE active = true
	`) // want "avoid SELECT \\*"
}

// SELECT * in subquery - should trigger warning
func getUsersWithOrders(db *sql.DB) {
	db.Query("SELECT * FROM users WHERE id IN (SELECT user_id FROM orders)") // want "avoid SELECT \\*"
}

// Multiple SELECT * - should trigger multiple warnings
func getEverything(db *sql.DB) {
	db.Query("SELECT * FROM users")    // want "avoid SELECT \\*"
	db.Query("SELECT * FROM products") // want "avoid SELECT \\*"
}

// Acceptable: explicit columns
func getUserNames(db *sql.DB) {
	db.Query("SELECT id, name, email FROM users")
}

// Acceptable: COUNT(*)
func countUsers(db *sql.DB) {
	db.Query("SELECT COUNT(*) FROM users")
}

// Acceptable: system tables
func getTableInfo(db *sql.DB) {
	db.Query("SELECT * FROM information_schema.tables")
	db.Query("SELECT * FROM pg_catalog.pg_tables")
}
