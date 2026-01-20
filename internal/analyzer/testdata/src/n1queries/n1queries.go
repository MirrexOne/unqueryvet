// Package n1queries contains test cases for N+1 query detection.
package n1queries

import (
	"database/sql"
)

// N+1 in range loop - should trigger warning
func fetchUsersInLoop(db *sql.DB, userIDs []int) {
	for _, id := range userIDs { // want "N\\+1 query detected"
		db.Query("SELECT name FROM users WHERE id = ?", id)
	}
}

// N+1 in for loop - should trigger warning
func fetchOrdersInLoop(db *sql.DB, count int) {
	for i := 0; i < count; i++ { // want "N\\+1 query detected"
		db.QueryRow("SELECT total FROM orders WHERE id = ?", i)
	}
}

// N+1 with Exec in loop - should trigger warning
func updateUsersInLoop(db *sql.DB, users []string) {
	for _, user := range users { // want "N\\+1 query detected"
		db.Exec("UPDATE users SET active = true WHERE name = ?", user)
	}
}

// N+1 with transaction in loop - should trigger warning
func insertInLoop(tx *sql.Tx, items []string) {
	for _, item := range items { // want "N\\+1 query detected"
		tx.Exec("INSERT INTO items (name) VALUES (?)", item)
	}
}

// Acceptable: single query outside loop
func fetchAllUsers(db *sql.DB) {
	db.Query("SELECT * FROM users") // This is fine for N+1 (but triggers SELECT *)
}

// Acceptable: batch query
func fetchUsersByIDs(db *sql.DB, ids []int) {
	db.Query("SELECT name FROM users WHERE id IN (?)", ids)
}
