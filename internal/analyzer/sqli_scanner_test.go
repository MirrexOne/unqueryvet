package analyzer

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestSQLInjection_SprintfInQuery(t *testing.T) {
	src := `
package main

import (
	"database/sql"
	"fmt"
)

func getUser(db *sql.DB, userID string) {
	query := fmt.Sprintf("SELECT * FROM users WHERE id = '%s'", userID)
	db.Query(query)
}
`
	scanner := NewSQLInjectionScanner()
	violations := parseSQLI(t, src, scanner)

	// Should detect the sprintf pattern
	if len(violations) == 0 {
		t.Error("expected at least 1 violation for sprintf SQL injection")
	}
}

func TestSQLInjection_StringConcatenation(t *testing.T) {
	src := `
package main

import "database/sql"

func getUser(db *sql.DB, userID string) {
	db.Query("SELECT * FROM users WHERE id = '" + userID + "'")
}
`
	scanner := NewSQLInjectionScanner()
	violations := parseSQLI(t, src, scanner)

	if len(violations) == 0 {
		t.Error("expected violation for string concatenation SQL injection")
	}

	found := false
	for _, v := range violations {
		if v.VulnType == "concat" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'concat' violation type")
	}
}

func TestSQLInjection_SafeParameterized(t *testing.T) {
	src := `
package main

import "database/sql"

func getUser(db *sql.DB, userID string) {
	db.Query("SELECT * FROM users WHERE id = ?", userID)
}
`
	scanner := NewSQLInjectionScanner()
	violations := parseSQLI(t, src, scanner)

	// Parameterized queries are safe - should have no high severity issues
	highSeverity := 0
	for _, v := range violations {
		if v.Severity == SQLISeverityHigh || v.Severity == SQLISeverityCritical {
			highSeverity++
		}
	}
	if highSeverity > 0 {
		t.Errorf("expected no high severity violations for parameterized query, got %d", highSeverity)
	}
}

func TestSQLInjection_ExecWithSprintf(t *testing.T) {
	src := `
package main

import (
	"database/sql"
	"fmt"
)

func deleteUser(db *sql.DB, userID string) {
	query := fmt.Sprintf("DELETE FROM users WHERE id = '%s'", userID)
	db.Exec(query)
}
`
	scanner := NewSQLInjectionScanner()
	violations := parseSQLI(t, src, scanner)

	if len(violations) == 0 {
		t.Error("expected violation for sprintf in Exec")
	}
}

func TestSQLInjection_DirectSprintfInQuery(t *testing.T) {
	src := `
package main

import (
	"database/sql"
	"fmt"
)

func getUser(db *sql.DB, name string) {
	db.Query(fmt.Sprintf("SELECT * FROM users WHERE name = '%s'", name))
}
`
	scanner := NewSQLInjectionScanner()
	violations := parseSQLI(t, src, scanner)

	found := false
	for _, v := range violations {
		if (v.Severity == SQLISeverityHigh || v.Severity == SQLISeverityCritical) && v.VulnType == "sprintf" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected high severity sprintf violation")
	}
}

func TestSQLInjection_SQLxSelect(t *testing.T) {
	src := `
package main

import "fmt"

func getUsers(db interface{}, filter string) {
	db.Select(fmt.Sprintf("SELECT * FROM users WHERE %s", filter))
}
`
	scanner := NewSQLInjectionScanner()
	violations := parseSQLI(t, src, scanner)

	if len(violations) == 0 {
		t.Error("expected violation for sprintf in sqlx Select")
	}
}

func TestSQLInjection_GORMRaw(t *testing.T) {
	src := `
package main

import "fmt"

func getRawUsers(db interface{}, condition string) {
	query := fmt.Sprintf("SELECT * FROM users WHERE %s", condition)
	db.Raw(query)
}
`
	scanner := NewSQLInjectionScanner()
	violations := parseSQLI(t, src, scanner)

	if len(violations) == 0 {
		t.Error("expected violation for sprintf in GORM Raw")
	}
}

func TestSQLInjection_NamedExec(t *testing.T) {
	src := `
package main

import "fmt"

func updateUser(db interface{}, field, value string) {
	db.NamedExec(fmt.Sprintf("UPDATE users SET %s = :value", field), map[string]interface{}{"value": value})
}
`
	scanner := NewSQLInjectionScanner()
	violations := parseSQLI(t, src, scanner)

	if len(violations) == 0 {
		t.Error("expected violation for sprintf in NamedExec")
	}
}

func TestSQLInjection_SafeLiteral(t *testing.T) {
	src := `
package main

import "database/sql"

func getActiveUsers(db *sql.DB) {
	db.Query("SELECT * FROM users WHERE active = true")
}
`
	scanner := NewSQLInjectionScanner()
	violations := parseSQLI(t, src, scanner)

	// Static queries are safe
	highSeverity := 0
	for _, v := range violations {
		if v.Severity == SQLISeverityHigh || v.Severity == SQLISeverityCritical {
			highSeverity++
		}
	}
	if highSeverity > 0 {
		t.Errorf("expected no high severity violations for static query, got %d", highSeverity)
	}
}

func parseSQLI(t *testing.T, src string, scanner *SQLInjectionScanner) []SQLInjectionViolation {
	t.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("failed to parse source: %v", err)
	}

	var violations []SQLInjectionViolation
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.CallExpr:
			if v := scanner.checkCallExpr(node); v != nil {
				violations = append(violations, *v)
			}
		case *ast.BinaryExpr:
			if v := scanner.checkBinaryExpr(node); v != nil {
				violations = append(violations, *v)
			}
		}
		return true
	})

	return violations
}
