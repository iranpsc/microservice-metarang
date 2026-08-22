package migrate

import (
	"reflect"
	"testing"
)

func TestSplitStatements(t *testing.T) {
	script := `
-- comment
ALTER TABLE t ADD COLUMN a INT;
INSERT INTO t (name) VALUES ('a;b');
/* block
   comment */
UPDATE t SET x = "c;d";
`
	got := SplitStatements(script)
	want := []string{
		"-- comment\nALTER TABLE t ADD COLUMN a INT",
		"INSERT INTO t (name) VALUES ('a;b')",
		"/* block\n   comment */\nUPDATE t SET x = \"c;d\"",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v\nwant %#v", got, want)
	}
}

func TestSplitStatementsCommentOnlyDropped(t *testing.T) {
	got := SplitStatements("-- only comment;\n# also\n")
	if len(got) != 0 {
		t.Fatalf("got %#v", got)
	}
}

func TestSplitStatementsEscapedQuote(t *testing.T) {
	got := SplitStatements(`INSERT INTO t VALUES ('it''s'); INSERT INTO t VALUES ('x');`)
	if len(got) != 2 {
		t.Fatalf("got %#v", got)
	}
}
