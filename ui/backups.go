package ui

import (
	"context"
	"errors"
	"fmt"
	"image"
	"path/filepath"
	"strings"

	"dback/models"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

func (u *UI) openBackupFolder(filePath string) {
	if strings.TrimSpace(filePath) == "" {
		u.showInfo("Backup Files", "No backup file path available.")
		return
	}
	dir := filepath.Dir(filePath)
	if err := u.platform.OpenFolder(dir); err != nil {
		u.showError(fmt.Errorf("open backup folder: %w", err))
	}
}

func (u *UI) layoutBackups(gtx layout.Context, th *material.Theme) layout.Dimensions {
	theme := u.theme

	switch u.view {
	case ViewBackupDetail:
		return u.layoutBackupDetail(gtx, th)
	default:
		return u.layoutBackupsMain(gtx, th, theme)
	}
}

func (u *UI) layoutBackupsMain(gtx layout.Context, th *material.Theme, theme *AppTheme) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return pageHeader(gtx, th, theme, "Backups", nil)
		}),
		layout.Rigid(vgap(theme)),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return tabBar(gtx, th, theme,
				func(gtx layout.Context) layout.Dimensions {
					return tabButton(gtx, th, theme, &u.tabBackupFiles, "Backup Files", u.backupTab == 0, func() {
						u.backupTab = 0
						u.invalidate()
					})
				},
				func(gtx layout.Context) layout.Dimensions {
					return tabButton(gtx, th, theme, &u.tabBackupJobs, "Jobs", u.backupTab == 1, func() {
						u.backupTab = 1
						u.invalidate()
					})
				},
			)
		}),
		layout.Rigid(vgap(theme)),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			if u.backupTab == 1 {
				return u.layoutJobsTable(gtx, th, theme)
			}
			return u.layoutBackupFiles(gtx, th, theme)
		}),
	)
}

func dropdownOptionsFromCache(cache *templateOptionCache) DropdownOptions {
	if cache == nil {
		return DropdownOptions{Values: []string{"(no templates)"}, Labels: []string{"(no templates)"}}
	}
	return DropdownOptions{Values: cache.names, Labels: cache.labels}
}

func (u *UI) layoutBackupFiles(gtx layout.Context, th *material.Theme, theme *AppTheme) layout.Dimensions {
	u.backupHostSelect.Update(gtx)
	if u.backupHostSelect.Value == "" {
		u.backupHostSelect.Value = backupFilterAll
	}
	if u.backupHostFilter != u.backupHostSelect.Value {
		u.backupHostFilter = u.backupHostSelect.Value
		u.invalidateBackupCache()
	}
	u.backupCache.rebuild(u)
	records := u.backupCache.records
	hostValues := u.backupCache.hostValues
	hostLabels := u.backupCache.hostLabels

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return labeledEnumDropdownField(gtx, th, theme, &u.backupHostSelect, "Host filter", hostValues, hostLabels, &u.backupHostDropdown, u.invalidate, nil)
		}),
		layout.Rigid(vgap(theme)),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return scrollArea(gtx, th, &u.backupList, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Bottom: unit.Dp(24)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					var rows []layout.FlexChild
					rows = append(rows, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return backupTableHeader(gtx, th, theme)
					}))
					for _, record := range records {
						rec := record
						menu, ok := u.backupRowMenus[rec.ID]
						if !ok {
							menu = backupRowMenuWidgets{
								more:    new(widget.Clickable),
								import_: new(widget.Clickable),
								verify:  new(widget.Clickable),
								folder:  new(widget.Clickable),
							}
							u.backupRowMenus[rec.ID] = menu
						}
						selected := u.selectedBackupID == rec.ID
						rowMenu := menu
						rows = append(rows, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Inset{Bottom: theme.Gap}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return backupTableRow(gtx, th, theme, selected, []string{
									formatRelativeTime(rec.ExportDate),
									rec.ProfileName,
									rec.DatabaseName,
									rec.FileSize,
									u.backupQuickVerifyStatus(rec),
									u.backupDeepVerifyStatus(rec),
								}, func(gtx layout.Context) layout.Dimensions {
									return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return secondaryButton(gtx, th, theme, rowMenu.import_, "Import", func() {
												u.openBackupDetail(rec)
											})
										}),
										layout.Rigid(hgap(theme)),
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return u.layoutBackupRowMoreButton(gtx, th, theme, rec, rowMenu)
										}),
									)
								})
							})
						}))
					}
					if len(records) == 0 {
						rows = append(rows, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Inset{
								Top: unit.Dp(12), Bottom: unit.Dp(12),
								Left: unit.Dp(16), Right: unit.Dp(16),
							}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return mutedLabel(gtx, th, theme, "No backup files yet.")
							})
						}))
					}
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx, rows...)
				})
			})
		}),
	)
}

