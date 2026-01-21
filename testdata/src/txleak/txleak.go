// Package txleak contains test cases for transaction leak detection.
package txleak

import (
	"context"
	"database/sql"
)

// ============================================================================
// FALSE NEGATIVE SCENARIOS (should be detected as problems)
// ============================================================================

// leakyTransaction - CRITICAL: No Commit or Rollback
func leakyTransaction(db *sql.DB) error {
	tx, err := db.Begin() // want "unclosed transaction: tx - missing both Commit\\(\\) and Rollback\\(\\)"
	if err != nil {
		return err
	}
	_, err = tx.Exec("INSERT INTO users (name) VALUES ('test')")
	if err != nil {
		return err
	}
	return nil // Oops - forgot to commit/rollback!
}

// noRollbackOnError - HIGH: Commit but no Rollback
func noRollbackOnError(db *sql.DB) error {
	tx, err := db.Begin() // want "transaction tx has Commit\\(\\) but no Rollback\\(\\) for error paths"
	if err != nil {
		return err
	}
	_, err = tx.Exec("INSERT INTO users (name) VALUES ('test')")
	if err != nil {
		return err // Leaks on error!
	}
	return tx.Commit()
}

// noCommit - MEDIUM: Rollback but no Commit
func noCommit(db *sql.DB) error {
	tx, err := db.Begin() // want "transaction tx has Rollback\\(\\) but missing Commit\\(\\)"
	if err != nil {
		return err
	}
	_, err = tx.Exec("INSERT INTO users (name) VALUES ('test')")
	if err != nil {
		tx.Rollback()
		return err
	}
	return nil // Missing commit!
}

// shadowingBug - HIGH: Variable shadowing causes outer tx to leak
func shadowingBug(db *sql.DB) error {
	tx, err := db.Begin() // want "transaction tx is shadowed by another transaction in inner scope"
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if true {
		tx, err := db.Begin() // This shadows the outer tx!
		if err != nil {
			return err
		}
		defer tx.Rollback()
		_ = tx.Commit() // Only commits inner tx
	}

	return tx.Commit() // This commits outer tx, but inner was shadowed
}

// conditionalCommit - MEDIUM: Commit might not execute
func conditionalCommit(db *sql.DB, shouldCommit bool) error {
	tx, err := db.Begin() // want "Commit\\(\\) is inside conditional"
	if err != nil {
		return err
	}

	_, err = tx.Exec("INSERT INTO users (name) VALUES ('test')")
	if err != nil {
		tx.Rollback()
		return err
	}

	if shouldCommit { // Commit only in one branch
		return tx.Commit()
	}

	return nil // tx never committed if shouldCommit is false!
}

// panicWithoutDefer - MEDIUM: Panic will leak transaction
func panicWithoutDefer(db *sql.DB) error {
	tx, err := db.Begin() // want "may leak if panic\\(\\) is called"
	if err != nil {
		return err
	}

	_, err = tx.Exec("INSERT INTO users (name) VALUES ('test')")
	if err != nil {
		panic(err) // Crashes before commit/rollback
	}

	return tx.Commit()
}

// goroutineCapture - HIGH: Transaction captured by goroutine
func goroutineCapture(db *sql.DB, ch chan error) {
	tx, err := db.Begin() // want "captured by goroutine without defer"
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
	// Function returns immediately, goroutine still holds tx
}

// earlyReturnWithoutDefer - HIGH: Early returns bypass commit
func earlyReturnWithoutDefer(db *sql.DB, items []string) error {
	tx, err := db.Begin() // want "has early return paths that bypass Commit\\(\\)"
	if err != nil {
		return err
	}

	if len(items) == 0 {
		return nil // Early return - tx never committed!
	}

	_, err = tx.Exec("INSERT INTO users (name) VALUES ('test')")
	if err != nil {
		return err // Another early return
	}

	return tx.Commit()
}

// BeginTxVariant - CRITICAL: BeginTx without Commit/Rollback
func BeginTxVariant(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil) // want "unclosed transaction: tx - missing both Commit\\(\\) and Rollback\\(\\)"
	if err != nil {
		return err
	}
	_, err = tx.Exec("INSERT INTO users (name) VALUES ('test')")
	return err
}

// multipleTransactionsOneBad - Multiple transactions, one leaks
func multipleTransactionsOneBad(db *sql.DB) error {
	tx1, err := db.Begin() // want "unclosed transaction: tx1"
	if err != nil {
		return err
	}
	_, _ = tx1.Exec("INSERT INTO table1 (id) VALUES (1)")
	// tx1 has no commit/rollback!

	tx2, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx2.Rollback()
	_, _ = tx2.Exec("INSERT INTO table2 (id) VALUES (2)")
	return tx2.Commit() // tx2 is properly handled
}

// ============================================================================
// FALSE POSITIVE SCENARIOS (should NOT be flagged)
// ============================================================================

// properDeferPattern - GOOD: Proper defer pattern
func properDeferPattern(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() // Safe - no-op after Commit

	_, err = tx.Exec("INSERT INTO users (name) VALUES ('test')")
	if err != nil {
		return err
	}
	return tx.Commit()
}

