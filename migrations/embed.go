// Package migrations embeds the versioned SQL migration files so they can be run
// via golang-migrate's iofs source without shipping the raw .sql files alongside
// the binary (PLAN §2.1, §7.3).
package migrations

import "embed"

// FS holds every *.sql migration file.
//
//go:embed *.sql
var FS embed.FS
