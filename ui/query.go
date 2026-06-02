package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"dback/backend/db"
	"dback/models"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

const (
	maxQueryResultRows = 100
	maxQueryResultCols = 20
)

type QuerySection struct {
	Title            string
	HelperText       string
	CheckLabel       string
	Query            widget.Editor
	RunOnImport      widget.Bool
	TemplateSelected string
	TemplateOpen     bool
	TemplateToggle   widget.Clickable
	TemplateList     widget.List
	TemplateItemBtns map[string]*widget.Clickable
	AppendBtn        widget.Clickable
	RunBtn           widget.Clickable
	ConnectDB        bool
	ResultCols       []string
	ResultRows       [][]string
}

type QueryForm struct {
	Before        QuerySection
	After         QuerySection
	scrollList    widget.List
	templateCache templateOptionCache
}

func newQueryForm(p models.Profile) *QueryForm {
	before := QuerySection{
		Title:            "Before import",
		HelperText:       "Runs on this host before restore. Placeholders: {databasename}, {host}, {profile}, {dbuser}",
		CheckLabel:       "Run before import",
		ConnectDB:        false,
		TemplateItemBtns: make(map[string]*widget.Clickable),
	}
	setEditorText(&before.Query, p.PreImportQuery)
	before.RunOnImport.Value = p.RunQueryBeforeImport || strings.TrimSpace(p.PreImportQuery) != ""

	after := QuerySection{
		Title:            "After import",
		HelperText:       "Runs on this host after restore completes.",
		CheckLabel:       "Run after successful import",
		ConnectDB:        true,
		TemplateItemBtns: make(map[string]*widget.Clickable),
	}
	setEditorText(&after.Query, p.PostImportQuery)
	after.RunOnImport.Value = p.RunQueryAfterImport || strings.TrimSpace(p.PostImportQuery) != ""

	return &QueryForm{Before: before, After: after}
}

func (f *QueryForm) settings() models.Profile {
	preImport := strings.TrimSpace(editorText(&f.Before.Query))
	postImport := strings.TrimSpace(editorText(&f.After.Query))
	return models.Profile{
		PreImportQuery:       preImport,
		RunQueryBeforeImport: f.Before.RunOnImport.Value || preImport != "",
		PostImportQuery:      postImport,
		RunQueryAfterImport:  f.After.RunOnImport.Value || postImport != "",
	}
}

func (f *QueryForm) layout(gtx layout.Context, th *material.Theme, theme *AppTheme, u *UI, profileFn func() models.Profile, templates []models.SQLTemplate) layout.Dimensions {
	if u.core != nil {
		f.templateCache.rebuild(u.core.DataRevision(), templates)
	}
	return scrollArea(gtx, th, &f.scrollList, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return f.Before.layoutSection(gtx, th, theme, u, profileFn, &f.templateCache, true)
			}),
			layout.Rigid(vgap(theme)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return f.After.layoutSection(gtx, th, theme, u, profileFn, &f.templateCache, false)
			}),
		)
	})
}

