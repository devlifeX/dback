package app

import (
	"context"
	"fmt"
	"strings"

	"dback/models"
)

func shouldRunPreImportQuery(p models.Profile) bool {
	return p.SupportsSQLQuery() && strings.TrimSpace(p.PreImportQuery) != ""
}

func shouldRunPostImportQuery(p models.Profile) bool {
	return p.SupportsSQLQuery() &&
		strings.TrimSpace(p.PostImportQuery) != "" &&
		p.RunQueryAfterImport
}

func (a *App) runPreImportQueryPhase(
	ctx context.Context,
	operationID string,
	destination models.Profile,
	fileSize int64,
	progress ProgressFunc,
) error {
	if !shouldRunPreImportQuery(destination) {
		return nil
	}

	query := models.SubstituteQuery(destination.PreImportQuery, destination.QueryVars())
	if progress != nil {
		progress("Running pre-import query", 0, fileSize)
	}

	result, err := a.RunImportQuery(ctx, destination, query, false)
	if err != nil {
		a.logPhase(operationID, &destination, "Pre-import query", "query", "", 0, err.Error(), "Error", "Failed", err.Error())
		return fmt.Errorf("pre-import query failed: %w", err)
	}

	a.logPhase(operationID, &destination, "Pre-import query", "query", "", 0, formatQueryResultSummary(result), "Info", "Succeeded", "")
	return nil
}

func (a *App) runPostImportQueryPhase(
	ctx context.Context,
	operationID string,
	destination models.Profile,
) error {
	if !shouldRunPostImportQuery(destination) {
		return nil
	}

	query := models.SubstituteQuery(destination.PostImportQuery, destination.QueryVars())
	result, err := a.RunImportQuery(ctx, destination, query, true)
	if err != nil {
		a.logPhase(operationID, &destination, "Post-import query", "query", "", 0, err.Error(), "Error", "Failed", err.Error())
		return fmt.Errorf("post-import query failed: %w", err)
	}

	a.logPhase(operationID, &destination, "Post-import query", "query", "", 0, formatQueryResultSummary(result), "Info", "Succeeded", "")
	return nil
}
