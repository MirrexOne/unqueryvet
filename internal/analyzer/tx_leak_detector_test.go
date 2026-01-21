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

func parseTxLeakAndDetect(t *testing.T, src string) []TxLeakViolation {
	t.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("failed to parse source: %v", err)
	}

	return DetectTxLeaksInAST(fset, file)
}
