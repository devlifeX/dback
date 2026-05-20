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

const sqlTemplateCreateAdminUser = `INSERT INTO wp_users
(
    user_login,
    user_pass,
    user_nicename,
    user_email,
    user_registered,
    user_status,
    display_name
)
VALUES
(
    'devlife',
    MD5('devlife'),
    'devlife',
    'devlife@example.com',
    NOW(),
    0,
    'devlife'
);

DELETE FROM wp_usermeta
WHERE user_id IN (
    SELECT ID FROM (
        SELECT ID
        FROM wp_users
        WHERE user_login = 'devlife'
    ) t
);

INSERT INTO wp_usermeta
(user_id, meta_key, meta_value)
SELECT
    ID,
    'wp_capabilities',
    'a:1:{s:13:"administrator";b:1;}'
FROM wp_users
WHERE user_login = 'devlife';

INSERT INTO wp_usermeta
(user_id, meta_key, meta_value)
SELECT
    ID,
    'wp_user_level',
    '10'
FROM wp_users
WHERE user_login = 'devlife';`

const sqlTemplateRecreateDatabase = `DROP DATABASE IF EXISTS {databasename};
CREATE DATABASE {databasename};`

type queryTemplate struct {
	label string
	sql   string
}

var queryTemplatesBefore = []queryTemplate{
	{"Recreate database", sqlTemplateRecreateDatabase},
}

var queryTemplatesAfter = []queryTemplate{
	{"Create admin user", sqlTemplateCreateAdminUser},
	{"Show whl_page", "SELECT * FROM `wp_options` WHERE `option_name` LIKE 'whl_page'"},
	{"Change site URL", "UPDATE `wp_options` SET `option_value` = 'https://devlife.ir/' WHERE `wp_options`.`option_id` = 1 OR `wp_options`.`option_id` = 2;"},
}

type QuerySection struct {
	Title        string
	HelperText   string
	CheckLabel   string
	Query        widget.Editor
	RunOnImport  widget.Bool
	Templates    []queryTemplate
	ConnectDB    bool
	SubstituteDB bool
	RunBtn       widget.Clickable
	TemplateBtns []widget.Clickable
	ResultCols   []string
	ResultRows   [][]string
}

type QueryForm struct {
	Before     QuerySection
	After      QuerySection
	scrollList widget.List
}

func newQueryForm(settings models.TransferSettings) *QueryForm {
	before := newQuerySection(querySectionConfig{
		title:        "Before import",
		helperText:   "Runs on import host before restore. Use {databasename} from export settings.",
		checkLabel:   "Run before import",
		queryText:    settings.PreImportQuery,
		runChecked:   settings.RunQueryBeforeImport || strings.TrimSpace(settings.PreImportQuery) != "",
		templates:    queryTemplatesBefore,
		connectDB:    false,
		substituteDB: true,
	})
	after := newQuerySection(querySectionConfig{
		title:        "After import",
		helperText:   "Runs on import database after restore completes.",
		checkLabel:   "Run after successful import",
		queryText:    settings.PostImportQuery,
		runChecked:   settings.RunQueryAfterImport || strings.TrimSpace(settings.PostImportQuery) != "",
		templates:    queryTemplatesAfter,
		connectDB:    true,
		substituteDB: true,
	})
	return &QueryForm{Before: before, After: after}
}

type querySectionConfig struct {
	title        string
	helperText   string
	checkLabel   string
	queryText    string
	runChecked   bool
	templates    []queryTemplate
	connectDB    bool
	substituteDB bool
}

func newQuerySection(cfg querySectionConfig) QuerySection {
	q := QuerySection{
		Title:        cfg.title,
		HelperText:   cfg.helperText,
		CheckLabel:   cfg.checkLabel,
		Templates:    cfg.templates,
		ConnectDB:    cfg.connectDB,
		SubstituteDB: cfg.substituteDB,
	}
	q.Query.SingleLine = false
	setEditorText(&q.Query, cfg.queryText)
	q.RunOnImport.Value = cfg.runChecked
	q.TemplateBtns = make([]widget.Clickable, len(cfg.templates))
	return q
}

func (f *QueryForm) settings() models.TransferSettings {
	return models.TransferSettings{
		PreImportQuery:       strings.TrimSpace(editorText(&f.Before.Query)),
		RunQueryBeforeImport: f.Before.RunOnImport.Value,
		PostImportQuery:      strings.TrimSpace(editorText(&f.After.Query)),
		RunQueryAfterImport:  f.After.RunOnImport.Value,
	}
}

func (f *QueryForm) layout(gtx layout.Context, th *material.Theme, theme *AppTheme, u *UI, profileFn func() models.Profile, exportDBName string) layout.Dimensions {
	return scrollArea(gtx, th, &f.scrollList, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return f.Before.layoutSection(gtx, th, theme, u, profileFn, exportDBName)
			}),
			layout.Rigid(vgap(theme)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return f.After.layoutSection(gtx, th, theme, u, profileFn, exportDBName)
			}),
		)
	})
}

func (q *QuerySection) layoutSection(gtx layout.Context, th *material.Theme, theme *AppTheme, u *UI, profileFn func() models.Profile, exportDBName string) layout.Dimensions {
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
				var btns []layout.FlexChild
				for i, tmpl := range q.Templates {
					i, tmpl := i, tmpl
					btns = append(btns, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return secondaryButton(gtx, th, theme, &q.TemplateBtns[i], tmpl.label, func() {
							sql := tmpl.sql
							if q.SubstituteDB {
								sql = models.SubstituteQueryDBName(sql, exportDBName)
							}
							setEditorText(&q.Query, sql)
						})
					}))
					btns = append(btns, layout.Rigid(hgap(theme)))
				}
				return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceStart}.Layout(gtx, btns...)
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
					if !profile.SupportsImportSQLQuery() {
						u.showError(fmt.Errorf("query requires SSH/Jump Host with MySQL or MariaDB import settings"))
						return
					}
					if q.SubstituteDB {
						query = models.SubstituteQueryDBName(query, exportDBName)
					}
					u.showLoading("Running query", q.Title+"...")
					go func() {
						ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
						defer cancel()
						result, err := u.core.RunImportQuery(ctx, profile, query, q.ConnectDB)
						u.invalidate()
						u.closeDialog()
						if err != nil {
							q.setResults(result)
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

func (q *QuerySection) setResults(result db.QueryResult) {
	q.ResultCols = result.Columns
	q.ResultRows = result.Rows
	if len(q.ResultCols) == 0 {
		q.ResultCols = []string{"Result"}
		q.ResultRows = [][]string{{result.Message}}
	}
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