func (u *UI) layoutJobsTable(gtx layout.Context, th *material.Theme, theme *AppTheme) layout.Dimensions {
	jobs := u.currentJobs()
	return scrollArea(gtx, th, &u.jobsList, func(gtx layout.Context) layout.Dimensions {
		header := []string{"Type", "Profile", "Status", "Progress", ""}
		var rows []layout.FlexChild
		rows = append(rows, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return tableHeader(gtx, th, theme, header)
		}))
		for _, job := range jobs {
			job := job
			btn, ok := u.jobCancelBtns[job.ID]
			if !ok {
				btn = new(widget.Clickable)
				u.jobCancelBtns[job.ID] = btn
			}
			status := jobStatusLine(job)
			action := "Cancel"
			if job.Done {
				action = ""
			}
			cancelBtn := btn
			j := job
			rows = append(rows, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return tableRow(gtx, th, theme, false, []string{
							j.Kind,
							j.ProfileName,
							status,
							fmt.Sprintf("%.0f%%", j.Progress*100),
						}, action, cancelBtn, func() {
							if !j.Done && j.Cancel != nil {
								j.Cancel()
								u.updateJob(j.ID, "Canceling...", j.Progress, "")
							}
						})
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return progressBar(gtx, theme, job.Progress)
					}),
					layout.Rigid(vgap(theme)),
				)
			}))
		}
		if len(jobs) == 0 {
			rows = append(rows, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return mutedLabel(gtx, th, theme, "No active jobs.")
			}))
		}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, rows...)
	})
}

func (u *UI) defaultImportDestID(sourceProfileID string, importable []models.Profile) string {
	if destID := u.core.ImportDestForProfile(sourceProfileID); destID != "" {
		if _, ok := profileByID(importable, destID); ok {
			return destID
		}
	}
	if len(importable) > 0 {
		return importable[0].ID
	}
	return ""
}

func (u *UI) rememberImportDest(sourceProfileID, destProfileID string) {
	if sourceProfileID == "" || destProfileID == "" {
		return
	}
	_ = u.core.SetImportDestForProfile(sourceProfileID, destProfileID)
}

func (u *UI) openBackupDetail(record models.ExportRecord) {
	u.selectedBackup = &record
	u.view = ViewBackupDetail
	importable := importableProfiles(u.core.Profiles())
	u.destSelect.Value = u.defaultImportDestID(record.ProfileID, importable)
	u.invalidate()
}

func (u *UI) layoutBackupDetail(gtx layout.Context, th *material.Theme) layout.Dimensions {
	theme := u.theme
	record := u.selectedBackup
	if record == nil {
		u.view = ViewList
		return layout.Dimensions{}
	}

	profiles := importableProfiles(u.core.Profiles())
	values := make([]string, 0, len(profiles))
	labels := make([]string, 0, len(profiles))
	for _, p := range profiles {
		values = append(values, p.ID)
		label := p.Name
		if p.Group != "" {
			label += " (" + p.Group + ")"
		}
		labels = append(labels, label)
	}

	canImport := len(profiles) > 0

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return secondaryButton(gtx, th, theme, &u.backBtn, "← Back", func() {
						u.view = ViewList
						u.invalidate()
					})
				}),
				layout.Rigid(hgap(theme)),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return sectionTitle(gtx, th, theme, "Backup Detail")
				}),
			)
		}),
		layout.Rigid(vgap(theme)),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return card(gtx, theme, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						lbl := material.Body1(th, record.ProfileName)
						lbl.Color = theme.Text
						return lbl.Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						when := formatRelativeTime(record.ExportDate)
						if when == "" {
							return mutedLabel(gtx, th, theme, record.FileSize)
						}
						return mutedLabel(gtx, th, theme, when+" · "+record.FileSize)
					}),
					layout.Rigid(vgap(theme)),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return mutedLabel(gtx, th, theme, record.FilePath)
					}),
				)
			})
		}),
		layout.Rigid(vgap(theme)),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return card(gtx, theme, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return sectionLabel(gtx, th, theme, "Verification")
					}),
					layout.Rigid(vgap(theme)),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return mutedLabel(gtx, th, theme, "File integrity: "+u.backupDetailFileVerifyLabel(*record))
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return mutedLabel(gtx, th, theme, "Deep verify: "+u.backupDetailDeepVerifyLabel(*record))
					}),
				)
			})
		}),
		layout.Rigid(vgap(theme)),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if canImport {
				sourceProfileID := record.ProfileID
				return labeledEnumDropdownField(gtx, th, theme, &u.destSelect, "Destination Host", values, labels, &u.destHostDropdown, u.invalidate, func(destID string) {
					u.rememberImportDest(sourceProfileID, destID)
				})
			}
			return mutedLabel(gtx, th, theme, "No import destinations available. All hosts are protected from import, or no hosts exist.")
		}),
		layout.Rigid(vgap(theme)),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if !canImport {
						return disabledButton(gtx, th, theme, "Import to Selected Host")
					}
					return primaryButton(gtx, th, theme, &u.restoreBtn, "Import to Selected Host", func() {
						u.runRestore(*record)
					})
				}),
				layout.Rigid(hgap(theme)),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return secondaryButton(gtx, th, theme, &u.openBackupFolderBtn, "Open Folder", func() {
						u.openBackupFolder(record.FilePath)
					})
				}),
				layout.Rigid(hgap(theme)),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return secondaryButton(gtx, th, theme, &u.verifyBackupBtn, "Deep verify", func() {
						u.runDeepVerifyPrompt(*record)
					})
				}),
			)
		}),
	)
}

