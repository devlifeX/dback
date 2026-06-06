package verify

import (
	"strings"
	"testing"

	"dback/backend/db"
)

func TestParseTableRowsResult(t *testing.T) {
	result := db.QueryResult{
		Columns: []string{"table_name", "table_rows"},
		Rows: [][]string{
			{"orders", "8731"},
			{"users", "1042"},
		},
	}
	counts, err := ParseTableRowsResult(result)
	if err != nil {
		t.Fatal(err)
	}
	if counts["orders"] != 8731 || counts["users"] != 1042 {
		t.Fatalf("unexpected counts: %#v", counts)
	}
}

func TestParseSingleCountFromBatchOutput(t *testing.T) {
	result := db.ParseMySQLBatchOutput("COUNT(*)\n42")
	n, err := parseSingleCount(result)
	if err != nil {
		t.Fatal(err)
	}
	if n != 42 {
		t.Fatalf("expected 42, got %d", n)
	}
}

func TestBuildFastTableRowsQuery(t *testing.T) {
	q := BuildFastTableRowsQuery("mydb", false)
	for _, part := range []string{"information_schema", "mydb", "BASE TABLE"} {
		if !strings.Contains(q, part) {
			t.Fatalf("query missing %q: %s", part, q)
		}
	}
	wp := BuildFastTableRowsQuery("", true)
	if !strings.Contains(wp, "DATABASE()") {
		t.Fatalf("unexpected wordpress query: %q", wp)
	}
}
