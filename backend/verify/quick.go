package verify

import (
	"fmt"
	"os"
)

// QuickCheckResult holds the outcome of a SHA256 integrity check.
type QuickCheckResult struct {
	Passed   bool
	Actual   string
	Expected string
}

// QuickCheck recalculates SHA256 of filePath and compares it to expectedSHA.
func QuickCheck(filePath, expectedSHA string) (QuickCheckResult, error) {
	if expectedSHA == "" {
		return QuickCheckResult{}, fmt.Errorf("no stored checksum for this backup; create a new backup to enable verify")
	}
	if _, err := os.Stat(filePath); err != nil {
		if os.IsNotExist(err) {
			return QuickCheckResult{}, fmt.Errorf("backup file not found locally; download it from S3 sync first")
		}
		return QuickCheckResult{}, err
	}
	actual, err := ChecksumFile(filePath)
	if err != nil {
		return QuickCheckResult{}, err
	}
	return QuickCheckResult{
		Passed:   actual == expectedSHA,
		Actual:   actual,
		Expected: expectedSHA,
	}, nil
}
