package verify

import (
	"dback/models"
	"sort"
)

// ReportSummary holds aggregate deep-verify table statistics.
type ReportSummary struct {
	Total      int
	Matched    int
	Mismatched int
}

// PartitionReport splits a table report into summary stats and matched/mismatched rows.
func PartitionReport(report []models.TableVerifyResult) (summary ReportSummary, mismatched, matched []models.TableVerifyResult) {
	summary.Total = len(report)
	for _, row := range report {
		if row.Match {
			matched = append(matched, row)
			summary.Matched++
		} else {
			mismatched = append(mismatched, row)
			summary.Mismatched++
		}
	}
	return summary, mismatched, matched
}

// BuildTableReport compares actual row counts against a stored fingerprint.
func BuildTableReport(fp *models.BackupFingerprint, actual map[string]int64) ([]models.TableVerifyResult, bool) {
	if fp == nil {
		return nil, false
	}
	tables := make([]string, 0, len(fp.Tables))
	for name := range fp.Tables {
		tables = append(tables, name)
	}
	sort.Strings(tables)

	allPassed := true
	report := make([]models.TableVerifyResult, 0, len(tables))
	for _, table := range tables {
		expected := fp.Tables[table].Rows
		got := actual[table]
		match := got == expected
		if !match {
			allPassed = false
		}
		report = append(report, models.TableVerifyResult{
			Table:    table,
			Expected: expected,
			Actual:   got,
			Match:    match,
		})
	}
	for name, count := range actual {
		if _, ok := fp.Tables[name]; ok {
			continue
		}
		allPassed = false
		report = append(report, models.TableVerifyResult{
			Table:    name,
			Expected: 0,
			Actual:   count,
			Match:    false,
		})
	}
	sort.Slice(report, func(i, j int) bool {
		return report[i].Table < report[j].Table
	})
	return report, allPassed
}
