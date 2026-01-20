// Package models is a stub of SQLBoiler generated models
package models

import (
	"context"

	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
)

// User represents a user model.
type User struct {
	ID    int
	Name  string
	Email string
}

// UserSlice is a slice of User.
type UserSlice []*User

// UserQuery is the query builder for users table.
type UserQuery struct {
	mods []qm.QueryMod
}

// Users creates a new query builder for the users table.
func Users(mods ...qm.QueryMod) *UserQuery {
	return &UserQuery{mods: mods}
}

// All executes the query and returns all results.
func (q *UserQuery) All(ctx context.Context, exec boil.ContextExecutor) (UserSlice, error) {
	return nil, nil
}

// One executes the query and returns one result.
func (q *UserQuery) One(ctx context.Context, exec boil.ContextExecutor) (*User, error) {
	return nil, nil
}

// Count executes the query and returns the count.
func (q *UserQuery) Count(ctx context.Context, exec boil.ContextExecutor) (int64, error) {
	return 0, nil
}

// Exists executes the query and returns whether a row exists.
func (q *UserQuery) Exists(ctx context.Context, exec boil.ContextExecutor) (bool, error) {
	return false, nil
}
