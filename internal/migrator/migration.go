package migrator

import (
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Migration represents a single database migration.
type Migration struct {
	Version  int    // numeric prefix, e.g. 000013 → 13
	Name     string // descriptive name, e.g. "add_last_activity_at"
	Filename string // unique identifier, e.g. "000013_add_last_activity_at"
	FilePath string // full path to the .up.sql file
	Content  string // cached .up.sql content
}

// parseFilename parses a filename like "000013_add_last_activity_at.up.sql"
// and returns the Migration metadata. Returns (zero, false) for non-.up.sql files.
func parseFilename(path string) (Migration, bool, error) {
	base := filepath.Base(path)
	if !strings.HasSuffix(base, ".up.sql") {
		return Migration{}, false, nil
	}

	stem := strings.TrimSuffix(base, ".up.sql")

	// Split into version prefix and name: "000013_add_last_activity_at"
	underscoreIdx := strings.Index(stem, "_")
	if underscoreIdx < 1 {
		return Migration{}, false, fmt.Errorf("invalid migration filename %q: missing version separator", base)
	}

	versionStr := stem[:underscoreIdx]
	version, err := strconv.Atoi(versionStr)
	if err != nil {
		return Migration{}, false, fmt.Errorf("invalid migration version in %q: %w", base, err)
	}

	name := stem[underscoreIdx+1:]
	if name == "" {
		return Migration{}, false, fmt.Errorf("invalid migration filename %q: empty name", base)
	}

	return Migration{
		Version:  version,
		Name:     name,
		Filename: stem,
		FilePath: path,
	}, true, nil
}

// discoverMigrations scans a directory for all .up.sql files and returns them
// sorted by Version (ascending), then by Name (alphabetical).
func discoverMigrations(dir string) ([]Migration, error) {
	pattern := filepath.Join(dir, "*.up.sql")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("glob migrations in %q: %w", dir, err)
	}

	if len(matches) == 0 {
		return nil, nil
	}

	var migrations []Migration
	for _, m := range matches {
		mig, ok, err := parseFilename(m)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		migrations = append(migrations, mig)
	}

	sort.Slice(migrations, func(i, j int) bool {
		if migrations[i].Version != migrations[j].Version {
			return migrations[i].Version < migrations[j].Version
		}
		return migrations[i].Name < migrations[j].Name
	})

	return migrations, nil
}
