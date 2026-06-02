package wordpress

import (
	"fmt"
	"strings"
)

type PreflightResult struct {
	Success   bool
	Summary   string
	Checks    []PreflightCheck
	Raw       map[string]interface{}
	DBVersion string
	Driver    string
}

type PreflightCheck struct {
	Name    string
	Status  string
	Details string
}

func parsePreflightResult(data map[string]interface{}) PreflightResult {
	result := PreflightResult{Raw: data}
	if v, ok := data["success"].(bool); ok {
		result.Success = v
	}
	if v, ok := data["db_version"].(string); ok {
		result.DBVersion = v
	}
	if v, ok := data["driver"].(string); ok {
		result.Driver = v
	}
	if checks, ok := data["checks"].([]interface{}); ok {
		for _, item := range checks {
			m, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			result.Checks = append(result.Checks, PreflightCheck{
				Name:    fmt.Sprint(m["name"]),
				Status:  fmt.Sprint(m["status"]),
				Details: fmt.Sprint(m["details"]),
			})
		}
	}
	result.Summary = preflightSummary(result)
	return result
}

func preflightSummary(r PreflightResult) string {
	var b strings.Builder
	if r.Driver != "" {
		b.WriteString("driver=" + r.Driver)
	}
	if r.DBVersion != "" {
		if b.Len() > 0 {
			b.WriteString(" | ")
		}
		b.WriteString("db=" + strings.TrimSpace(r.DBVersion))
	}
	for _, check := range r.Checks {
		if check.Status != "ok" {
			if b.Len() > 0 {
				b.WriteString(" | ")
			}
			b.WriteString(check.Name + ": " + check.Details)
		}
	}
	if b.Len() == 0 {
		b.WriteString("WordPress preflight passed")
	}
	return b.String()
}

func (r PreflightResult) FailureError() error {
	if r.Success {
		return nil
	}
	for _, check := range r.Checks {
		if check.Status == "fail" {
			return fmt.Errorf("preflight failed: %s — %s", check.Name, check.Details)
		}
	}
	return fmt.Errorf("wordpress preflight failed")
}
