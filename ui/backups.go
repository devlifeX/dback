package ui

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"dback/models"

	"gioui.org/layout"
	"gioui.org/op"
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

	status := "Select a backup file to import or open its folder."
	if u.selectedBackup != nil {
		status = filepath.Base(u.selectedBackup.FilePath)
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return labeledEnumDropdownField(gtx, th, theme, &u.backupHostSelect, "Host filter", hostValues, hostLabels, &u.backupHostDropdown)
		}),
		layout.Rigid(vgap(theme)),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return scrollArea(gtx, th, &u.backupList, func(gtx layout.Context) layout.Dimensions {
				header := []string{"When", "Profile", "Database", "Size", "File", ""}
				var rows []layout.FlexChild
				rows = append(rows, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return tableHeader(gtx, th, theme, header)
				}))
				for _, record := range records {
					btn, ok := u.backupRows[record.ID]
					if !ok {
						btn = new(widget.Clickable)
						u.backupRows[record.ID] = btn
					}
					rowBtn, ok := u.backupRows["sel_"+record.ID]
					if !ok {
						rowBtn = new(widget.Clickable)
						u.backupRows["sel_"+record.ID] = rowBtn
					}
					selected := u.selectedBackupID == record.ID
					importBtn := btn
					selectBtn := rowBtn
					rec := record
					folderBtn, okFolder := u.backupFolderBtns[rec.ID]
					if !okFolder {
						folderBtn = new(widget.Clickable)
						u.backupFolderBtns[rec.ID] = folderBtn
					}
					openBtn := folderBtn
					rows = append(rows, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if selectBtn.Clicked(gtx) {
							r := rec
							u.selectedBackup = &r
							u.selectedBackupID = rec.ID
						}
						return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								return tableRow(gtx, th, theme, selected, []string{
									formatRelativeTime(rec.ExportDate),
									rec.ProfileName,
									rec.DatabaseName,
									rec.FileSize,
									filepath.Base(rec.FilePath),
								}, "Import", importBtn, func() {
									u.openBackupDetail(rec)
								})
							}),
							layout.Rigid(hgap(theme)),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return secondaryButton(gtx, th, theme, openBtn, "Folder", func() {
									u.openBackupFolder(rec.FilePath)
								})
							}),
						)
					}))
				}
				if len(records) == 0 {
					rows = append(rows, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return mutedLabel(gtx, th, theme, "No backup files yet.")
					}))
				}
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx, rows...)
			})
		}),
		layout.Rigid(vgap(theme)),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return mutedLabel(gtx, th, theme, status)
		}),
		layout.Rigid(vgap(theme)),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return primaryButton(gtx, th, theme, &u.importSelectedBtn, "Import Selected", func() {
						if u.selectedBackup == nil {
							u.showInfo("Backup Files", "Select a backup first.")
							return
						}
						u.openBackupDetail(*u.selectedBackup)
					})
				}),
				layout.Rigid(hgap(theme)),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return secondaryButton(gtx, th, theme, &u.openFolderBtn, "Open Folder", func() {
						if u.selectedBackup == nil {
							u.showInfo("Backup Files", "Select a backup first.")
							return
						}
						u.openBackupFolder(u.selectedBackup.FilePath)
					})
				}),
			)
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

func (u *UI) openBackupDetail(record models.ExportRecord) {
	u.selectedBackup = &record
	u.view = ViewBackupDetail
	importable := importableProfiles(u.core.Profiles())
	if len(importable) > 0 {
		u.destSelect.Value = importable[0].ID
	} else {
		u.destSelect.Value = ""
	}
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
			if canImport {
				return labeledEnumDropdownField(gtx, th, theme, &u.destSelect, "Destination Host", values, labels, &u.destHostDropdown)
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
			u.finishJob(job.ID, "Import canceled or failed", err)
			return
		}
		u.finishJob(job.ID, "Import complete", nil)
	}()
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
