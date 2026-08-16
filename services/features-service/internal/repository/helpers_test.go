package repository

import "testing"

func TestParseCoordStringAndIDsCSV(t *testing.T) {
	if parseCoordString(" 1.5 ") != 1.5 {
		t.Fatalf("parseCoordString")
	}
	if parseCoordString("nope") != 0 {
		t.Fatalf("invalid parse")
	}
	if uint64IDsToCSV(nil) != "" {
		t.Fatal("empty csv")
	}
	if uint64IDsToCSV([]uint64{1, 2}) != "1,2" {
		t.Fatal("csv")
	}
	if placeholders(0) != "" {
		t.Fatal("placeholders 0")
	}
	if placeholders(3) != "?,?,?" {
		t.Fatal("placeholders 3")
	}
}
