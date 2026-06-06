package verify

import (
	"strings"
	"testing"

	"dback/models"
)

func TestBuildTableReport(t *testing.T) {
	fp := &models.BackupFingerprint{
		Tables: map[string]models.FingerprintTable{
			"users":    {Rows: 1042},
			"orders":   {Rows: 8731},
			"products": {Rows: 234},
		},
	}
	actual := map[string]int64{
		"users":    1042,
		"orders":   8731,
		"products": 230,
	}
	report, passed := BuildTableReport(fp, actual)
	if passed {
		t.Fatal("expected failure due to products mismatch")
	}
	if len(report) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(report))
	}
}

func TestPartitionReport(t *testing.T) {
	report := []models.TableVerifyResult{
		{Table: "a", Match: true},
		{Table: "b", Match: false},
		{Table: "c", Match: true},
	}
	summary, mismatched, matched := PartitionReport(report)
	if summary.Total != 3 || summary.Matched != 2 || summary.Mismatched != 1 {
		t.Fatalf("unexpected summary: %#v", summary)
	}
	if len(mismatched) != 1 || len(matched) != 2 {
		t.Fatalf("unexpected partitions: mismatched=%d matched=%d", len(mismatched), len(matched))
	}
}

func TestTempDBName(t *testing.T) {
	name := TempDBName()
	if name == "" || !strings.Contains(name, "dback_verify_") {
		t.Fatalf("unexpected temp db name: %q", name)
	}
}
