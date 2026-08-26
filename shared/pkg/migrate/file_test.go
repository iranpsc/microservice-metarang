package migrate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseMigrationMarkers(t *testing.T) {
	raw := `-- header comment
-- migrate:up
ALTER TABLE t ADD COLUMN a INT;
-- migrate:down
ALTER TABLE t DROP COLUMN a;
`
	m, err := parseMigration("2026_08_21_add_a.sql", "x", raw)
	if err != nil {
		t.Fatal(err)
	}
	if m.Name != "2026_08_21_add_a" {
		t.Fatalf("name = %q", m.Name)
	}
	if !strings.Contains(m.Up, "ADD COLUMN") {
		t.Fatalf("up = %q", m.Up)
	}
	if !strings.Contains(m.Down, "DROP COLUMN") {
		t.Fatalf("down = %q", m.Down)
	}
}

func TestParseMigrationWholeFileIsUp(t *testing.T) {
	m, err := parseMigration("2026_07_18_add_user_id_to_buildings.sql", "x", "ALTER TABLE buildings ADD COLUMN user_id BIGINT;")
	if err != nil {
		t.Fatal(err)
	}
	if m.Down != "" {
		t.Fatalf("expected empty down, got %q", m.Down)
	}
	if !strings.Contains(m.Up, "user_id") {
		t.Fatalf("up = %q", m.Up)
	}
}

func TestParseMigrationEmptyUp(t *testing.T) {
	_, err := parseMigration("2026_08_21_empty.sql", "x", "-- migrate:up\n\n-- migrate:down\nSELECT 1;")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLoadDirSortAndReject(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, dir, "2026_07_18_b.sql", "-- migrate:up\nSELECT 2;\n-- migrate:down\nSELECT 2;")
	mustWrite(t, dir, "2026_07_17_a.sql", "-- migrate:up\nSELECT 1;\n-- migrate:down\nSELECT 1;")
	mustWrite(t, dir, "readme.txt", "ignore")

	files, err := LoadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 || files[0].Name != "2026_07_17_a" || files[1].Name != "2026_07_18_b" {
		t.Fatalf("unexpected files: %+v", files)
	}

	mustWrite(t, dir, "not-a-migration.sql", "SELECT 1;")
	if _, err := LoadDir(dir); err == nil {
		t.Fatal("expected invalid filename error")
	}
}

func TestLoadDirTimedName(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, dir, "2026_08_21_104530_wallet.sql", "-- migrate:up\nSELECT 1;")
	files, err := LoadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0].Name != "2026_08_21_104530_wallet" {
		t.Fatalf("got %+v", files)
	}
}

func TestWriteStub(t *testing.T) {
	dir := t.TempDir()
	path, err := WriteStub(dir, "Add Wallet Login", mustParseTime(t, "2026-08-21T10:45:30Z"))
	if err != nil {
		t.Fatal(err)
	}
	base := filepath.Base(path)
	if base != "2026_08_21_104530_add_wallet_login.sql" {
		t.Fatalf("filename = %s", base)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "-- migrate:up") || !strings.Contains(string(raw), "-- migrate:down") {
		t.Fatalf("stub = %s", raw)
	}
}

func mustWrite(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
