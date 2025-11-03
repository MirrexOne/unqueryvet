package examples

import (
	"database/sql"
)

// This file demonstrates Unqueryvet warnings and how to fix them
// NOTE: These are examples only - they don't actually execute

// ExampleBadCode shows patterns that Unqueryvet will warn about
func ExampleBadCode() {
	var db *sql.DB
	_ = db // db would be used in real code

	// BAD: Package-level style constants with SELECT *
	const (
		queryUsers   = "SELECT * FROM users"
		queryOrders  = "SELECT * FROM orders WHERE status = 'pending'"
		queryComplex = `
			SELECT * 
			FROM products
			WHERE price > 100
			ORDER BY created_at DESC
		`
	)
	_, _, _ = queryUsers, queryOrders, queryComplex

	// BAD: Single constant with SELECT *
	const querySingle = "SELECT * FROM inventory"
	_ = querySingle

	// BAD: Variables with SELECT *
	var (
		dynamicQuery = "SELECT * FROM logs WHERE date > NOW()"
		configQuery  = "SELECT * FROM settings"
	)
	_, _ = dynamicQuery, configQuery

	// BAD: Config-style constants
	const (
		getAllUsers    = "SELECT * FROM users"
		getAllProducts = "SELECT * FROM products"
		getAllOrders   = "SELECT * FROM orders"
	)
	_, _, _ = getAllUsers, getAllProducts, getAllOrders

	// BAD: Local constant with SELECT *
	const localBadQuery = "SELECT * FROM categories"
	_ = localBadQuery

	// BAD: Multiple local constants with SELECT *
	const (
		localQuery1 = "SELECT * FROM tags"
		localQuery2 = "SELECT * FROM comments WHERE active = true"
	)
	_, _ = localQuery1, localQuery2

	// BAD: Mixed declaration styles - all use SELECT *
	const constQuery = "SELECT * FROM users"
	var varQuery = "SELECT * FROM products"
	shortQuery := "SELECT * FROM orders"
	_, _, _ = constQuery, varQuery, shortQuery

	// BAD: Direct SELECT * in string literal
	query1 := "SELECT * FROM users WHERE active = true"
	_ = query1

	// BAD: SELECT * in function call argument
	// In real code: rows, err := db.Query("SELECT * FROM products")
	_ = "SELECT * FROM products"

	// BAD: Multiline SELECT *
	multilineQuery := `
		SELECT *
		FROM orders
		WHERE status = 'pending'
	`
	_ = multilineQuery

	// BAD: Real-world example - query constants for single use
	const (
		getUsersQuery    = "SELECT * FROM users WHERE role = 'admin'"
		getProductsQuery = "SELECT * FROM products WHERE in_stock = true"
		getOrdersQuery   = "SELECT * FROM orders WHERE created_at > NOW() - INTERVAL '7 days'"
	)
	_, _, _ = getUsersQuery, getProductsQuery, getOrdersQuery

	// BAD: SQL Builder patterns (pseudo-code)
	// query := squirrel.Select("*").From("users")
	// query := squirrel.Select().From("users")  // Empty Select defaults to SELECT *
}

// ExampleGoodCode shows patterns that Unqueryvet approves
func ExampleGoodCode() {
	var db *sql.DB
	_ = db // db would be used in real code

	// GOOD: Package-level style constants with explicit columns
	const (
		goodQueryPackage   = "SELECT id, name, email FROM users"
		countQueryPackage  = "SELECT COUNT(*) FROM users"
		schemaQueryPackage = "SELECT * FROM information_schema.tables"
	)
	_, _, _ = goodQueryPackage, countQueryPackage, schemaQueryPackage

	// GOOD: Not a SQL query
	const messagePackage = "Use * for multiplication"
	_ = messagePackage

	// GOOD: Config-style constants with explicit columns
	const (
		getUserBasics   = "SELECT id, username, email FROM users"
		getProductInfo  = "SELECT id, name, price, stock FROM products"
		getOrderDetails = "SELECT id, user_id, total, status FROM orders"
	)
	_, _, _ = getUserBasics, getProductInfo, getOrderDetails

	// GOOD: Local constant with explicit columns
	const localGoodQuery = "SELECT id, name FROM categories"
	_ = localGoodQuery

	// GOOD: Multiple local constants with explicit columns
	const (
		goodLocalQuery1 = "SELECT id, name FROM tags"
		goodLocalQuery2 = "SELECT id, text, author_id FROM comments WHERE active = true"
	)
	_, _ = goodLocalQuery1, goodLocalQuery2

	// GOOD: Local constant with COUNT(*)
	const localCountQuery = "SELECT COUNT(*) FROM sessions"
	_ = localCountQuery

	// GOOD: Explicit column selection in variable
	query1 := "SELECT id, name, email FROM users WHERE active = true"
	_ = query1

	// GOOD: Specific columns in function call argument
	// In real code: rows, err := db.Query("SELECT id, name, price FROM products")
	_ = "SELECT id, name, price FROM products"

	// GOOD: Multiline with explicit columns
	multilineQuery := `
		SELECT id, customer_id, total, status
		FROM orders
		WHERE status = 'pending'
	`
	_ = multilineQuery

	// GOOD: COUNT(*) is allowed by default
	countQuery := "SELECT COUNT(*) FROM users"
	_ = countQuery

	// GOOD: Information schema queries are allowed
	schemaQuery := "SELECT * FROM information_schema.tables WHERE table_name = 'users'"
	_ = schemaQuery

	// GOOD: Real-world example - properly defined queries
	const (
		getUsersQueryGood    = "SELECT id, username, email, role FROM users WHERE role = 'admin'"
		getProductsQueryGood = "SELECT id, name, price, stock FROM products WHERE in_stock = true"
		getOrdersQueryGood   = "SELECT id, user_id, total, created_at FROM orders WHERE created_at > NOW() - INTERVAL '7 days'"
	)
	_, _, _ = getUsersQueryGood, getProductsQueryGood, getOrdersQueryGood

	// GOOD: SQL Builder patterns (pseudo-code)
	// query := squirrel.Select("id", "name", "email").From("users")
	// query := squirrel.Select().Columns("id", "name").From("users")
}

// ExampleSuppression shows how to suppress Unqueryvet warnings
func ExampleSuppression() {
	var db *sql.DB
	_ = db // db would be used in real code

	// Using nolint directive to suppress warning for variable
	debugQuery := "SELECT * FROM debug_logs" //nolint:unqueryvet
	_ = debugQuery

	// Using nolint directive for constant
	const tempQuery = "SELECT * FROM temp_table" //nolint:unqueryvet
	_ = tempQuery

	// Using nolint directive for multiple constants
	const (
		debugConst = "SELECT * FROM backup"    //nolint:unqueryvet
		tempConst  = "SELECT * FROM temp_data" //nolint:unqueryvet // temporary for debugging
	)
	_, _ = debugConst, tempConst

	// Using nolint for package-level style variable
	var tempVar = "SELECT * FROM staging" //nolint:unqueryvet
	_ = tempVar
}
