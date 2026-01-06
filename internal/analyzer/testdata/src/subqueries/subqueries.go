// Package subqueries contains test cases for SELECT * in subquery detection
package subqueries

// Test SELECT * in subqueries
func testSubqueries() {
	// Should trigger warning - SELECT * in subquery
	q1 := "SELECT id FROM (SELECT * FROM users)" // want "avoid SELECT"

	// Should trigger warning - SELECT * in IN clause
	q2 := "SELECT * FROM users WHERE id IN (SELECT * FROM orders)" // want "avoid SELECT"

	// Should trigger warning - SELECT * in EXISTS
	q3 := `SELECT u.name FROM users u
           WHERE EXISTS (SELECT * FROM orders WHERE user_id = u.id)` // want "avoid SELECT"

	// Should NOT trigger warning - explicit columns in subquery
	q4 := "SELECT id FROM (SELECT id, name FROM users)"

	// Should NOT trigger warning - explicit columns in IN
	q5 := "SELECT id FROM users WHERE id IN (SELECT user_id FROM orders)"

	_, _, _, _, _ = q1, q2, q3, q4, q5
}
