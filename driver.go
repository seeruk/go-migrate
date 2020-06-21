package migrate

import (
	"context"
)

// Driver ...
// TODO: Can this be simplified, leaving more to each driver? It's quite heavily tied to SQL
// databases currently?
type Driver interface {
	Begin(ctx context.Context) error
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
	Lock(ctx context.Context) error
	Exec(ctx context.Context, command string) error
	CreateVersionsTable(ctx context.Context) error
	InsertVersion(ctx context.Context, version int) error
	Versions(ctx context.Context) ([]int, error)
	VersionTableExists(ctx context.Context) (bool, error)
}