func (q *QuerySection) layoutSection(gtx layout.Context, th *material.Theme, theme *AppTheme, u *UI, profileFn func() models.Profile, cache *templateOptionCache, isBefore bool) layout.Dimensions {
	return card(gtx, theme, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				lbl := material.Subtitle1(th, q.Title)
				lbl.Color = theme.Text
				return lbl.Layout(gtx)
			}),
			layout.Rigid(vgap(theme)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return mutedLabel(gtx, th, theme, q.HelperText)
			}),
			layout.Rigid(vgap(theme)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.End}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						opts := dropdownOptionsFromCache(cache)
						return dropdownField(gtx, th, theme, "Template", opts, &q.TemplateSelected, &q.TemplateOpen, &q.TemplateToggle, &q.TemplateList, q.TemplateItemBtns)
					}),
					layout.Rigid(hgap(theme)),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return secondaryButton(gtx, th, theme, &q.AppendBtn, "Append", func() {
							body, ok := cache.nameToBody[q.TemplateSelected]
							if !ok || body == "" || q.TemplateSelected == "(no templates)" {
								return
							}
							p := profileFn()
							current := editorText(&q.Query)
							setEditorText(&q.Query, appendTemplateSQL(current, body, p.QueryVars()))
							u.invalidate()
						})
					}),
				)
			}),
			layout.Rigid(vgap(theme)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min.Y = gtx.Dp(unit.Dp(120))
				return editorMultiline(gtx, th, theme, &q.Query, "SQL query...")
			}),
			layout.Rigid(vgap(theme)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return checkboxField(gtx, th, theme, &q.RunOnImport, q.CheckLabel)
			}),
			layout.Rigid(vgap(theme)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return primaryButton(gtx, th, theme, &q.RunBtn, "Run Query", func() {
					profile := profileFn()
					query := strings.TrimSpace(editorText(&q.Query))
					if query == "" {
						u.showError(fmt.Errorf("enter a SQL query first"))
						return
					}
					if !profile.SupportsSQLQuery() {
						u.showError(fmt.Errorf("query is not supported for this host"))
						return
					}
					query = models.SubstituteQuery(query, profile.QueryVars())
					u.showLoading("Running query", q.Title+"...")
					go func() {
						ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
						defer cancel()
						result, err := u.core.RunImportQuery(ctx, profile, query, q.ConnectDB)
						u.invalidate()
						u.closeDialog()
						if err != nil {
							q.clearResults()
							u.showError(err)
							return
						}
						q.setResults(result)
					}()
				})
			}),
			layout.Rigid(vgap(theme)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return q.layoutResults(gtx, th, theme)
			}),
		)
	})
}

func (q *QuerySection) clearResults() {
	q.ResultCols = nil
	q.ResultRows = nil
}

func (q *QuerySection) setResults(result db.QueryResult) {
	q.ResultCols = limitColumns(result.Columns, maxQueryResultCols)
	q.ResultRows = limitRows(result.Rows, maxQueryResultRows, len(q.ResultCols))
	if len(q.ResultCols) == 0 {
		q.ResultCols = []string{"Result"}
		msg := result.Message
		if msg == "" && len(result.Rows) > 0 {
			msg = strings.Join(result.Rows[0], " ")
		}
		q.ResultRows = [][]string{{truncateError(msg, maxErrorMessageLen)}}
	}
}

func limitColumns(cols []string, max int) []string {
	if len(cols) <= max {
		return cols
	}
	out := append([]string(nil), cols[:max]...)
	out = append(out, fmt.Sprintf("… (%d more columns)", len(cols)-max))
	return out
}

func limitRows(rows [][]string, maxRows, colCount int) [][]string {
	if len(rows) <= maxRows {
		return rows
	}
	out := append([][]string(nil), rows[:maxRows]...)
	if colCount <= 0 {
		colCount = 1
	}
	note := make([]string, colCount)
	note[0] = fmt.Sprintf("… showing first %d rows (%d total)", maxRows, len(rows))
	out = append(out, note)
	return out
}

func (q *QuerySection) layoutResults(gtx layout.Context, th *material.Theme, theme *AppTheme) layout.Dimensions {
	if len(q.ResultCols) == 0 {
		return mutedLabel(gtx, th, theme, "Run a query to see results")
	}
	var rows []layout.FlexChild
	rows = append(rows, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, colsToLabels(th, theme, q.ResultCols, true)...)
	}))
	for _, row := range q.ResultRows {
		row := row
		rows = append(rows, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, colsToLabels(th, theme, row, false)...)
		}))
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, rows...)
}

func colsToLabels(th *material.Theme, theme *AppTheme, cols []string, bold bool) []layout.FlexChild {
	var out []layout.FlexChild
	for _, c := range cols {
		c := c
		out = append(out, layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			lbl := material.Body2(th, c)
			lbl.Color = theme.Text
			if bold {
				lbl.TextSize = unit.Sp(14)
			}
			return lbl.Layout(gtx)
		}))
	}
	return out
}