func (u *UI) runRestore(record models.ExportRecord) {
	profiles := importableProfiles(u.core.Profiles())
	var dest models.Profile
	for _, p := range profiles {
		if p.ID == u.destSelect.Value {
			dest = p
			break
		}
	}
	if dest.ID == "" {
		u.showError(fmt.Errorf("select an import destination host"))
		return
	}
	u.rememberImportDest(record.ProfileID, dest.ID)
	if !dest.AllowsImport() {
		u.showError(fmt.Errorf("host %q is protected from import", dest.Name))
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	job := u.addJob("Import", dest.Name, cancel)
	u.backupTab = 1
	u.view = ViewList
	u.openBackups()
	go func() {
		defer cancel()
		err := u.core.Restore(ctx, record, dest, func(message string, current int64, total int64) {
			progress := float64(0)
			if total > 0 {
				progress = float64(current) / float64(total)
			}
			u.updateJob(job.ID, message, progress, "")
		})
		if err != nil {
			if errors.Is(err, context.Canceled) {
				u.finishJob(job.ID, "Import canceled", nil)
				return
			}
			u.finishJob(job.ID, "Import failed", err)
			u.showError(err)
			return
		}
		u.finishJob(job.ID, "Import complete", nil)
	}()
}

type backupTableColumn struct {
	Label    string
	Weight   float32
	MinWidth unit.Dp
}

var backupTableColumns = []backupTableColumn{
	{Label: "When", Weight: 1.2},
	{Label: "Profile", Weight: 1.5},
	{Label: "Database", Weight: 1.2},
	{Label: "Size", Weight: 0.8},
	{Label: "Verify", Weight: 0.9},
	{Label: "Deep verify", Weight: 1.0},
	{Label: "Actions", MinWidth: unit.Dp(180)},
}

func backupTableHeaderInset() layout.Inset {
	return layout.Inset{
		Top: unit.Dp(4), Bottom: unit.Dp(8),
		Left: unit.Dp(16), Right: unit.Dp(16),
	}
}

func backupTableRowInset() layout.Inset {
	return layout.Inset{
		Top: unit.Dp(12), Bottom: unit.Dp(12),
		Left: unit.Dp(16), Right: unit.Dp(16),
	}
}

func backupTableHeader(gtx layout.Context, th *material.Theme, theme *AppTheme) layout.Dimensions {
	return backupTableHeaderInset().Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		var children []layout.FlexChild
		for _, col := range backupTableColumns {
			col := col
			if col.Weight > 0 {
				children = append(children, layout.Flexed(col.Weight, func(gtx layout.Context) layout.Dimensions {
					lbl := material.Caption(th, col.Label)
					lbl.Color = theme.TextMuted
					return lbl.Layout(gtx)
				}))
			} else {
				children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = gtx.Dp(col.MinWidth)
					lbl := material.Caption(th, col.Label)
					lbl.Color = theme.TextMuted
					return lbl.Layout(gtx)
				}))
			}
		}
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx, children...)
	})
}