// properDeferWithCommit - GOOD: Both Rollback (deferred) and Commit
func properDeferWithCommit(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec("INSERT INTO users (name) VALUES ('test')")
	if err != nil {
		return err
	}

	_, err = tx.Exec("UPDATE users SET status = 'active' WHERE name = 'test'")
	if err != nil {
		return err
	}

	return tx.Commit()
}

// transactionFactory - GOOD: Returned to caller
func transactionFactory(db *sql.DB) (*sql.Tx, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	return tx, nil // OK - caller handles lifecycle
}

// closurePattern - GOOD: Defer with closure
func closurePattern(db *sql.DB) (err error) {
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

// passedToFunctionWithDefer - GOOD: Passed to function but has defer
func passedToFunctionWithDefer(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() // Has safety net

	if err := processWithTx(tx); err != nil {
		return err
	}
	return tx.Commit()
}

func processWithTx(tx *sql.Tx) error {
	_, err := tx.Exec("INSERT INTO users (name) VALUES ('test')")
	return err
}

// storedInStructWithDefer - GOOD: Stored in struct but has defer
func storedInStructWithDefer(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	repo := &Repository{}
	repo.tx = tx

	if err := repo.doWork(); err != nil {
		return err
	}
	return tx.Commit()
}

// Repository is a test struct for transaction storage
type Repository struct {
	tx *sql.Tx
}

func (r *Repository) doWork() error {
	_, err := r.tx.Exec("INSERT INTO users (name) VALUES ('test')")
	return err
}

// callbackPattern - GOOD: Using callback-based transaction
type DB interface {
	Transaction(func(tx interface{}) error) error
}

func callbackPattern(db DB) error {
	return db.Transaction(func(tx interface{}) error {
		// tx lifecycle managed automatically
		return nil
	})
}

// goroutineWithDefer - GOOD: Goroutine capture but has defer
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

	// Wait for goroutine
	if err := <-ch; err != nil {
		return
	}
	tx.Commit()
}

// panicWithDefer - GOOD: Has defer to handle panic
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

// nestedFunctions - GOOD: Transaction with proper handling
func nestedFunctions(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := insertUser(tx); err != nil {
		return err
	}
	return tx.Commit()
}

func insertUser(tx *sql.Tx) error {
	_, err := tx.Exec("INSERT INTO users (name) VALUES ('test')")
	return err
}

// ============================================================================
// NEW FALSE NEGATIVE SCENARIOS (should be detected as problems)
// ============================================================================

// entOrmLeaky - CRITICAL: Ent ORM Tx() without Commit/Rollback
func entOrmLeaky(ctx context.Context, client interface {
	Tx(context.Context) (interface {
		Commit() error
		Rollback() error
	}, error)
}) error {
	tx, err := client.Tx(ctx) // want "unclosed transaction: tx"
	if err != nil {
		return err
	}
	_ = tx
	return nil
}

// storedInStructNoDefer - CRITICAL: tx stored in struct without defer
type ServiceNoDefer struct {
	tx *sql.Tx
}

func (s *ServiceNoDefer) startTx(db *sql.DB) error {
	tx, err := db.Begin() // want "unclosed transaction: tx"
	if err != nil {
		return err
	}
	s.tx = tx // Stored in struct, no defer - dangerous!
	return nil
}

// rollbackInGoroutineOnly - HIGH: Defer only in goroutine, not main function
func rollbackInGoroutineOnly(db *sql.DB) {
	tx, err := db.Begin() // want "captured by goroutine"
	if err != nil {
		return
	}
	// No defer in main function!

	go func() {
		defer tx.Rollback() // Defer is in goroutine, not safe
		_, _ = tx.Exec("INSERT INTO users (name) VALUES ('test')")
		_ = tx.Commit()
	}()
	// Main function returns immediately
}

// commitInForLoop - MEDIUM: Commit inside for loop
func commitInForLoop(db *sql.DB, count int) error {
	tx, err := db.Begin() // want "Commit\\(\\) is inside loop"
	if err != nil {
		return err
	}

	for i := 0; i < count; i++ {
		_, err = tx.Exec("INSERT INTO users (id) VALUES (?)", i)
		if err != nil {
			tx.Rollback()
			return err
		}
		return tx.Commit() // Commit inside for loop - may never run if count=0
	}
	return nil
}

// ============================================================================
// NEW FALSE POSITIVE SCENARIOS (should NOT be flagged)
// ============================================================================

// deferWithTxParameter - GOOD: Defer with tx passed to cleanup function
func cleanup(tx *sql.Tx) {
	tx.Rollback()
}

func deferWithTxParameter(db *sql.DB) error {
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

// deferBeforeBegin - GOOD: Defer declared before Begin
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

// entOrmProper - GOOD: Ent ORM with proper handling
func entOrmProper(ctx context.Context, client interface {
	Tx(context.Context) (interface {
		Commit() error
		Rollback() error
	}, error)
}) error {
	tx, err := client.Tx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// ... do work ...
	return tx.Commit()
}

// multiplePathsDifferentTxProper - GOOD: Multiple return paths, each properly handled
func multiplePathsDifferentTxProper(db *sql.DB, flag bool) error {
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
		defer tx2.Rollback()
		return tx2.Commit()
	}
}

// bothPathsCovered - GOOD: Both commit and rollback paths covered
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

// Unused variable to prevent import errors
var _ = context.Background
