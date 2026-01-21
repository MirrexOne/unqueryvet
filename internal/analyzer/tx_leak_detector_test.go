package analyzer

import (
	"go/parser"
	"go/token"
	"testing"
)

// ============================================================================
// FALSE NEGATIVE TESTS (problems that SHOULD be detected)
// ============================================================================

func TestTxLeakDetector_NoCommitRollback(t *testing.T) {
	src := `
package main

import "database/sql"

func leaky(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec("INSERT INTO users (name) VALUES ('test')")
	return err
}
`
	violations := parseTxLeakAndDetect(t, src)

	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}

	if violations[0].Severity != TxLeakSeverityCritical {
		t.Errorf("expected severity critical, got %s", violations[0].Severity)
	}

	if violations[0].ViolationType != "no_commit_rollback" {
		t.Errorf("expected violation type 'no_commit_rollback', got '%s'", violations[0].ViolationType)
	}

	if violations[0].TxVarName != "tx" {
		t.Errorf("expected tx var name 'tx', got '%s'", violations[0].TxVarName)
	}
}

func TestTxLeakDetector_CommitNoRollback(t *testing.T) {
	src := `
package main

import "database/sql"

func noRollback(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec("INSERT INTO users (name) VALUES ('test')")
	if err != nil {
		return err // No rollback!
	}
	return tx.Commit()
}
`
	violations := parseTxLeakAndDetect(t, src)

	if len(violations) < 1 {
		t.Fatalf("expected at least 1 violation, got %d", len(violations))
	}

	found := false
	for _, v := range violations {
		if v.ViolationType == "no_rollback" {
			found = true
			if v.Severity != TxLeakSeverityHigh {
				t.Errorf("expected severity high, got %s", v.Severity)
			}
			break
		}
	}
	if !found {
		t.Error("expected 'no_rollback' violation")
	}
}

func TestTxLeakDetector_RollbackNoCommit(t *testing.T) {
	src := `
package main

import "database/sql"

func noCommit(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec("INSERT INTO users (name) VALUES ('test')")
	if err != nil {
		tx.Rollback()
		return err
	}
	return nil // No commit!
}
`
	violations := parseTxLeakAndDetect(t, src)

	found := false
	for _, v := range violations {
		if v.ViolationType == "no_commit" {
			found = true
			if v.Severity != TxLeakSeverityMedium {
				t.Errorf("expected severity medium, got %s", v.Severity)
			}
			break
		}
	}
	if !found {
		t.Error("expected 'no_commit' violation")
	}
}

func TestTxLeakDetector_Shadowing(t *testing.T) {
	src := `
package main

import "database/sql"

func shadowingBug(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if true {
		tx, err := db.Begin() // Shadows outer tx
		if err != nil {
			return err
		}
		defer tx.Rollback()
		_ = tx.Commit()
	}

	return tx.Commit()
}
`
	violations := parseTxLeakAndDetect(t, src)

	found := false
	for _, v := range violations {
		if v.ViolationType == "shadowed_transaction" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'shadowed_transaction' violation for variable shadowing")
	}
}

func TestTxLeakDetector_PanicWithoutDefer(t *testing.T) {
	src := `
package main

import "database/sql"

func panicWithoutDefer(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec("INSERT INTO users (name) VALUES ('test')")
	if err != nil {
		panic(err)
	}

	return tx.Commit()
}
`
	violations := parseTxLeakAndDetect(t, src)

	found := false
	for _, v := range violations {
		if v.ViolationType == "panic_without_defer" {
			found = true
			if v.Severity != TxLeakSeverityMedium {
				t.Errorf("expected severity medium, got %s", v.Severity)
			}
			break
		}
	}
	if !found {
		t.Error("expected 'panic_without_defer' violation")
	}
}

func TestTxLeakDetector_GoroutineCapture(t *testing.T) {
	src := `
package main

import "database/sql"

func goroutineCapture(db *sql.DB, ch chan error) {
	tx, err := db.Begin()
	if err != nil {
		ch <- err
		return
	}

	go func() {
		_, err := tx.Exec("INSERT INTO users (name) VALUES ('test')")
		if err != nil {
			tx.Rollback()
			ch <- err
			return
		}
		ch <- tx.Commit()
	}()
}
`
	violations := parseTxLeakAndDetect(t, src)

	found := false
	for _, v := range violations {
		if v.ViolationType == "goroutine_capture" {
			found = true
			if v.Severity != TxLeakSeverityHigh {
				t.Errorf("expected severity high, got %s", v.Severity)
			}
			break
		}
	}
	if !found {
		t.Error("expected 'goroutine_capture' violation")
	}
}

func TestTxLeakDetector_ConditionalCommit(t *testing.T) {
	src := `
package main

import "database/sql"

func conditionalCommit(db *sql.DB, shouldCommit bool) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec("INSERT INTO users (name) VALUES ('test')")
	if err != nil {
		tx.Rollback()
		return err
	}

	if shouldCommit {
		return tx.Commit()
	}

	return nil
}
`
	violations := parseTxLeakAndDetect(t, src)

	found := false
	for _, v := range violations {
		if v.ViolationType == "conditional_commit" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'conditional_commit' violation")
	}
}

func TestTxLeakDetector_EarlyReturn(t *testing.T) {
	src := `
package main

import "database/sql"

func earlyReturn(db *sql.DB, items []string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	if len(items) == 0 {
		return nil // Early return without commit
	}

	_, err = tx.Exec("INSERT INTO users (name) VALUES ('test')")
	if err != nil {
		return err
	}

	return tx.Commit()
}
`
	violations := parseTxLeakAndDetect(t, src)

	found := false
	for _, v := range violations {
		if v.ViolationType == "early_return" || v.ViolationType == "no_rollback" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'early_return' or 'no_rollback' violation")
	}
}

// ============================================================================
// FALSE POSITIVE TESTS (correct code that should NOT be flagged)
// ============================================================================

func TestTxLeakDetector_ProperDeferPattern(t *testing.T) {
	src := `
package main

import "database/sql"

func proper(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec("INSERT INTO users (name) VALUES ('test')")
	if err != nil {
		return err
	}
	return tx.Commit()
}
`
	violations := parseTxLeakAndDetect(t, src)

	// Should have no critical violations
	for _, v := range violations {
		if v.Severity == TxLeakSeverityCritical || v.Severity == TxLeakSeverityHigh {
			t.Errorf("unexpected violation for proper defer pattern: %s - %s", v.ViolationType, v.Message)
		}
	}
}

func TestTxLeakDetector_ReturnedTransaction(t *testing.T) {
	src := `
package main

import "database/sql"

func factory(db *sql.DB) (*sql.Tx, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	return tx, nil
}
`
	violations := parseTxLeakAndDetect(t, src)

	if len(violations) != 0 {
		t.Errorf("expected 0 violations for returned transaction, got %d", len(violations))
		for _, v := range violations {
			t.Logf("  violation: %s - %s", v.ViolationType, v.Message)
		}
	}
}

func TestTxLeakDetector_DeferClosure(t *testing.T) {
	src := `
package main

import "database/sql"

func closureDefer(db *sql.DB) (err error) {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	_, err = tx.Exec("INSERT INTO users (name) VALUES ('test')")
	if err != nil {
		return err
	}
	err = tx.Commit()
	return
}
`
	violations := parseTxLeakAndDetect(t, src)

	// Should have no critical/high violations
	for _, v := range violations {
		if v.Severity == TxLeakSeverityCritical {
			t.Errorf("unexpected critical violation for defer closure pattern: %s", v.Message)
		}
	}
}

