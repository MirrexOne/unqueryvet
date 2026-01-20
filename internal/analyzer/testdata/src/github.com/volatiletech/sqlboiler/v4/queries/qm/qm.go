// Package qm is a stub of github.com/volatiletech/sqlboiler/v4/queries/qm
// for testing purposes. It provides the same types/functions that the checker looks for.
package qm

// QueryMod is a query modifier interface (stub).
type QueryMod interface {
	Apply()
}

type queryMod struct{}

func (q queryMod) Apply() {}

// Select creates a SELECT query mod.
func Select(cols ...string) QueryMod {
	return queryMod{}
}

// Where creates a WHERE query mod.
func Where(clause string, args ...interface{}) QueryMod {
	return queryMod{}
}

// OrderBy creates an ORDER BY query mod.
func OrderBy(clause string) QueryMod {
	return queryMod{}
}

// Limit creates a LIMIT query mod.
func Limit(count int) QueryMod {
	return queryMod{}
}
