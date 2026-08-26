package migrate

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

func testMigrator(t *testing.T) (*Migrator, *memoryStore, *scriptRecorder, *bytes.Buffer) {
	t.Helper()
	store := newMemoryStore()
	rec := &scriptRecorder{}
	out := &bytes.Buffer{}
	now := time.Date(2026, 8, 21, 10, 0, 0, 0, time.UTC)
	m := &Migrator{
		Store:  store,
		Exec:   rec.exec,
		Locker: NoopLocker{},
		Out:    out,
		Now: func() time.Time {
			n := now
			now = now.Add(time.Millisecond)
			return n
		},
	}
	return m, store, rec, out
}

type scriptRecorder struct {
	stmts []string
	fail  string
}

func (r *scriptRecorder) exec(_ context.Context, statement string) error {
	if r.fail != "" && strings.Contains(statement, r.fail) {
		return errors.New("exec failed")
	}
	r.stmts = append(r.stmts, statement)
	return nil
}

func sampleFiles() []Migration {
	return []Migration{
		{Name: "2026_01_01_create_a", Up: "CREATE TABLE a (id INT);", Down: "DROP TABLE a;"},
		{Name: "2026_01_02_create_b", Up: "CREATE TABLE b (id INT);", Down: "DROP TABLE b;"},
		{Name: "2026_01_03_create_c", Up: "CREATE TABLE c (id INT);", Down: "DROP TABLE c;"},
	}
}

func TestRunBatchesAndPending(t *testing.T) {
	ctx := context.Background()
	m, store, rec, _ := testMigrator(t)
	files := sampleFiles()

	if err := m.Run(ctx, files[:2], 0, false); err != nil {
		t.Fatal(err)
	}
	if err := m.Run(ctx, files, 0, false); err != nil {
		t.Fatal(err)
	}

	applied, err := store.Applied(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(applied) != 3 {
		t.Fatalf("applied = %+v", applied)
	}
	if applied[0].Batch != 1 || applied[1].Batch != 1 || applied[2].Batch != 2 {
		t.Fatalf("batches = %+v", applied)
	}
	if len(rec.stmts) != 3 {
		t.Fatalf("exec count = %d", len(rec.stmts))
	}
}

func TestRunNothingToMigrate(t *testing.T) {
	ctx := context.Background()
	m, _, _, out := testMigrator(t)
	files := sampleFiles()[:1]
	if err := m.Run(ctx, files, 0, false); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := m.Run(ctx, files, 0, false); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Nothing to migrate.") {
		t.Fatalf("out = %q", out.String())
	}
}

func TestRunStepAndPretend(t *testing.T) {
	ctx := context.Background()
	m, store, rec, out := testMigrator(t)
	if err := m.Run(ctx, sampleFiles(), 1, true); err != nil {
		t.Fatal(err)
	}
	if len(rec.stmts) != 0 {
		t.Fatalf("pretend executed %+v", rec.stmts)
	}
	applied, _ := store.Applied(ctx)
	if len(applied) != 0 {
		t.Fatalf("pretend recorded %+v", applied)
	}
	if !strings.Contains(out.String(), "CREATE TABLE a") {
		t.Fatalf("out = %q", out.String())
	}

	if err := m.Run(ctx, sampleFiles(), 1, false); err != nil {
		t.Fatal(err)
	}
	applied, _ = store.Applied(ctx)
	if len(applied) != 1 || applied[0].Name != "2026_01_01_create_a" {
		t.Fatalf("applied = %+v", applied)
	}
}

func TestRunDoesNotRecordOnExecError(t *testing.T) {
	ctx := context.Background()
	m, store, rec, _ := testMigrator(t)
	rec.fail = "CREATE TABLE b"
	err := m.Run(ctx, sampleFiles(), 0, false)
	if err == nil {
		t.Fatal("expected error")
	}
	applied, _ := store.Applied(ctx)
	if len(applied) != 1 {
		t.Fatalf("should keep first success only, got %+v", applied)
	}
}

func TestRollbackLastBatch(t *testing.T) {
	ctx := context.Background()
	m, store, rec, _ := testMigrator(t)
	if err := m.Run(ctx, sampleFiles()[:2], 0, false); err != nil {
		t.Fatal(err)
	}
	if err := m.Run(ctx, sampleFiles(), 0, false); err != nil {
		t.Fatal(err)
	}
	rec.stmts = nil
	if err := m.Rollback(ctx, sampleFiles(), 0, false); err != nil {
		t.Fatal(err)
	}
	applied, _ := store.Applied(ctx)
	if len(applied) != 2 {
		t.Fatalf("applied = %+v", applied)
	}
	if len(rec.stmts) != 1 || rec.stmts[0] != "DROP TABLE c" {
		t.Fatalf("down stmts = %+v", rec.stmts)
	}
}

