package db

import (
	"strings"
)

type QueryResult struct {
	Columns []string
	Rows    [][]string
	Message string
}

func ParseMySQLBatchOutput(out string) QueryResult {
	out = strings.TrimSpace(out)
	if out == "" {
		return QueryResult{Columns: []string{"Result"}, Rows: [][]string{{"(empty)"}}}
	}

	blocks := splitBlocks(out)
	var lastTabular QueryResult
	foundTabular := false

	for _, block := range blocks {
		result := parseTSVBlock(block)
		if len(result.Columns) > 0 {
			lastTabular = result
			foundTabular = true
		}
	}

	if foundTabular {
		return lastTabular
	}

	return QueryResult{
		Columns: []string{"Result"},
		Rows:    [][]string{{out}},
	}
}

func splitBlocks(out string) []string {
	lines := strings.Split(out, "\n")
	var blocks []string
	var current []string
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			if len(current) > 0 {
				blocks = append(blocks, strings.Join(current, "\n"))
				current = nil
			}
			continue
		}
		current = append(current, line)
	}
	if len(current) > 0 {
		blocks = append(blocks, strings.Join(current, "\n"))
	}
	if len(blocks) == 0 {
		blocks = []string{out}
	}
	return blocks
}

func parseTSVBlock(block string) QueryResult {
	lines := strings.Split(block, "\n")
	var rows [][]string
	for _, line := range lines {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		if !strings.Contains(line, "\t") {
			continue
		}
		rows = append(rows, strings.Split(line, "\t"))
	}
	if len(rows) == 0 {
		return QueryResult{}
	}
	return QueryResult{
		Columns: rows[0],
		Rows:    rows[1:],
	}
}
