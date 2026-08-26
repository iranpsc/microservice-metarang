package migrate

import (
	"context"
	"database/sql"
	"fmt"
)

const createTableSQL = `
CREATE TABLE IF NOT EXISTS ` + "`migrations`" + ` (
  ` + "`id`" + ` int(10) unsigned NOT NULL AUTO_INCREMENT,
  ` + "`migration`" + ` varchar(191) NOT NULL,
  ` + "`batch`" + ` int(11) NOT NULL,
  PRIMARY KEY (` + "`id`" + `)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`

// Applied is one row from the Laravel-compatible migrations table.
type Applied struct {
	Name  string
	Batch int
}

// Store persists which migrations have run.
type Store interface {
	Ensure(ctx context.Context) error
	Applied(ctx context.Context) ([]Applied, error)
	Record(ctx context.Context, name string, batch int) error
	Forget(ctx context.Context, name string) error
}

// SQLStore uses Laravel's `migrations` table (id, migration, batch).
type SQLStore struct {
	DB *sql.DB
}

func (s *SQLStore) Ensure(ctx context.Context) error {
	if _, err := s.DB.ExecContext(ctx, createTableSQL); err != nil {
		return fmt.Errorf("create migrations table: %w", err)
	}
	return nil
}

func SQLExec(db *sql.DB) ExecFunc {
	return func(ctx context.Context, statement string) error {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			preview := statement
			if len(preview) > 120 {
				preview = preview[:120] + "..."
			}
			return fmt.Errorf("exec %q: %w", preview, err)
		}
		return nil
	}
}

func (s *SQLStore) Applied(ctx context.Context) ([]Applied, error) {
	rows, err := s.DB.QueryContext(ctx, "SELECT migration, batch FROM migrations ORDER BY id ASC")
	if err != nil {
		return nil, fmt.Errorf("list applied migrations: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []Applied
	for rows.Next() {
		var a Applied
		if err := rows.Scan(&a.Name, &a.Batch); err != nil {
			return nil, fmt.Errorf("scan applied migration: %w", err)
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate applied migrations: %w", err)
	}
	return out, nil
}

func (s *SQLStore) Record(ctx context.Context, name string, batch int) error {
	_, err := s.DB.ExecContext(ctx, "INSERT INTO migrations (migration, batch) VALUES (?, ?)", name, batch)
	if err != nil {
		return fmt.Errorf("record migration %s: %w", name, err)
	}
	return nil
}

func (s *SQLStore) Forget(ctx context.Context, name string) error {
	_, err := s.DB.ExecContext(ctx, "DELETE FROM migrations WHERE migration = ?", name)
	if err != nil {
		return fmt.Errorf("forget migration %s: %w", name, err)
	}
	return nil
}

type memoryStore struct {
	rows []Applied
}

func newMemoryStore() *memoryStore {
	return &memoryStore{}
}

func (s *memoryStore) Ensure(context.Context) error { return nil }

func (s *memoryStore) Applied(context.Context) ([]Applied, error) {
	out := make([]Applied, len(s.rows))
	copy(out, s.rows)
	return out, nil
}

func (s *memoryStore) Record(_ context.Context, name string, batch int) error {
	s.rows = append(s.rows, Applied{Name: name, Batch: batch})
	return nil
}

func (s *memoryStore) Forget(_ context.Context, name string) error {
	filtered := s.rows[:0]
	for _, row := range s.rows {
		if row.Name != name {
			filtered = append(filtered, row)
		}
	}
	s.rows = filtered
	return nil
}
