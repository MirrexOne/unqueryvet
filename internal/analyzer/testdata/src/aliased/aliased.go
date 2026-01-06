// Package aliased contains test cases for SELECT alias.* detection
package aliased

// Test SELECT alias.* patterns
func testAliasedWildcard() {
	// Should trigger warning - single alias
	q1 := "SELECT t.* FROM users t" // want "avoid SELECT"

	// Should trigger warning - multiple aliases
	q2 := "SELECT u.*, o.* FROM users u JOIN orders o ON u.id = o.user_id" // want "avoid SELECT"

	// Should trigger warning - table name as alias
	q3 := "SELECT users.* FROM users" // want "avoid SELECT"

	// Should NOT trigger warning - explicit columns with alias
	q4 := "SELECT t.id, t.name FROM users t"

	// Should NOT trigger warning - explicit columns
	q5 := "SELECT u.id, u.name, o.total FROM users u JOIN orders o ON u.id = o.user_id"

	_, _, _, _, _ = q1, q2, q3, q4, q5
}