func TestRollbackStepReverseOrder(t *testing.T) {
	ctx := context.Background()
	m, store, rec, _ := testMigrator(t)
	if err := m.Run(ctx, sampleFiles(), 0, false); err != nil {
		t.Fatal(err)
	}
	rec.stmts = nil
	if err := m.Rollback(ctx, sampleFiles(), 2, false); err != nil {
		t.Fatal(err)
	}
	if fmt.Sprintf("%v", rec.stmts) != "[DROP TABLE c DROP TABLE b]" {
		t.Fatalf("order = %v", rec.stmts)
	}
	applied, _ := store.Applied(ctx)
	if len(applied) != 1 || applied[0].Name != "2026_01_01_create_a" {
		t.Fatalf("applied = %+v", applied)
	}
}

func TestRollbackMissingDown(t *testing.T) {
	ctx := context.Background()
	m, _, _, _ := testMigrator(t)
	files := []Migration{{Name: "2026_01_01_x", Up: "SELECT 1;", Down: ""}}
	if err := m.Run(ctx, files, 0, false); err != nil {
		t.Fatal(err)
	}
	err := m.Rollback(ctx, files, 0, false)
	if err == nil || !strings.Contains(err.Error(), "migrate:down") {
		t.Fatalf("err = %v", err)
	}
}

func TestRollbackMissingFile(t *testing.T) {
	ctx := context.Background()
	m, store, _, _ := testMigrator(t)
	_ = store.Record(ctx, "ghost", 1)
	err := m.Rollback(ctx, nil, 0, false)
	if err == nil || !strings.Contains(err.Error(), "SQL file not found") {
		t.Fatalf("err = %v", err)
	}
}

func TestResetAndRefresh(t *testing.T) {
	ctx := context.Background()
	m, store, rec, _ := testMigrator(t)
	if err := m.Run(ctx, sampleFiles(), 0, false); err != nil {
		t.Fatal(err)
	}
	if err := m.Reset(ctx, sampleFiles(), false); err != nil {
		t.Fatal(err)
	}
	applied, _ := store.Applied(ctx)
	if len(applied) != 0 {
		t.Fatalf("applied = %+v", applied)
	}

	rec.stmts = nil
	if err := m.Refresh(ctx, sampleFiles()[:2], false); err != nil {
		t.Fatal(err)
	}
	applied, _ = store.Applied(ctx)
	if len(applied) != 2 || applied[0].Batch != 1 {
		t.Fatalf("applied = %+v", applied)
	}
}

func TestBaselineSkipsSQL(t *testing.T) {
	ctx := context.Background()
	m, store, rec, _ := testMigrator(t)
	if err := m.Baseline(ctx, sampleFiles()); err != nil {
		t.Fatal(err)
	}
	if len(rec.stmts) != 0 {
		t.Fatalf("baseline executed %+v", rec.stmts)
	}
	applied, _ := store.Applied(ctx)
	if len(applied) != 3 || applied[0].Batch != 1 {
		t.Fatalf("applied = %+v", applied)
	}
	if err := m.Baseline(ctx, sampleFiles()); err != nil {
		t.Fatal(err)
	}
	applied, _ = store.Applied(ctx)
	if len(applied) != 3 {
		t.Fatalf("double baseline = %+v", applied)
	}
}

func TestStatus(t *testing.T) {
	ctx := context.Background()
	m, store, _, _ := testMigrator(t)
	_ = store.Record(ctx, "2026_01_01_create_a", 1)
	_ = store.Record(ctx, "deleted_old", 1)
	rows, err := m.Status(ctx, sampleFiles())
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 4 {
		t.Fatalf("rows = %+v", rows)
	}
	if rows[0].Pending || rows[1].Pending != true || rows[3].Name != "deleted_old" {
		t.Fatalf("rows = %+v", rows)
	}
}

func TestMemoryStoreForget(t *testing.T) {
	s := newMemoryStore()
	ctx := context.Background()
	_ = s.Record(ctx, "a", 1)
	_ = s.Record(ctx, "b", 1)
	_ = s.Forget(ctx, "a")
	got, _ := s.Applied(ctx)
	if len(got) != 1 || got[0].Name != "b" {
		t.Fatalf("got %+v", got)
	}
}

func mustParseTime(t *testing.T, v string) time.Time {
	t.Helper()
	ts, err := time.Parse(time.RFC3339, v)
	if err != nil {
		t.Fatal(err)
	}
	return ts
}
