// Package db carries the SQL migrations inside the binary, so a fresh
// database gets its schema the first time the server starts, wherever it
// runs. Files are applied in name order; each must be safe to re-run.
package db

import "embed"

//go:embed migrations/*.sql
var Migrations embed.FS
