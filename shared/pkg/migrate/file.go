package migrate

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	markerUp   = "-- migrate:up"
	markerDown = "-- migrate:down"
)

// fileNamePattern matches Laravel-style names:
// YYYY_MM_DD_description.sql or YYYY_MM_DD_HHMMSS_description.sql
var fileNamePattern = regexp.MustCompile(`^\d{4}_\d{2}_\d{2}(?:_\d{6})?_.+\.sql$`)

// Migration is one SQL file on disk.
type Migration struct {
	// Name is the value stored in migrations.migration (filename without .sql).
	Name string
	Path string
	Up   string
	Down string
}

// LoadDir reads and sorts SQL migrations from dir.
func LoadDir(dir string) ([]Migration, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read migrations dir %s: %w", dir, err)
	}

	var migrations []Migration
	for _, entry := range entries {
		if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		if !strings.HasSuffix(strings.ToLower(entry.Name()), ".sql") {
			continue
		}
		if !fileNamePattern.MatchString(entry.Name()) {
			return nil, fmt.Errorf("invalid migration filename %q (want YYYY_MM_DD[_HHMMSS]_description.sql)", entry.Name())
		}

		path := filepath.Join(dir, entry.Name())
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}

		m, err := parseMigration(entry.Name(), path, string(raw))
		if err != nil {
			return nil, err
		}
		migrations = append(migrations, m)
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Name < migrations[j].Name
	})
	return migrations, nil
}

func parseMigration(fileName, path, raw string) (Migration, error) {
	name := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	up, down := splitUpDown(raw)
	if strings.TrimSpace(up) == "" {
		return Migration{}, fmt.Errorf("migration %s has empty up SQL", fileName)
	}
	return Migration{Name: name, Path: path, Up: up, Down: down}, nil
}

func splitUpDown(raw string) (up, down string) {
	lower := strings.ToLower(raw)
	upIdx := indexMarker(lower, markerUp)
	downIdx := indexMarker(lower, markerDown)

	switch {
	case upIdx >= 0 && downIdx >= 0:
		if downIdx < upIdx {
			// Unusual but valid: down section first.
			down = stripMarkerLine(raw[downIdx:upIdx], markerDown)
			up = stripMarkerLine(raw[upIdx:], markerUp)
			return strings.TrimSpace(up), strings.TrimSpace(down)
		}
		up = stripMarkerLine(raw[upIdx:downIdx], markerUp)
		down = stripMarkerLine(raw[downIdx:], markerDown)
		return strings.TrimSpace(up), strings.TrimSpace(down)
	case upIdx >= 0:
		return strings.TrimSpace(stripMarkerLine(raw[upIdx:], markerUp)), ""
	case downIdx >= 0:
		return strings.TrimSpace(raw[:downIdx]), strings.TrimSpace(stripMarkerLine(raw[downIdx:], markerDown))
	default:
		return strings.TrimSpace(raw), ""
	}
}

func indexMarker(lower, marker string) int {
	return strings.Index(lower, strings.ToLower(marker))
}

func stripMarkerLine(section, marker string) string {
	trimmed := strings.TrimLeft(section, " \t\r")
	lower := strings.ToLower(trimmed)
	m := strings.ToLower(marker)
	if strings.HasPrefix(lower, m) {
		rest := trimmed[len(marker):]
		if i := strings.IndexAny(rest, "\r\n"); i >= 0 {
			return rest[i+1:]
		}
		return ""
	}
	return section
}