func (u *UI) layoutBackupRowMoreButton(gtx layout.Context, th *material.Theme, theme *AppTheme, rec models.ExportRecord, menu backupRowMenuWidgets) layout.Dimensions {
	if menu.more.Clicked(gtx) {
		if u.backupMenuOpenID == rec.ID {
			u.backupMenuOpenID = ""
		} else {
			u.backupMenuOpenID = rec.ID
		}
		u.invalidate()
	}

	btnDims := renderButton(gtx, th, theme, menu.more, "⋮", btnSecondary, false)

	if u.backupMenuOpenID != rec.ID {
		return btnDims
	}

	verifyFG := theme.Success
	folderFG := theme.Link
	items := []menuPopupItem{
		{
			label: "Deep verify",
			fg:    &verifyFG,
			btn:   menu.verify,
			onClick: func() {
				u.backupMenuOpenID = ""
				u.runDeepVerifyPrompt(rec)
			},
		},
		{
			label: "Folder",
			fg:    &folderFG,
			btn:   menu.folder,
			onClick: func() {
				u.backupMenuOpenID = ""
				u.openBackupFolder(rec.FilePath)
			},
		},
	}

	if u.menuCloseArea.Clicked(gtx) {
		u.backupMenuOpenID = ""
		u.invalidate()
	}

	popupW := gtx.Dp(unit.Dp(160))
	popupOffX := btnDims.Size.X - popupW
	popupOffY := btnDims.Size.Y + gtx.Dp(unit.Dp(4))

	macro := op.Record(gtx.Ops)

	backdropStack := op.Offset(image.Pt(-9999, -9999)).Push(gtx.Ops)
	backdropClip := clip.Rect{Max: image.Pt(99999, 99999)}.Push(gtx.Ops)
	u.menuCloseArea.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Dimensions{Size: image.Pt(99999, 99999)}
	})
	backdropClip.Pop()
	backdropStack.Pop()

	popupStack := op.Offset(image.Pt(popupOffX, popupOffY)).Push(gtx.Ops)
	menuPopup(gtx, th, theme, items)
	popupStack.Pop()

	op.Defer(gtx.Ops, macro.Stop())

	return btnDims
}

func backupTableRow(gtx layout.Context, th *material.Theme, theme *AppTheme, selected bool, cells []string, actions layout.Widget) layout.Dimensions {
	bg := theme.Surface
	borderCol := theme.Border
	if selected {
		bg = theme.AccentSoft
		borderCol = theme.Link
	}
	macro := op.Record(gtx.Ops)
	dims := backupTableRowInset().Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		var children []layout.FlexChild
		cellIdx := 0
		for _, col := range backupTableColumns {
			col := col
			if col.Weight == 0 && col.MinWidth > 0 {
				children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = gtx.Dp(col.MinWidth)
					return actions(gtx)
				}))
				continue
			}
			text := ""
			if cellIdx < len(cells) {
				text = cells[cellIdx]
			}
			cellIdx++
			weight := col.Weight
			cellText := text
			children = append(children, layout.Flexed(weight, func(gtx layout.Context) layout.Dimensions {
				lbl := material.Body2(th, cellText)
				lbl.Color = theme.Text
				return lbl.Layout(gtx)
			}))
		}
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx, children...)
	})
	call := macro.Stop()
	radius := gtx.Dp(theme.RadiusSm)
	borderedRoundedRect(gtx, dims.Size, radius, bg, borderCol, gtx.Dp(unit.Dp(1)))
	call.Add(gtx.Ops)
	return dims
}

func tableHeader(gtx layout.Context, th *material.Theme, theme *AppTheme, cols []string) layout.Dimensions {
	var children []layout.FlexChild
	for _, c := range cols {
		c := c
		if c == "" {
			continue
		}
		children = append(children, layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			lbl := material.Caption(th, c)
			lbl.Color = theme.TextMuted
			return lbl.Layout(gtx)
		}))
	}
	return layout.Inset{
		Top: unit.Dp(4), Bottom: unit.Dp(12),
		Left: unit.Dp(4), Right: unit.Dp(4),
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx, children...)
	})
}

func tableRow(gtx layout.Context, th *material.Theme, theme *AppTheme, selected bool, cols []string, action string, actionBtn *widget.Clickable, onClick func()) layout.Dimensions {
	bg := theme.Surface
	borderCol := theme.Border
	if selected {
		bg = theme.AccentSoft
		borderCol = theme.Link
	}
	macro := op.Record(gtx.Ops)
	dims := layout.Inset{
		Top: unit.Dp(12), Bottom: unit.Dp(12),
		Left: unit.Dp(16), Right: unit.Dp(16),
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		var children []layout.FlexChild
		for _, c := range cols {
			c := c
			children = append(children, layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				lbl := material.Body2(th, c)
				lbl.Color = theme.Text
				return lbl.Layout(gtx)
			}))
		}
		if action != "" && actionBtn != nil {
			children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return secondaryButton(gtx, th, theme, actionBtn, action, onClick)
			}))
		}
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx, children...)
	})
	call := macro.Stop()
	radius := gtx.Dp(theme.RadiusSm)
	borderedRoundedRect(gtx, dims.Size, radius, bg, borderCol, gtx.Dp(unit.Dp(1)))
	call.Add(gtx.Ops)
	return dims
}
