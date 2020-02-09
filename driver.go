package migrate

import (
	"context"
	"database/sql"
)

// Driver ...
type Driver interface {
	Begin(ctx context.Context) (*sql.Tx, error)
	Commit() error
	Rollback() error
	Lock(ctx context.Context) error
	CreateVersionsTable(ctx context.Context) error
	InsertVersion(ctx context.Context, version int) error
	Versions(ctx context.Context) ([]int, error)
	VersionTableExists(ctx context.Context) (bool, error)
}
