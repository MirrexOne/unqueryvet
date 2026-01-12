package analyzer

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestN1Detector_RangeLoop(t *testing.T) {
	src := `
package main

import "database/sql"

func getUsers(db *sql.DB, ids []int) {
	for _, id := range ids {
		db.Query("SELECT * FROM users WHERE id = ?", id)
	}
}
`
	detector := NewN1Detector()
	violations := parseAndDetect(t, src, detector)

	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}

	if violations[0].LoopType != "range" {
		t.Errorf("expected loop type 'range', got '%s'", violations[0].LoopType)
	}

	if violations[0].QueryType != "Query" {
		t.Errorf("expected query type 'Query', got '%s'", violations[0].QueryType)
	}
}

func TestN1Detector_ForLoop(t *testing.T) {
	src := `
package main

import "database/sql"

func getUsers(db *sql.DB, count int) {
	for i := 0; i < count; i++ {
		db.QueryRow("SELECT name FROM users WHERE id = ?", i)
	}
}
`
	detector := NewN1Detector()
	violations := parseAndDetect(t, src, detector)

	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}

	if violations[0].LoopType != "for" {
		t.Errorf("expected loop type 'for', got '%s'", violations[0].LoopType)
	}

	if violations[0].QueryType != "QueryRow" {
		t.Errorf("expected query type 'QueryRow', got '%s'", violations[0].QueryType)
	}
}

func TestN1Detector_NoViolation_BatchQuery(t *testing.T) {
	src := `
package main

import "database/sql"

func getUsers(db *sql.DB, ids []int) {
	// This is correct - single batch query
	db.Query("SELECT * FROM users WHERE id IN (?)", ids)
}
`
	detector := NewN1Detector()
	violations := parseAndDetect(t, src, detector)

	if len(violations) != 0 {
		t.Errorf("expected 0 violations, got %d", len(violations))
	}
}

func TestN1Detector_NestedLoops(t *testing.T) {
	src := `
package main

import "database/sql"

func getNestedData(db *sql.DB, categories []int) {
	for _, catID := range categories {
		db.Query("SELECT * FROM products WHERE category_id = ?", catID)
		
		for i := 0; i < 10; i++ {
			db.Query("SELECT * FROM variants WHERE product_id = ?", i)
		}
	}
}
`
	detector := NewN1Detector()
	violations := parseAndDetect(t, src, detector)

	// Should detect both - one in outer loop, one in inner loop
	if len(violations) != 2 {
		t.Errorf("expected 2 violations, got %d", len(violations))
	}
}

func TestN1Detector_MultipleQueryMethods(t *testing.T) {
	src := `
package main

import "database/sql"

func processOrders(db *sql.DB, orderIDs []int) {
	for _, id := range orderIDs {
		db.Query("SELECT * FROM orders WHERE id = ?", id)
		db.Exec("UPDATE orders SET processed = true WHERE id = ?", id)
		db.QueryContext(nil, "SELECT * FROM order_items WHERE order_id = ?", id)
	}
}
`
	detector := NewN1Detector()
	violations := parseAndDetect(t, src, detector)

	if len(violations) != 3 {
		t.Errorf("expected 3 violations (Query, Exec, QueryContext), got %d", len(violations))
	}
}

func TestN1Detector_CommonDBVariables(t *testing.T) {
	src := `
package main

func processWithRepo(repo interface{}, ids []int) {
	for _, id := range ids {
		repo.Find(id)
	}
}

func processWithStore(store interface{}, ids []int) {
	for _, id := range ids {
		store.Get(id)
	}
}

func processWithTx(tx interface{}, ids []int) {
	for _, id := range ids {
		tx.Query("SELECT * FROM users WHERE id = ?", id)
	}
}
`
	detector := NewN1Detector()
	violations := parseAndDetect(t, src, detector)

	if len(violations) != 3 {
		t.Errorf("expected 3 violations, got %d", len(violations))
	}
}

func TestN1Detector_GORMStyle(t *testing.T) {
	src := `
package main

func getGORMUsers(db interface{}, ids []int) {
	for _, id := range ids {
		db.First("SELECT * FROM users WHERE id = ?", id)
		db.Where("id = ?", id).Find(nil)
	}
}
`
	detector := NewN1Detector()
	violations := parseAndDetect(t, src, detector)

	// First and Where are both detected as query methods
	if len(violations) != 2 {
		t.Errorf("expected 2 violations, got %d", len(violations))
	}
}

func TestN1Detector_SQLxStyle(t *testing.T) {
	src := `
package main

func getSQLxUsers(db interface{}, ids []int) {
	var users []interface{}
	for _, id := range ids {
		db.Select(&users, "SELECT * FROM users WHERE id = ?", id)
		db.Get(&users, "SELECT * FROM users WHERE id = ?", id)
	}
}
`
	detector := NewN1Detector()
	violations := parseAndDetect(t, src, detector)

	if len(violations) != 2 {
		t.Errorf("expected 2 violations (Select, Get), got %d", len(violations))
	}
}

func parseAndDetect(t *testing.T, src string, detector *N1Detector) []N1Violation {
	t.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("failed to parse source: %v", err)
	}

	var violations []N1Violation
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.ForStmt:
			if node.Body != nil {
				violations = append(violations, detector.checkBlockForQueries(nil, node.Body, "for")...)
			}
			return true
		case *ast.RangeStmt:
			if node.Body != nil {
				violations = append(violations, detector.checkBlockForQueries(nil, node.Body, "range")...)
			}
			return true
		}
		return true
	})

	return violations
}
