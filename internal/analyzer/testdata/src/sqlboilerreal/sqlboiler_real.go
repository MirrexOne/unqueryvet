// Package sqlboilerreal tests that REAL SQLBoiler code IS detected by the checker.
// This is an e2e test to confirm the checker works for actual SQLBoiler types.
package sqlboilerreal

import (
	"context"

	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/models"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
)

// fakeExecutor implements boil.ContextExecutor for testing
type fakeExecutor struct{}

func (f fakeExecutor) ExecContext(ctx context.Context, query string, args ...interface{}) (boil.Result, error) {
	return nil, nil
}
func (f fakeExecutor) QueryContext(ctx context.Context, query string, args ...interface{}) (boil.Rows, error) {
	return nil, nil
}

var db boil.ContextExecutor = fakeExecutor{}
var ctx = context.Background()

// =============================================================================
// TEST 1: qm.Select("*") should be detected
// =============================================================================

func TestQmSelectStar() {
	// This SHOULD trigger a warning because it's real SQLBoiler qm.Select("*")
	_ = qm.Select("*") // want "SQLBoiler qm.Select.*explicitly specify columns"
}

func TestQmSelectExplicit() {
	// This should NOT trigger a warning - explicit columns
	_ = qm.Select("id", "name", "email") // OK - explicit columns
}

// =============================================================================
// TEST 2: models.Users().All() without qm.Select should be detected
// This is the EXACT pattern from issue #5
// =============================================================================

func TestModelAllWithoutSelect() {
	// This SHOULD trigger: "SQLBoiler model().All() without qm.Select() defaults to SELECT *"
	_, _ = models.Users().All(ctx, db) // want "SQLBoiler model.*All.*without qm.Select.*defaults to SELECT"
}

func TestModelOneWithoutSelect() {
	// This SHOULD trigger warning for One() without Select
	_, _ = models.Users().One(ctx, db) // want "SQLBoiler model.*without qm.Select.*defaults to SELECT"
}

func TestModelAllWithSelectExplicit() {
	// This should NOT trigger - has explicit qm.Select with columns
	_, _ = models.Users(qm.Select("id", "name")).All(ctx, db) // OK - explicit columns
}

func TestModelAllWithSelectStar() {
	// This SHOULD trigger for qm.Select("*")
	_, _ = models.Users(qm.Select("*")).All(ctx, db) // want "SQLBoiler qm.Select.*explicitly specify columns"
}

// =============================================================================
// TEST 3: Other qm functions should NOT be flagged
// =============================================================================

func TestQmWhere() {
	_ = qm.Where("id = ?", 1) // OK - not Select
}

func TestQmOrderBy() {
	_ = qm.OrderBy("created_at DESC") // OK - not Select
}

func TestQmLimit() {
	_ = qm.Limit(10) // OK - not Select
}