func TestTxLeakDetector_PassedToFunctionWithDefer(t *testing.T) {
	src := `
package main

import "database/sql"

func passedWithDefer(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := processWithTx(tx); err != nil {
		return err
	}
	return tx.Commit()
}

func processWithTx(tx *sql.Tx) error {
	_, err := tx.Exec("INSERT INTO users (name) VALUES ('test')")
	return err
}
`
	violations := parseTxLeakAndDetect(t, src)

	// Should have no critical violations (defer handles the passed tx)
	for _, v := range violations {
		if v.Severity == TxLeakSeverityCritical {
			t.Errorf("unexpected critical violation when tx passed to function with defer: %s", v.Message)
		}
	}
}

func TestTxLeakDetector_GoroutineWithDefer(t *testing.T) {
	src := `
package main

import "database/sql"

func goroutineWithDefer(db *sql.DB, ch chan error) {
	tx, err := db.Begin()
	if err != nil {
		ch <- err
		return
	}
	defer tx.Rollback() // Has defer as safety net

	go func() {
		_, err := tx.Exec("INSERT INTO users (name) VALUES ('test')")
		ch <- err
	}()

	if err := <-ch; err != nil {
		return
	}
	tx.Commit()
}
`
	violations := parseTxLeakAndDetect(t, src)

	// Should not have goroutine_capture violation because defer exists
	for _, v := range violations {
		if v.ViolationType == "goroutine_capture" {
			t.Errorf("unexpected goroutine_capture violation when defer exists: %s", v.Message)
		}
	}
}

func TestTxLeakDetector_PanicWithDefer(t *testing.T) {
	src := `
package main

import "database/sql"

func panicWithDefer(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() // Will execute even on panic

	_, err = tx.Exec("INSERT INTO users (name) VALUES ('test')")
	if err != nil {
		panic(err)
	}

	return tx.Commit()
}
`
	violations := parseTxLeakAndDetect(t, src)

	// Should not have panic_without_defer violation because defer exists
	for _, v := range violations {
		if v.ViolationType == "panic_without_defer" {
			t.Errorf("unexpected panic_without_defer violation when defer exists: %s", v.Message)
		}
	}
}

