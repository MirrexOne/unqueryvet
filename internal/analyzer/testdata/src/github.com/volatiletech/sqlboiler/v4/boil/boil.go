// Package boil is a stub of github.com/volatiletech/sqlboiler/v4/boil
package boil

import "context"

// ContextExecutor is the interface for executing queries with context.
type ContextExecutor interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (Rows, error)
}

// Result is a stub for sql.Result
type Result interface{}

// Rows is a stub for sql.Rows
type Rows interface{}

// Executor is the basic executor interface.
type Executor interface{}
