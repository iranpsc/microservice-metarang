package migrate

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

const lockName = "metarang_migrations"

// Locker serializes migrate runs (MySQL GET_LOCK).
type Locker interface {
	Lock(ctx context.Context) (unlock func() error, err error)
}

// NoopLocker never locks. Used in tests.
type NoopLocker struct{}

func (NoopLocker) Lock(context.Context) (func() error, error) {
	return func() error { return nil }, nil
}

// MySQLLocker holds GET_LOCK on a dedicated connection for the process lifetime of a command.
type MySQLLocker struct {
	DB      *sql.DB
	Timeout time.Duration
}

func (l *MySQLLocker) Lock(ctx context.Context) (func() error, error) {
	timeout := l.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	conn, err := l.DB.Conn(ctx)
	if err != nil {
		return nil, fmt.Errorf("migration lock connection: %w", err)
	}

	var got sql.NullInt64
	err = conn.QueryRowContext(ctx, "SELECT GET_LOCK(?, ?)", lockName, int(timeout.Seconds())).Scan(&got)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("GET_LOCK: %w", err)
	}
	if !got.Valid || got.Int64 != 1 {
		_ = conn.Close()
		return nil, fmt.Errorf("could not acquire migration lock %q within %s", lockName, timeout)
	}

	unlock := func() error {
		defer func() { _ = conn.Close() }()
		_, err := conn.ExecContext(context.Background(), "SELECT RELEASE_LOCK(?)", lockName)
		return err
	}
	return unlock, nil
}
