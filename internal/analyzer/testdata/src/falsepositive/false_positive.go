// Package falsepositive tests that custom interfaces with methods like All(), One(), Count()
// do NOT trigger false positives from the SQLBoiler checker.
// This file should NOT produce any warnings.
package falsepositive

import "context"

// WidgetDB is a custom interface that mimics database patterns (issue #5 reproduction).
// This should NOT trigger any warnings because it's not from SQLBoiler.
type WidgetDB interface {
	All(ctx context.Context) ([]Widget, error)
}

type Widget struct {
	ID   int
	Name string
}

type widgetDBMock struct{}

func (m *widgetDBMock) All(ctx context.Context) ([]Widget, error) {
	return []Widget{{ID: 1, Name: "test"}}, nil
}

// UseWidgetDB demonstrates the exact pattern from issue #5.
// This should NOT trigger warnings - it's a mock, not SQLBoiler.
func UseWidgetDB() {
	var db WidgetDB = &widgetDBMock{}

	// This exact pattern was reported as false positive in issue #5
	_, _ = db.All(context.Background()) // OK - not SQLBoiler
}

// ========== EXACT ISSUE #5 REPRODUCTION ==========
// The issue reported this exact pattern with moq mocks:
//   widgetDBMoq.onCall().All(req.Context()).returnResults(...)
// Error was: "SQLBoiler model().All() without qm.Select() defaults to SELECT *"

// MockBuilder simulates the moq mock pattern from issue #5
type MockBuilder struct{}

func (m *MockBuilder) All(ctx context.Context) *MockResult {
	return &MockResult{}
}

type MockResult struct{}

func (r *MockResult) returnResults(widgets []Widget, err error) {}

// WidgetDBMoq simulates the exact moq mock from issue #5
type WidgetDBMoq struct{}

func (m *WidgetDBMoq) onCall() *MockBuilder {
	return &MockBuilder{}
}

// TestExactIssue5Pattern reproduces the EXACT pattern from issue #5
func TestExactIssue5Pattern() {
	widgetDBMoq := &WidgetDBMoq{}

	// THIS IS THE EXACT PATTERN FROM ISSUE #5:
	// widgetDBMoq.onCall().All(req.Context()).returnResults(...)
	// It was incorrectly flagged as: "SQLBoiler model().All() without qm.Select() defaults to SELECT *"
	widgetDBMoq.onCall().All(context.Background()).returnResults([]Widget{{ID: 1}}, nil) // OK - not SQLBoiler
}

// Also test the pattern: something().All() which matches SQLBoiler's model().All()
type FakeModelQuery struct{}

func (q *FakeModelQuery) All(ctx context.Context) ([]Widget, error) {
	return nil, nil
}

func FakeUsers() *FakeModelQuery {
	return &FakeModelQuery{}
}

func TestFakeModelPattern() {
	// This pattern looks EXACTLY like SQLBoiler: models.Users().All(ctx, db)
	// But it's not from SQLBoiler package, so should NOT trigger warnings
	_, _ = FakeUsers().All(context.Background()) // OK - not SQLBoiler
}

// CustomRepository is a custom interface that has methods with the same names
// as SQLBoiler methods, but is NOT from SQLBoiler package.
type CustomRepository interface {
	All() []Item
	One() *Item
	Count() int
	Exists() bool
}

type Item struct {
	ID   int
	Name string
}

type myRepo struct{}

func (r *myRepo) All() []Item  { return nil }
func (r *myRepo) One() *Item   { return nil }
func (r *myRepo) Count() int   { return 0 }
func (r *myRepo) Exists() bool { return false }

// UseCustomRepo demonstrates usage of custom repository.
// These calls should NOT trigger any warnings because myRepo is not from SQLBoiler.
func UseCustomRepo() {
	repo := &myRepo{}

	// None of these should trigger warnings - they're not SQLBoiler
	_ = repo.All()    // OK - not SQLBoiler
	_ = repo.One()    // OK - not SQLBoiler
	_ = repo.Count()  // OK - not SQLBoiler
	_ = repo.Exists() // OK - not SQLBoiler
}

// AnotherService with similar method names
type AnotherService struct{}

func (s *AnotherService) All() []string                            { return nil }
func (s *AnotherService) Select(columns ...string) *AnotherService { return s }
func (s *AnotherService) Find() *Item                              { return nil }
func (s *AnotherService) First() *Item                             { return nil }

func UseAnotherService() {
	svc := &AnotherService{}

	// These should NOT trigger warnings - they're not GORM/SQLBoiler/etc
	_ = svc.All()                // OK - not SQLBoiler
	_ = svc.Select("id", "name") // OK - not Squirrel/GORM
	_ = svc.Find()               // OK - not GORM
	_ = svc.First()              // OK - not GORM
}

// DB-like struct but not from any SQL library
type FakeDB struct{}

func (d *FakeDB) Query(sql string)               {}
func (d *FakeDB) Select(cols ...string) *FakeDB  { return d }
func (d *FakeDB) Columns(cols ...string) *FakeDB { return d }
func (d *FakeDB) Column(col string) *FakeDB      { return d }

func UseFakeDB() {
	db := &FakeDB{}

	// These should NOT trigger warnings
	db.Query("SELECT id, name FROM users") // OK - not real DB
	db.Select("*")                         // OK - not real GORM/Squirrel
}

// FakeBuilder mimics SQL builder patterns but is NOT from any SQL library
type FakeBuilder struct{}

func (b *FakeBuilder) Select(cols ...string) *FakeBuilder  { return b }
func (b *FakeBuilder) Columns(cols ...string) *FakeBuilder { return b }
func (b *FakeBuilder) From(table string) *FakeBuilder      { return b }

func UseFakeBuilder() {
	// This pattern should NOT trigger warnings - it's not a real SQL builder
	builder := &FakeBuilder{}
	builder.Select().Columns("*") // OK - not real Squirrel

	// Chain pattern
	builder.Select().Columns("id", "name").From("users") // OK - not real Squirrel

	// Empty select pattern
	b := builder.Select() // OK - not real SQL builder
	_ = b
}
