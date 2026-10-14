package postgres

import (
	"context"
	"fmt"
	"io/fs"
	"sort"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Migrate applies every *.sql file under migrations/ in name order. The
// files are written to be idempotent (IF NOT EXISTS), so this runs on every
// start-up and simply does nothing once the schema is in place. A session
// advisory lock stops two instances migrating at the same moment.
func Migrate(ctx context.Context, pool *pgxpool.Pool, migrations fs.FS) error {
	names, err := fs.Glob(migrations, "migrations/*.sql")
	if err != nil {
		return err
	}
	sort.Strings(names)

	conn, err := pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	const lockKey = 7_201_031 // arbitrary, unique to this app
	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock($1)", lockKey); err != nil {
		return err
	}
	defer conn.Exec(context.WithoutCancel(ctx), "SELECT pg_advisory_unlock($1)", lockKey) //nolint:errcheck

	for _, name := range names {
		sql, err := fs.ReadFile(migrations, name)
		if err != nil {
			return err
		}
		if _, err := conn.Exec(ctx, string(sql)); err != nil {
			return fmt.Errorf("migration %s: %w", name, err)
		}
	}
	return nil
}