func TestTxLeakDetector_BeginTx(t *testing.T) {
	src := `
package main

import (
	"context"
	"database/sql"
)

func beginTxLeaky(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	_, err = tx.Exec("INSERT INTO users (name) VALUES ('test')")
	return err
}
`
	violations := parseTxLeakAndDetect(t, src)

	if len(violations) < 1 {
		t.Fatalf("expected at least 1 violation for BeginTx, got %d", len(violations))
	}

	found := false
	for _, v := range violations {
		if v.Severity == TxLeakSeverityCritical {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected critical violation for BeginTx without commit/rollback")
	}
}

func TestTxLeakDetector_MultipleTransactions(t *testing.T) {
	src := `
package main

import "database/sql"

func multiTx(db *sql.DB) error {
	tx1, err := db.Begin()
	if err != nil {
		return err
	}
	_, _ = tx1.Exec("INSERT INTO table1 (id) VALUES (1)")
	// tx1 is leaky

	tx2, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx2.Rollback()
	_, _ = tx2.Exec("INSERT INTO table2 (id) VALUES (2)")
	return tx2.Commit()
}
`
	violations := parseTxLeakAndDetect(t, src)

	// Should have violation for tx1 only
	foundTx1 := false
	foundTx2Critical := false
	for _, v := range violations {
		if v.TxVarName == "tx1" && v.Severity == TxLeakSeverityCritical {
			foundTx1 = true
		}
		if v.TxVarName == "tx2" && v.Severity == TxLeakSeverityCritical {
			foundTx2Critical = true
		}
	}

	if !foundTx1 {
		t.Error("expected violation for tx1")
	}
	if foundTx2Critical {
		t.Error("unexpected critical violation for tx2 (properly handled)")
	}
}

func TestTxLeakDetector_SQLxBeginTxx(t *testing.T) {
	src := `
package main

func sqlxLeaky(db interface{}) error {
	tx, err := db.BeginTxx(nil, nil)
	if err != nil {
		return err
	}
	_, err = tx.Exec("INSERT INTO users (name) VALUES ('test')")
	return err
}
`
	violations := parseTxLeakAndDetect(t, src)

	if len(violations) < 1 {
		t.Fatalf("expected at least 1 violation for BeginTxx, got %d", len(violations))
	}
}

func TestTxLeakDetector_MustBegin(t *testing.T) {
	src := `
package main

func mustBeginLeaky(db interface{}) error {
	tx := db.MustBegin()
	_, err := tx.Exec("INSERT INTO users (name) VALUES ('test')")
	return err
}
`
	violations := parseTxLeakAndDetect(t, src)

	if len(violations) < 1 {
		t.Fatalf("expected at least 1 violation for MustBegin, got %d", len(violations))
	}
}

func TestTxLeakDetector_NoTransactions(t *testing.T) {
	src := `
package main

import "database/sql"

func noTx(db *sql.DB) error {
	_, err := db.Exec("INSERT INTO users (name) VALUES ('test')")
	return err
}
`
	violations := parseTxLeakAndDetect(t, src)

	if len(violations) != 0 {
		t.Errorf("expected 0 violations when no transactions, got %d", len(violations))
	}
}

func TestTxLeakDetector_CallbackPattern(t *testing.T) {
	src := `
package main

type DB interface {
	Transaction(func(tx interface{}) error) error
	RunInTransaction(func(tx interface{}) error) error
}

func callbackPattern(db DB) error {
	return db.Transaction(func(tx interface{}) error {
		return nil
	})
}

func runInTxPattern(db DB) error {
	return db.RunInTransaction(func(tx interface{}) error {
		return nil
	})
}
`
	violations := parseTxLeakAndDetect(t, src)

	if len(violations) != 0 {
		t.Errorf("expected 0 violations for callback patterns, got %d", len(violations))
		for _, v := range violations {
			t.Logf("  violation: %s - %s", v.ViolationType, v.Message)
		}
	}
}

// ============================================================================
// NEW SCENARIO TESTS (switch/case, select/case, fatal, loop, reassign, etc.)
// ============================================================================

func TestTxLeakDetector_CommitInSwitch(t *testing.T) {
	src := `
package main

import "database/sql"

func commitInSwitch(db *sql.DB, action string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec("INSERT INTO users (name) VALUES ('test')")
	if err != nil {
		tx.Rollback()
		return err
	}

	switch action {
	case "save":
		return tx.Commit()
	case "discard":
		tx.Rollback()
		return nil
	}
	// No default case - commit might not execute!
	return nil
}
`
	violations := parseTxLeakAndDetect(t, src)

	found := false
	for _, v := range violations {
		if v.ViolationType == "commit_in_switch" {
			found = true
			if v.Severity != TxLeakSeverityMedium {
				t.Errorf("expected severity medium, got %s", v.Severity)
			}
			break
		}
	}
	if !found {
		t.Error("expected 'commit_in_switch' violation")
	}
}

func TestTxLeakDetector_CommitInSelect(t *testing.T) {
	src := `
package main

import "database/sql"

func commitInSelect(db *sql.DB, ch chan bool) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec("INSERT INTO users (name) VALUES ('test')")
	if err != nil {
		tx.Rollback()
		return err
	}

	select {
	case <-ch:
		return tx.Commit()
	}
	// Unreachable, but select might block forever
	return nil
}
`
	violations := parseTxLeakAndDetect(t, src)

	found := false
	for _, v := range violations {
		if v.ViolationType == "commit_in_select" {
			found = true
			if v.Severity != TxLeakSeverityMedium {
				t.Errorf("expected severity medium, got %s", v.Severity)
			}
			break
		}
	}
	if !found {
		t.Error("expected 'commit_in_select' violation")
	}
}

func TestTxLeakDetector_FatalPath(t *testing.T) {
	src := `
package main

import (
	"database/sql"
	"log"
)

func fatalPath(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec("INSERT INTO users (name) VALUES ('test')")
	if err != nil {
		log.Fatal(err) // Fatal will not run defer
	}

	return tx.Commit()
}
`
	violations := parseTxLeakAndDetect(t, src)

	found := false
	for _, v := range violations {
		if v.ViolationType == "fatal_without_defer" {
			found = true
			if v.Severity != TxLeakSeverityHigh {
				t.Errorf("expected severity high, got %s", v.Severity)
			}
			break
		}
	}
	if !found {
		t.Error("expected 'fatal_without_defer' violation")
	}
}

func TestTxLeakDetector_OsExitPath(t *testing.T) {
	src := `
package main

import (
	"database/sql"
	"os"
)

func osExitPath(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec("INSERT INTO users (name) VALUES ('test')")
	if err != nil {
		os.Exit(1) // os.Exit will not run defer
	}

	return tx.Commit()
}
`
	violations := parseTxLeakAndDetect(t, src)

	found := false
	for _, v := range violations {
		if v.ViolationType == "fatal_without_defer" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'fatal_without_defer' violation for os.Exit")
	}
}

func TestTxLeakDetector_CommitInLoop(t *testing.T) {
	src := `
package main

import "database/sql"

func commitInLoop(db *sql.DB, items []string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	for _, item := range items {
		_, err = tx.Exec("INSERT INTO users (name) VALUES (?)", item)
		if err != nil {
			tx.Rollback()
			return err
		}
		return tx.Commit() // Commit inside loop!
	}
	// If items is empty, Commit never runs
	return nil
}
`
	violations := parseTxLeakAndDetect(t, src)

	found := false
	for _, v := range violations {
		if v.ViolationType == "commit_in_loop" {
			found = true
			if v.Severity != TxLeakSeverityMedium {
				t.Errorf("expected severity medium, got %s", v.Severity)
			}
			break
		}
	}
	if !found {
		t.Error("expected 'commit_in_loop' violation")
	}
}

func TestTxLeakDetector_VariableReassignment(t *testing.T) {
	src := `
package main

import "database/sql"

func variableReassignment(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec("INSERT INTO users (name) VALUES ('test1')")
	if err != nil {
		tx.Rollback()
		return err
	}

	// Reassign tx without committing first transaction!
	tx, err = db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec("INSERT INTO users (name) VALUES ('test2')")
	if err != nil {
		return err
	}
	return tx.Commit()
}
`
	violations := parseTxLeakAndDetect(t, src)

	found := false
	for _, v := range violations {
		if v.ViolationType == "variable_reassignment" {
			found = true
			if v.Severity != TxLeakSeverityHigh {
				t.Errorf("expected severity high, got %s", v.Severity)
			}
			break
		}
	}
	if !found {
		t.Error("expected 'variable_reassignment' violation")
	}
}

func TestTxLeakDetector_CommitErrorIgnored(t *testing.T) {
	src := `
package main

import "database/sql"

func commitErrorIgnored(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec("INSERT INTO users (name) VALUES ('test')")
	if err != nil {
		return err
	}

	_ = tx.Commit() // Error ignored!
	return nil
}
`
	violations := parseTxLeakAndDetect(t, src)

	found := false
	for _, v := range violations {
		if v.ViolationType == "commit_error_ignored" {
			found = true
			if v.Severity != TxLeakSeverityLow {
				t.Errorf("expected severity low, got %s", v.Severity)
			}
			break
		}
	}
	if !found {
		t.Error("expected 'commit_error_ignored' violation")
	}
}

// ============================================================================
// FALSE POSITIVE TESTS - Proper patterns that should NOT flag
// ============================================================================

func TestTxLeakDetector_SwitchWithDefault(t *testing.T) {
	src := `
package main

import "database/sql"

func switchWithDefault(db *sql.DB, action string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec("INSERT INTO users (name) VALUES ('test')")
	if err != nil {
		return err
	}

	switch action {
	case "save":
		return tx.Commit()
	case "discard":
		return nil
	default:
		return tx.Commit() // Default case covers all
	}
}
`
	violations := parseTxLeakAndDetect(t, src)

	// Should not have commit_in_switch because defer exists
	for _, v := range violations {
		if v.ViolationType == "commit_in_switch" {
			t.Errorf("unexpected commit_in_switch violation when defer exists: %s", v.Message)
		}
	}
}

func TestTxLeakDetector_SelectWithDefault(t *testing.T) {
	src := `
package main

import "database/sql"

func selectWithDefault(db *sql.DB, ch chan bool) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec("INSERT INTO users (name) VALUES ('test')")
	if err != nil {
		return err
	}

	select {
	case <-ch:
		return tx.Commit()
	default:
		return tx.Commit() // Default ensures commit runs
	}
}
`
	violations := parseTxLeakAndDetect(t, src)

	// Should not have commit_in_select because defer exists
	for _, v := range violations {
		if v.ViolationType == "commit_in_select" {
			t.Errorf("unexpected commit_in_select violation when defer exists: %s", v.Message)
		}
	}
}

func TestTxLeakDetector_FatalWithDefer(t *testing.T) {
	src := `
package main

import (
	"database/sql"
	"log"
)

func fatalWithDefer(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() // Note: defer won't run on os.Exit/log.Fatal, but this is the best effort

	_, err = tx.Exec("INSERT INTO users (name) VALUES ('test')")
	if err != nil {
		log.Fatal(err)
	}

	return tx.Commit()
}
`
	violations := parseTxLeakAndDetect(t, src)

	// Should not have fatal_without_defer because defer exists
	for _, v := range violations {
		if v.ViolationType == "fatal_without_defer" {
			t.Errorf("unexpected fatal_without_defer violation when defer exists: %s", v.Message)
		}
	}
}

func TestTxLeakDetector_CommitAfterLoop(t *testing.T) {
	src := `
package main

import "database/sql"

func commitAfterLoop(db *sql.DB, items []string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, item := range items {
		_, err = tx.Exec("INSERT INTO users (name) VALUES (?)", item)
		if err != nil {
			return err
		}
	}
	// Commit is after the loop - this is fine
	return tx.Commit()
}
`
	violations := parseTxLeakAndDetect(t, src)

	// Should not have commit_in_loop because commit is after loop
	for _, v := range violations {
		if v.ViolationType == "commit_in_loop" {
			t.Errorf("unexpected commit_in_loop violation when commit is after loop: %s", v.Message)
		}
	}
}

func TestTxLeakDetector_CommitErrorHandled(t *testing.T) {
	src := `
package main

import "database/sql"

func commitErrorHandled(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec("INSERT INTO users (name) VALUES ('test')")
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}
`
	violations := parseTxLeakAndDetect(t, src)

	// Should not have commit_error_ignored because error is handled
	for _, v := range violations {
		if v.ViolationType == "commit_error_ignored" {
			t.Errorf("unexpected commit_error_ignored violation when error is handled: %s", v.Message)
		}
	}
}

// ============================================================================
// TEST FILE EXCLUSION TESTS
// ============================================================================

func TestTxLeakDetector_SkipsTestFunctions(t *testing.T) {
	src := `
package main

import (
	"database/sql"
	"testing"
)

func TestLeakyTransaction(t *testing.T) {
	db, _ := sql.Open("postgres", "")
	tx, _ := db.Begin()
	_, _ = tx.Exec("INSERT INTO users (name) VALUES ('test')")
	// Intentionally no commit/rollback - this is a test function
}

func BenchmarkLeaky(b *testing.B) {
	db, _ := sql.Open("postgres", "")
	for i := 0; i < b.N; i++ {
		tx, _ := db.Begin()
		_, _ = tx.Exec("INSERT INTO users (name) VALUES ('test')")
	}
}

func ExampleLeaky() {
	db, _ := sql.Open("postgres", "")
	tx, _ := db.Begin()
	tx.Exec("INSERT INTO users (name) VALUES ('test')")
	// Output:
}
`
	violations := parseTxLeakAndDetect(t, src)

	if len(violations) != 0 {
		t.Errorf("expected 0 violations for test functions, got %d", len(violations))
		for _, v := range violations {
			t.Logf("  violation: %s - %s", v.ViolationType, v.Message)
		}
	}
}

func TestTxLeakDetector_SkipsTestHelperFunctions(t *testing.T) {
	src := `
package main

import (
	"database/sql"
	"testing"
)

func setupTx(t *testing.T, db *sql.DB) *sql.Tx {
	tx, _ := db.Begin()
	// Test helper - no commit/rollback, caller handles it
	return tx
}

func helperWithB(b *testing.B, db *sql.DB) {
	tx, _ := db.Begin()
	_, _ = tx.Exec("INSERT INTO users (name) VALUES ('test')")
	// No commit/rollback - this is a test helper
}
`
	violations := parseTxLeakAndDetect(t, src)

	if len(violations) != 0 {
		t.Errorf("expected 0 violations for test helper functions, got %d", len(violations))
		for _, v := range violations {
			t.Logf("  violation: %s - %s", v.ViolationType, v.Message)
		}
	}
}

func TestTxLeakDetector_DetectsNonTestFunctions(t *testing.T) {
	src := `
package main

import "database/sql"

// Not a test function - should be detected
func notATest(db *sql.DB) error {
	tx, _ := db.Begin()
	_, _ = tx.Exec("INSERT INTO users (name) VALUES ('test')")
	return nil // Missing commit/rollback
}

// Has similar name but not a real test function
func TestingTheWaters(db *sql.DB) {
	tx, _ := db.Begin()
	_, _ = tx.Exec("INSERT INTO users (name) VALUES ('test')")
}
`
	violations := parseTxLeakAndDetect(t, src)

	if len(violations) < 2 {
		t.Errorf("expected at least 2 violations for non-test functions, got %d", len(violations))
	}
}

// ============================================================================
// NEW FALSE NEGATIVE TESTS (previously uncovered scenarios)
// ============================================================================

// Test: Ent ORM Tx() method should be detected
func TestTxLeakDetector_EntOrmTx(t *testing.T) {
	src := `
package main

import "context"

type Client struct{}

func (c *Client) Tx(ctx context.Context) (*Tx, error) {
	return &Tx{}, nil
}

type Tx struct{}

func (t *Tx) Commit() error { return nil }
func (t *Tx) Rollback() error { return nil }

func entLeaky(ctx context.Context, client *Client) error {
	tx, err := client.Tx(ctx)
	if err != nil {
		return err
	}
	// Missing commit/rollback
	return nil
}
`
	violations := parseTxLeakAndDetect(t, src)

	if len(violations) < 1 {
		t.Fatalf("expected at least 1 violation for Ent Tx(), got %d", len(violations))
	}

	found := false
	for _, v := range violations {
		if v.Severity == TxLeakSeverityCritical && v.ViolationType == "no_commit_rollback" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected critical violation for Ent Tx() without commit/rollback")
	}
}

// Test: Defer with function call that takes tx as parameter
func TestTxLeakDetector_DeferWithTxParameter(t *testing.T) {
	src := `
package main

import "database/sql"

func cleanup(tx *sql.Tx) {
	tx.Rollback()
}

func deferWithParam(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer cleanup(tx) // Defer calls cleanup(tx) which does Rollback

	_, err = tx.Exec("INSERT INTO users (name) VALUES ('test')")
	if err != nil {
		return err
	}
	return tx.Commit()
}
`
	violations := parseTxLeakAndDetect(t, src)

	// Should NOT have critical/high violations because defer cleanup(tx) handles rollback
	for _, v := range violations {
		if v.Severity == TxLeakSeverityCritical || v.ViolationType == "no_rollback" {
			t.Errorf("unexpected violation for defer with tx parameter: %s - %s", v.ViolationType, v.Message)
		}
	}
}

// Test: Defer declared BEFORE Begin
func TestTxLeakDetector_DeferBeforeBegin(t *testing.T) {
	src := `
package main

import "database/sql"

func deferBeforeBegin(db *sql.DB) error {
	var tx *sql.Tx
	var err error
	defer func() {
		if tx != nil {
			tx.Rollback()
		}
	}()

	tx, err = db.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec("INSERT INTO users (name) VALUES ('test')")
	if err != nil {
		return err
	}
	return tx.Commit()
}
`
	violations := parseTxLeakAndDetect(t, src)

	// Should NOT have critical violations because defer handles rollback
	for _, v := range violations {
		if v.Severity == TxLeakSeverityCritical {
			t.Errorf("unexpected critical violation for defer before begin: %s - %s", v.ViolationType, v.Message)
		}
	}
}

// Test: Multiple return paths with different tx variables
func TestTxLeakDetector_MultipleReturnPathsDifferentTx(t *testing.T) {
	src := `
package main

import "database/sql"

func multiplePathsDifferentTx(db *sql.DB, flag bool) error {
	if flag {
		tx1, err := db.Begin()
		if err != nil {
			return err
		}
		defer tx1.Rollback()
		return tx1.Commit()
	} else {
		tx2, err := db.Begin()
		if err != nil {
			return err
		}
		// Forgot defer for tx2!
		return tx2.Commit()
	}
}
`
	violations := parseTxLeakAndDetect(t, src)

	// Should have violation for tx2 (no rollback)
	found := false
	for _, v := range violations {
		if v.TxVarName == "tx2" && v.ViolationType == "no_rollback" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'no_rollback' violation for tx2 in else branch")
	}

	// Should NOT have violation for tx1
	for _, v := range violations {
		if v.TxVarName == "tx1" && (v.Severity == TxLeakSeverityCritical || v.Severity == TxLeakSeverityHigh) {
			t.Errorf("unexpected high/critical violation for tx1 which has defer: %s", v.Message)
		}
	}
}

// Test: Rollback in goroutine's defer (should still warn - goroutine defer is risky)
func TestTxLeakDetector_RollbackInGoroutineDefer(t *testing.T) {
	src := `
package main

import "database/sql"

func rollbackInGoroutineDefer(db *sql.DB) {
	tx, err := db.Begin()
	if err != nil {
		return
	}
	// No defer in main function!

	go func() {
		defer tx.Rollback() // Defer is in goroutine, not main function
		tx.Exec("INSERT INTO users (name) VALUES ('test')")
		tx.Commit()
	}()
	// Main function returns immediately, goroutine might not complete
}
`
	violations := parseTxLeakAndDetect(t, src)

	// Should have goroutine_capture or similar warning
	found := false
	for _, v := range violations {
		if v.ViolationType == "goroutine_capture" || v.Severity == TxLeakSeverityHigh {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected warning for tx with defer only in goroutine")
	}
}

// Test: Named return with complex error handling
func TestTxLeakDetector_NamedReturnWithRecover(t *testing.T) {
	src := `
package main

import (
	"database/sql"
	"fmt"
)

func namedReturnWithRecover(db *sql.DB) (err error) {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			err = fmt.Errorf("panic: %v", p)
		}
	}()

	_, err = tx.Exec("INSERT INTO users (name) VALUES ('test')")
	if err != nil {
		return err
	}
	return tx.Commit()
}
`
	violations := parseTxLeakAndDetect(t, src)

	// Should NOT have critical violation - defer with recover handles panic
	// But might have no_rollback for non-panic error paths
	hasCritical := false
	for _, v := range violations {
		if v.Severity == TxLeakSeverityCritical {
			hasCritical = true
			break
		}
	}
	if hasCritical {
		t.Error("unexpected critical violation for named return with recover pattern")
	}
}

// Test: Interface method Begin should be detected
func TestTxLeakDetector_InterfaceMethodBegin(t *testing.T) {
	src := `
package main

type DB interface {
	Begin() (Tx, error)
}

type Tx interface {
	Commit() error
	Rollback() error
}

func interfaceBegin(db DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	// Missing commit/rollback
	return nil
}
`
	violations := parseTxLeakAndDetect(t, src)

	if len(violations) < 1 {
		t.Fatalf("expected at least 1 violation for interface Begin(), got %d", len(violations))
	}
}

// Test: pgx pool.Begin should be detected
func TestTxLeakDetector_PgxPoolBegin(t *testing.T) {
	src := `
package main

import "context"

type Pool struct{}

func (p *Pool) Begin(ctx context.Context) (*Tx, error) {
	return &Tx{}, nil
}

type Tx struct{}

func (t *Tx) Commit(ctx context.Context) error { return nil }
func (t *Tx) Rollback(ctx context.Context) error { return nil }

func pgxPoolLeaky(ctx context.Context, pool *Pool) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	// Missing commit/rollback
	return nil
}
`
	violations := parseTxLeakAndDetect(t, src)

	if len(violations) < 1 {
		t.Fatalf("expected at least 1 violation for pgx pool.Begin(), got %d", len(violations))
	}
}

// Test: tx stored in struct field without defer - should warn
func TestTxLeakDetector_StoredInStructNoDefer(t *testing.T) {
	src := `
package main

import "database/sql"

type Service struct {
	tx *sql.Tx
}

func (s *Service) startTx(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	s.tx = tx // Stored in struct, no defer
	return nil
}
`
	violations := parseTxLeakAndDetect(t, src)

	// Should have violation because tx stored in struct without defer
	if len(violations) < 1 {
		t.Fatalf("expected at least 1 violation for tx stored in struct without defer, got %d", len(violations))
	}

	found := false
	for _, v := range violations {
		if v.Severity == TxLeakSeverityCritical {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected critical violation for tx stored in struct without defer")
	}
}

// ============================================================================
// NEW FALSE POSITIVE TESTS (correct code that should NOT be flagged)
// ============================================================================

// Test: Ent ORM with proper handling
func TestTxLeakDetector_EntOrmProper(t *testing.T) {
	src := `
package main

import "context"

type Client struct{}

func (c *Client) Tx(ctx context.Context) (*Tx, error) {
	return &Tx{}, nil
}

type Tx struct{}

func (t *Tx) Commit() error { return nil }
func (t *Tx) Rollback() error { return nil }

func entProper(ctx context.Context, client *Client) error {
	tx, err := client.Tx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// ... do work ...
	return tx.Commit()
}
`
	violations := parseTxLeakAndDetect(t, src)

	// Should have no critical/high violations
	for _, v := range violations {
		if v.Severity == TxLeakSeverityCritical || v.Severity == TxLeakSeverityHigh {
			t.Errorf("unexpected critical/high violation for proper Ent pattern: %s - %s", v.ViolationType, v.Message)
		}
	}
}

// Test: WithTx callback pattern
func TestTxLeakDetector_WithTxCallback(t *testing.T) {
	src := `
package main

type DB interface {
	WithTx(fn func(tx interface{}) error) error
}

func withTxPattern(db DB) error {
	return db.WithTx(func(tx interface{}) error {
		// tx lifecycle managed by WithTx
		return nil
	})
}
`
	violations := parseTxLeakAndDetect(t, src)

	if len(violations) != 0 {
		t.Errorf("expected 0 violations for WithTx callback, got %d", len(violations))
		for _, v := range violations {
			t.Logf("  violation: %s - %s", v.ViolationType, v.Message)
		}
	}
}

// Test: InTransaction callback pattern
func TestTxLeakDetector_InTransactionCallback(t *testing.T) {
	src := `
package main

type DB interface {
	InTransaction(fn func(tx interface{}) error) error
}

func inTxPattern(db DB) error {
	return db.InTransaction(func(tx interface{}) error {
		return nil
	})
}
`
	violations := parseTxLeakAndDetect(t, src)

	if len(violations) != 0 {
		t.Errorf("expected 0 violations for InTransaction callback, got %d", len(violations))
	}
}

// Test: Transactional callback pattern
func TestTxLeakDetector_TransactionalCallback(t *testing.T) {
	src := `
package main

type DB interface {
	Transactional(fn func(tx interface{}) error) error
}

func transactionalPattern(db DB) error {
	return db.Transactional(func(tx interface{}) error {
		return nil
	})
}
`
	violations := parseTxLeakAndDetect(t, src)

	if len(violations) != 0 {
		t.Errorf("expected 0 violations for Transactional callback, got %d", len(violations))
	}
}

// Test: ExecTx callback pattern
func TestTxLeakDetector_ExecTxCallback(t *testing.T) {
	src := `
package main

type DB interface {
	ExecTx(fn func(tx interface{}) error) error
}

func execTxPattern(db DB) error {
	return db.ExecTx(func(tx interface{}) error {
		return nil
	})
}
`
	violations := parseTxLeakAndDetect(t, src)

	if len(violations) != 0 {
		t.Errorf("expected 0 violations for ExecTx callback, got %d", len(violations))
	}
}

// Test: DoInTx callback pattern
func TestTxLeakDetector_DoInTxCallback(t *testing.T) {
	src := `
package main

type DB interface {
	DoInTx(fn func(tx interface{}) error) error
}

func doInTxPattern(db DB) error {
	return db.DoInTx(func(tx interface{}) error {
		return nil
	})
}
`
	violations := parseTxLeakAndDetect(t, src)

	if len(violations) != 0 {
		t.Errorf("expected 0 violations for DoInTx callback, got %d", len(violations))
	}
}

// Test: tx stored in struct WITH defer - should NOT warn
func TestTxLeakDetector_StoredInStructWithDefer(t *testing.T) {
	src := `
package main

import "database/sql"

type Service struct {
	tx *sql.Tx
}

func (s *Service) startTxWithDefer(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() // Has defer as safety net
	s.tx = tx
	return tx.Commit()
}
`
	violations := parseTxLeakAndDetect(t, src)

	// Should have no critical violations
	for _, v := range violations {
		if v.Severity == TxLeakSeverityCritical {
			t.Errorf("unexpected critical violation for tx stored in struct with defer: %s", v.Message)
		}
	}
}

// Test: Passed to function WITHOUT defer but function name suggests tx handling
func TestTxLeakDetector_PassedToCommitFunction(t *testing.T) {
	src := `
package main

import "database/sql"

func commitTransaction(tx *sql.Tx) error {
	return tx.Commit()
}

func passedToCommitFunc(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// tx is passed to function that commits
	return commitTransaction(tx)
}
`
	violations := parseTxLeakAndDetect(t, src)

	// Should have no critical violations - has defer
	for _, v := range violations {
		if v.Severity == TxLeakSeverityCritical {
			t.Errorf("unexpected critical violation for passed to commit function: %s", v.Message)
		}
	}
}

// Test: sqlx MustBeginTx should be detected
func TestTxLeakDetector_SQLxMustBeginTx(t *testing.T) {
	src := `
package main

import "context"

type DB struct{}

func (d *DB) MustBeginTx(ctx context.Context, opts interface{}) *Tx {
	return &Tx{}
}

type Tx struct{}

func (t *Tx) Commit() error { return nil }
func (t *Tx) Rollback() error { return nil }

func sqlxMustBeginTxLeaky(ctx context.Context, db *DB) error {
	tx := db.MustBeginTx(ctx, nil)
	// Missing commit/rollback
	_ = tx
	return nil
}
`
	violations := parseTxLeakAndDetect(t, src)

	if len(violations) < 1 {
		t.Fatalf("expected at least 1 violation for MustBeginTx, got %d", len(violations))
	}
}

// Test: Beginx should be detected (sqlx)
func TestTxLeakDetector_SQLxBeginx(t *testing.T) {
	src := `
package main

type DB struct{}

func (d *DB) Beginx() (*Tx, error) {
	return &Tx{}, nil
}

type Tx struct{}

func (t *Tx) Commit() error { return nil }
func (t *Tx) Rollback() error { return nil }

func sqlxBeginxLeaky(db *DB) error {
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	// Missing commit/rollback
	_ = tx
	return nil
}
`
	violations := parseTxLeakAndDetect(t, src)

	if len(violations) < 1 {
		t.Fatalf("expected at least 1 violation for Beginx, got %d", len(violations))
	}
}

// Test: bun RunInTx callback - should NOT be flagged
func TestTxLeakDetector_BunRunInTxCallback(t *testing.T) {
	src := `
package main

import "context"

type DB interface {
	RunInTx(ctx context.Context, opts interface{}, fn func(ctx context.Context, tx interface{}) error) error
}

func bunRunInTx(ctx context.Context, db DB) error {
	return db.RunInTx(ctx, nil, func(ctx context.Context, tx interface{}) error {
		// tx lifecycle managed by RunInTx
		return nil
	})
}
`
	violations := parseTxLeakAndDetect(t, src)

	if len(violations) != 0 {
		t.Errorf("expected 0 violations for bun RunInTx callback, got %d", len(violations))
		for _, v := range violations {
			t.Logf("  violation: %s - %s", v.ViolationType, v.Message)
		}
	}
}

// Test: pgx BeginFunc callback - should NOT be flagged
func TestTxLeakDetector_PgxBeginFuncCallback(t *testing.T) {
	src := `
package main

import "context"

type Pool interface {
	BeginFunc(ctx context.Context, fn func(interface{}) error) error
}

func pgxBeginFunc(ctx context.Context, pool Pool) error {
	return pool.BeginFunc(ctx, func(tx interface{}) error {
		// tx lifecycle managed by BeginFunc
		return nil
	})
}
`
	violations := parseTxLeakAndDetect(t, src)

	if len(violations) != 0 {
		t.Errorf("expected 0 violations for pgx BeginFunc callback, got %d", len(violations))
		for _, v := range violations {
			t.Logf("  violation: %s - %s", v.ViolationType, v.Message)
		}
	}
}

// Test: FuzzXxx function should be skipped
func TestTxLeakDetector_SkipsFuzzFunctions(t *testing.T) {
	src := `
package main

import (
	"database/sql"
	"testing"
)

func FuzzLeakyTransaction(f *testing.F) {
	db, _ := sql.Open("postgres", "")
	tx, _ := db.Begin()
	_, _ = tx.Exec("INSERT INTO users (name) VALUES ('test')")
	// Intentionally no commit/rollback - this is a fuzz function
}
`
	violations := parseTxLeakAndDetect(t, src)

	if len(violations) != 0 {
		t.Errorf("expected 0 violations for fuzz functions, got %d", len(violations))
		for _, v := range violations {
			t.Logf("  violation: %s - %s", v.ViolationType, v.Message)
		}
	}
}

// Test: Helper with testing.M should be skipped
func TestTxLeakDetector_SkipsTestMainHelper(t *testing.T) {
	src := `
package main

import (
	"database/sql"
	"testing"
)

func setupMain(m *testing.M, db *sql.DB) {
	tx, _ := db.Begin()
	_, _ = tx.Exec("INSERT INTO users (name) VALUES ('test')")
	// No commit/rollback - this is a test main helper
}
`
	violations := parseTxLeakAndDetect(t, src)

	if len(violations) != 0 {
		t.Errorf("expected 0 violations for test main helper, got %d", len(violations))
		for _, v := range violations {
			t.Logf("  violation: %s - %s", v.ViolationType, v.Message)
		}
	}
}

// Test: Commit and Rollback in different branches (both paths covered)
func TestTxLeakDetector_BothPathsCovered(t *testing.T) {
	src := `
package main

import "database/sql"

func bothPathsCovered(db *sql.DB, success bool) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec("INSERT INTO users (name) VALUES ('test')")
	if err != nil {
		tx.Rollback()
		return err
	}

	if success {
		return tx.Commit()
	} else {
		tx.Rollback()
		return nil
	}
}
`
	violations := parseTxLeakAndDetect(t, src)

	// Should NOT have no_commit_rollback - both are present
	for _, v := range violations {
		if v.ViolationType == "no_commit_rollback" {
			t.Errorf("unexpected no_commit_rollback violation: %s", v.Message)
		}
	}
}

// Test: log.Fatalf should be detected
func TestTxLeakDetector_LogFatalf(t *testing.T) {
	src := `
package main

import (
	"database/sql"
	"log"
)

func logFatalfPath(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec("INSERT INTO users (name) VALUES ('test')")
	if err != nil {
		log.Fatalf("failed: %v", err) // Fatalf will not run defer
	}

	return tx.Commit()
}
`
	violations := parseTxLeakAndDetect(t, src)

	found := false
	for _, v := range violations {
		if v.ViolationType == "fatal_without_defer" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'fatal_without_defer' violation for log.Fatalf")
	}
}

// Test: log.Fatalln should be detected
func TestTxLeakDetector_LogFatalln(t *testing.T) {
	src := `
package main

import (
	"database/sql"
	"log"
)

func logFatallnPath(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec("INSERT INTO users (name) VALUES ('test')")
	if err != nil {
		log.Fatalln("failed:", err) // Fatalln will not run defer
	}

	return tx.Commit()
}
`
	violations := parseTxLeakAndDetect(t, src)

	found := false
	for _, v := range violations {
		if v.ViolationType == "fatal_without_defer" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'fatal_without_defer' violation for log.Fatalln")
	}
}

// Test: For loop without range
func TestTxLeakDetector_CommitInForLoop(t *testing.T) {
	src := `
package main

import "database/sql"

func commitInForLoop(db *sql.DB, count int) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	for i := 0; i < count; i++ {
		_, err = tx.Exec("INSERT INTO users (id) VALUES (?)", i)
		if err != nil {
			tx.Rollback()
			return err
		}
		return tx.Commit() // Commit inside for loop!
	}
	// If count is 0, Commit never runs
	return nil
}
`
	violations := parseTxLeakAndDetect(t, src)

	found := false
	for _, v := range violations {
		if v.ViolationType == "commit_in_loop" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'commit_in_loop' violation for commit inside for loop")
	}
}

// Test: NewTx method should be detected
func TestTxLeakDetector_NewTx(t *testing.T) {
	src := `
package main

type DB struct{}

func (d *DB) NewTx() (*Tx, error) {
	return &Tx{}, nil
}

type Tx struct{}

func (t *Tx) Commit() error { return nil }
func (t *Tx) Rollback() error { return nil }

func newTxLeaky(db *DB) error {
	tx, err := db.NewTx()
	if err != nil {
		return err
	}
	// Missing commit/rollback
	_ = tx
	return nil
}
`
	violations := parseTxLeakAndDetect(t, src)

	if len(violations) < 1 {
		t.Fatalf("expected at least 1 violation for NewTx, got %d", len(violations))
	}
}

// ============================================================================
// CHANNEL SEND TESTS
// ============================================================================

// Test: TX sent through channel - should NOT be flagged (lifecycle managed by receiver)
func TestTxLeakDetector_ChannelSend(t *testing.T) {
	src := `
package main

import "database/sql"

func channelPass(db *sql.DB, ch chan *sql.Tx) {
	tx, _ := db.Begin()
	ch <- tx  // TX sent through channel - receiver manages lifecycle
}
`
	violations := parseTxLeakAndDetect(t, src)

	// Should NOT have no_commit_rollback because tx is sent through channel
	for _, v := range violations {
		if v.ViolationType == "no_commit_rollback" {
			t.Errorf("unexpected no_commit_rollback violation for channel send: %s", v.Message)
		}
	}
}

// Test: TX pointer sent through channel - should NOT be flagged
func TestTxLeakDetector_ChannelSendPointer(t *testing.T) {
	src := `
package main

import "database/sql"

func channelPassPointer(db *sql.DB, ch chan *sql.Tx) {
	tx, _ := db.Begin()
	ch <- &tx  // TX pointer sent through channel
}
`
	violations := parseTxLeakAndDetect(t, src)

	// Should NOT have no_commit_rollback because tx is sent through channel
	for _, v := range violations {
		if v.ViolationType == "no_commit_rollback" {
			t.Errorf("unexpected no_commit_rollback violation for channel send pointer: %s", v.Message)
		}
	}
}

// ============================================================================
// CLOSURE RETURN TESTS
// ============================================================================

// Test: TX captured by returned closure - should NOT be flagged
func TestTxLeakDetector_ClosureReturn(t *testing.T) {
	src := `
package main

import "database/sql"

func closureReturn(db *sql.DB) func() error {
	tx, _ := db.Begin()
	return func() error { return tx.Commit() }  // TX captured by returned closure
}
`
	violations := parseTxLeakAndDetect(t, src)

	// Should NOT have no_commit_rollback because tx is captured by returned closure
	for _, v := range violations {
		if v.ViolationType == "no_commit_rollback" {
			t.Errorf("unexpected no_commit_rollback violation for closure return: %s", v.Message)
		}
	}
}

// Test: TX captured by returned closure with error handling - should NOT be flagged
func TestTxLeakDetector_ClosureReturnWithRollback(t *testing.T) {
	src := `
package main

import "database/sql"

func closureReturnWithRollback(db *sql.DB) (func() error, func() error) {
	tx, _ := db.Begin()
	commit := func() error { return tx.Commit() }
	rollback := func() error { return tx.Rollback() }
	return commit, rollback
}
`
	violations := parseTxLeakAndDetect(t, src)

	// Should NOT have violations - both commit and rollback are in closures
	for _, v := range violations {
		if v.ViolationType == "no_commit_rollback" || v.ViolationType == "no_rollback" {
			t.Errorf("unexpected violation for closure return: %s - %s", v.ViolationType, v.Message)
		}
	}
}

// ============================================================================
// MAP/SLICE STORAGE TESTS
// ============================================================================

// Test: TX stored in map - should NOT be flagged if has defer
func TestTxLeakDetector_MapStorageWithDefer(t *testing.T) {
	src := `
package main

import "database/sql"

func mapStorage(db *sql.DB, txMap map[string]*sql.Tx) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	txMap["key"] = tx  // TX stored in map
	return tx.Commit()
}
`
	violations := parseTxLeakAndDetect(t, src)

	// Should NOT have critical violations because defer exists
	for _, v := range violations {
		if v.Severity == TxLeakSeverityCritical {
			t.Errorf("unexpected critical violation for map storage with defer: %s", v.Message)
		}
	}
}

// Test: TX stored in map without defer - should be flagged
func TestTxLeakDetector_MapStorageNoDefer(t *testing.T) {
	src := `
package main

import "database/sql"

func mapStorageNoDefer(db *sql.DB, txMap map[string]*sql.Tx) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	txMap["key"] = tx  // TX stored in map without defer
	return nil
}
`
	violations := parseTxLeakAndDetect(t, src)

	// Should have critical violation
	found := false
	for _, v := range violations {
		if v.Severity == TxLeakSeverityCritical {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected critical violation for map storage without defer")
	}
}

// Test: TX appended to slice - should NOT be flagged if has defer
func TestTxLeakDetector_SliceAppendWithDefer(t *testing.T) {
	src := `
package main

import "database/sql"

func sliceAppend(db *sql.DB, txSlice []*sql.Tx) ([]*sql.Tx, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	txSlice = append(txSlice, tx)  // TX appended to slice
	return txSlice, tx.Commit()
}
`
	violations := parseTxLeakAndDetect(t, src)

	// Should NOT have critical violations because defer exists
	for _, v := range violations {
		if v.Severity == TxLeakSeverityCritical {
			t.Errorf("unexpected critical violation for slice append with defer: %s", v.Message)
		}
	}
}

// Test: TX stored in slice by index - should NOT be flagged if has defer
func TestTxLeakDetector_SliceIndexStorageWithDefer(t *testing.T) {
	src := `
package main

import "database/sql"

func sliceIndexStorage(db *sql.DB, txSlice []*sql.Tx) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	txSlice[0] = tx  // TX stored in slice by index
	return tx.Commit()
}
`
	violations := parseTxLeakAndDetect(t, src)

	// Should NOT have critical violations because defer exists
	for _, v := range violations {
		if v.Severity == TxLeakSeverityCritical {
			t.Errorf("unexpected critical violation for slice index storage with defer: %s", v.Message)
		}
	}
}

// ============================================================================
// DEFERRED COMMIT TESTS (antipattern)
// ============================================================================

// Test: defer tx.Commit() - should be flagged as antipattern
func TestTxLeakDetector_DeferredCommit(t *testing.T) {
	src := `
package main

import "database/sql"

func deferCommit(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Commit()  // Antipattern! Errors before defer not handled
	_, err = tx.Exec("INSERT INTO users (name) VALUES ('test')")
	return err
}
`
	violations := parseTxLeakAndDetect(t, src)

	found := false
	for _, v := range violations {
		if v.ViolationType == "deferred_commit" {
			found = true
			if v.Severity != TxLeakSeverityMedium {
				t.Errorf("expected severity medium for deferred_commit, got %s", v.Severity)
			}
			break
		}
	}
	if !found {
		t.Error("expected 'deferred_commit' violation for defer tx.Commit()")
	}
}

// Test: defer in closure with Commit() - should be flagged as antipattern
func TestTxLeakDetector_DeferredCommitInClosure(t *testing.T) {
	src := `
package main

import "database/sql"

func deferCommitInClosure(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		tx.Commit()  // Antipattern in closure too
	}()
	_, err = tx.Exec("INSERT INTO users (name) VALUES ('test')")
	return err
}
`
	violations := parseTxLeakAndDetect(t, src)

	found := false
	for _, v := range violations {
		if v.ViolationType == "deferred_commit" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'deferred_commit' violation for defer with Commit() in closure")
	}
}

// Test: Proper pattern (defer Rollback, explicit Commit) - should NOT be flagged
func TestTxLeakDetector_ProperDeferRollbackExplicitCommit(t *testing.T) {
	src := `
package main

import "database/sql"

func properPattern(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()  // Proper pattern

	_, err = tx.Exec("INSERT INTO users (name) VALUES ('test')")
	if err != nil {
		return err
	}
	return tx.Commit()  // Explicit commit at the end
}
`
	violations := parseTxLeakAndDetect(t, src)

	// Should NOT have deferred_commit violation
	for _, v := range violations {
		if v.ViolationType == "deferred_commit" {
			t.Errorf("unexpected deferred_commit violation for proper pattern: %s", v.Message)
		}
	}
}

// ============================================================================
// pgx BeginTxFunc CALLBACK TEST
// ============================================================================

// Test: pgx conn.BeginTxFunc callback - should NOT be flagged
func TestTxLeakDetector_PgxConnBeginTxFuncCallback(t *testing.T) {
	src := `
package main

import "context"

type Conn interface {
	BeginTxFunc(ctx context.Context, opts interface{}, fn func(interface{}) error) error
}

func pgxConnBeginTxFunc(ctx context.Context, conn Conn) error {
	return conn.BeginTxFunc(ctx, nil, func(tx interface{}) error {
		// tx lifecycle managed by BeginTxFunc
		return nil
	})
}
`
	violations := parseTxLeakAndDetect(t, src)

	if len(violations) != 0 {
		t.Errorf("expected 0 violations for pgx conn.BeginTxFunc callback, got %d", len(violations))
		for _, v := range violations {
			t.Logf("  violation: %s - %s", v.ViolationType, v.Message)
		}
	}
}

// ============================================================================
// ROLLBACK ERROR IGNORED TESTS
// ============================================================================

// Test: Rollback error ignored - should be flagged
func TestTxLeakDetector_RollbackErrorIgnored(t *testing.T) {
	src := `
package main

import "database/sql"

func rollbackErrorIgnored(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()  // Error ignored!
	}()

	_, err = tx.Exec("INSERT INTO users (name) VALUES ('test')")
	if err != nil {
		return err
	}
	return tx.Commit()
}
`
	violations := parseTxLeakAndDetect(t, src)

	found := false
	for _, v := range violations {
		if v.ViolationType == "rollback_error_ignored" {
			found = true
			if v.Severity != TxLeakSeverityLow {
				t.Errorf("expected severity low for rollback_error_ignored, got %s", v.Severity)
			}
			break
		}
	}
	if !found {
		t.Error("expected 'rollback_error_ignored' violation for _ = tx.Rollback()")
	}
}

// Test: Rollback error handled properly - should NOT be flagged
func TestTxLeakDetector_RollbackErrorHandled(t *testing.T) {
	src := `
package main

import "database/sql"

func rollbackErrorHandled(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err := tx.Rollback(); err != nil {
			// Log the error
		}
	}()

	_, err = tx.Exec("INSERT INTO users (name) VALUES ('test')")
	if err != nil {
		return err
	}
	return tx.Commit()
}
`
	violations := parseTxLeakAndDetect(t, src)

	// Should NOT have rollback_error_ignored violation
	for _, v := range violations {
		if v.ViolationType == "rollback_error_ignored" {
			t.Errorf("unexpected rollback_error_ignored violation when error is handled: %s", v.Message)
		}
	}
}

// ============================================================================
// DEFER IN LOOP TESTS
// ============================================================================

// Test: Defer inside for loop - should be flagged as antipattern
func TestTxLeakDetector_DeferInForLoop(t *testing.T) {
	src := `
package main

import "database/sql"

func deferInForLoop(db *sql.DB, items []string) error {
	for _, item := range items {
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()  // Antipattern! Defers pile up

		_, err = tx.Exec("INSERT INTO users (name) VALUES (?)", item)
		if err != nil {
			return err
		}
		tx.Commit()
	}
	return nil
}
`
	violations := parseTxLeakAndDetect(t, src)

	found := false
	for _, v := range violations {
		if v.ViolationType == "defer_in_loop" {
			found = true
			if v.Severity != TxLeakSeverityHigh {
				t.Errorf("expected severity high for defer_in_loop, got %s", v.Severity)
			}
			break
		}
	}
	if !found {
		t.Error("expected 'defer_in_loop' violation for defer inside for loop")
	}
}

// Test: Defer inside range loop - should be flagged
func TestTxLeakDetector_DeferInRangeLoop(t *testing.T) {
	src := `
package main

import "database/sql"

func deferInRangeLoop(db *sql.DB, ids []int) error {
	for _, id := range ids {
		tx, _ := db.Begin()
		defer tx.Rollback()  // Antipattern!
		tx.Exec("DELETE FROM users WHERE id = ?", id)
		tx.Commit()
	}
	return nil
}
`
	violations := parseTxLeakAndDetect(t, src)

	found := false
	for _, v := range violations {
		if v.ViolationType == "defer_in_loop" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'defer_in_loop' violation for defer inside range loop")
	}
}

// Test: Defer with closure inside loop - should be flagged
func TestTxLeakDetector_DeferClosureInLoop(t *testing.T) {
	src := `
package main

import "database/sql"

func deferClosureInLoop(db *sql.DB, items []string) error {
	for _, item := range items {
		tx, _ := db.Begin()
		defer func() {
			tx.Rollback()  // Antipattern - closure in loop
		}()
		tx.Exec("INSERT...", item)
		tx.Commit()
	}
	return nil
}
`
	violations := parseTxLeakAndDetect(t, src)

	found := false
	for _, v := range violations {
		if v.ViolationType == "defer_in_loop" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'defer_in_loop' violation for defer closure inside loop")
	}
}

// Test: Defer outside loop - should NOT be flagged
func TestTxLeakDetector_DeferOutsideLoop(t *testing.T) {
	src := `
package main

import "database/sql"

func deferOutsideLoop(db *sql.DB, items []string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()  // OK - outside loop

	for _, item := range items {
		_, err = tx.Exec("INSERT INTO users (name) VALUES (?)", item)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}
`
	violations := parseTxLeakAndDetect(t, src)

	// Should NOT have defer_in_loop violation
	for _, v := range violations {
		if v.ViolationType == "defer_in_loop" {
			t.Errorf("unexpected defer_in_loop violation when defer is outside loop: %s", v.Message)
		}
	}
}

// Test: Nested loop with defer in inner loop - should be flagged
func TestTxLeakDetector_DeferInNestedLoop(t *testing.T) {
	src := `
package main

import "database/sql"

func deferInNestedLoop(db *sql.DB, groups [][]string) error {
	for _, group := range groups {
		for _, item := range group {
			tx, _ := db.Begin()
			defer tx.Rollback()  // Antipattern - nested loop
			tx.Exec("INSERT...", item)
			tx.Commit()
		}
	}
	return nil
}
`
	violations := parseTxLeakAndDetect(t, src)

	found := false
	for _, v := range violations {
		if v.ViolationType == "defer_in_loop" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'defer_in_loop' violation for defer in nested loop")
	}
}

func parseTxLeakAndDetect(t *testing.T, src string) []TxLeakViolation {
	t.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("failed to parse source: %v", err)
	}

	return DetectTxLeaksInAST(fset, file)
}
