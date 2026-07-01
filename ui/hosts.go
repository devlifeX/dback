package ui

import (
	"context"
	"errors"
	"fmt"
	"image"
	"path/filepath"
	"sort"
	"strings"
	"time"

	coreapp "dback/internal/app"
	"dback/internal/debug"
	"dback/internal/paths"
	"dback/internal/remote"
	"dback/models"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

var hostsList widget.List

func (u *UI) layoutHosts(gtx layout.Context, th *material.Theme) layout.Dimensions {
	theme := u.theme

	switch u.view {
	case ViewProfileEditor:
		return u.layoutProfileEditor(gtx, th)
	default:
		return u.layoutHostsList(gtx, th, theme)
	}
}

func (u *UI) layoutHostsList(gtx layout.Context, th *material.Theme, theme *AppTheme) layout.Dimensions {
	u.search = editorText(&u.searchEditor)
	profiles := u.filteredProfiles()
	allProfiles := u.core.Profiles()

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return pageHeader(gtx, th, theme, "Hosts", func(gtx layout.Context) layout.Dimensions {
				return primaryButton(gtx, th, theme, &u.addHostBtn, "+ Host", func() {
					u.openProfileEditor(defaultProfile())
				})
			})
		}),
		layout.Rigid(vgap(theme)),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return searchField(gtx, th, theme, &u.searchEditor, "Search hosts...")
		}),
		layout.Rigid(vgap(theme)),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return scrollArea(gtx, th, &hostsList, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return sectionLabel(gtx, th, theme, "Groups")
					}),
					layout.Rigid(vgap(theme)),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return u.layoutGroupCards(gtx, th, theme, allProfiles)
					}),
					layout.Rigid(spacer(theme, theme.SectionGap)),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return sectionLabel(gtx, th, theme, "Hosts")
					}),
					layout.Rigid(vgap(theme)),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return u.layoutProfileCards(gtx, th, theme, profiles)
					}),
				)
			})
		}),
	)
}

func (u *UI) filteredProfiles() []models.Profile {
	return filterProfiles(u.core.Profiles(), editorText(&u.searchEditor), u.selectedGroup)
}

func (u *UI) layoutGroupCards(gtx layout.Context, th *material.Theme, theme *AppTheme, profiles []models.Profile) layout.Dimensions {
	counts := map[string]int{}
	for _, p := range profiles {
		counts[normalizeGroup(p.Group)]++
	}
	if len(counts) == 0 {
		return card(gtx, theme, func(gtx layout.Context) layout.Dimensions {
			return mutedLabel(gtx, th, theme, "Create a host to start.")
		})
	}

	groups := collectGroups(profiles)
	var children []layout.FlexChild

	allKey := "__all__"
	allBtn, ok := u.groupChips[allKey]
	if !ok {
		allBtn = new(widget.Clickable)
		u.groupChips[allKey] = allBtn
	}
	total := len(profiles)
	children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		return chipButton(gtx, th, theme, allBtn, "All", fmt.Sprintf("%d hosts", total), u.selectedGroup == groupFilterAll, func() {
			u.selectedGroup = groupFilterAll
			u.invalidate()
		})
	}))

	for _, group := range groups {
		group := group
		count := counts[group]
		btn, ok := u.groupChips[group]
		if !ok {
			btn = new(widget.Clickable)
			u.groupChips[group] = btn
		}
		children = append(children, layout.Rigid(hgap(theme)))
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return chipButton(gtx, th, theme, btn, group, fmt.Sprintf("%d hosts", count), u.selectedGroup == group, func() {
				u.selectedGroup = group
				u.invalidate()
			})
		}))
	}

	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx, children...)
}

func (u *UI) layoutProfileCards(gtx layout.Context, th *material.Theme, theme *AppTheme, profiles []models.Profile) layout.Dimensions {
	if len(profiles) == 0 {
		return emptyState(gtx, th, theme, "No hosts match your search.")
	}
	var rows []layout.FlexChild
	for _, p := range profiles {
		p := p
		cards, ok := u.profileCards[p.ID]
		if !ok {
			cards = profileCardWidgets{
				backup:       new(widget.Clickable),
				backupFiles:  new(widget.Clickable),
				uploadRemote: new(widget.Clickable),
				edit:         new(widget.Clickable),
				duplicate:    new(widget.Clickable),
				delete:       new(widget.Clickable),
				more:         new(widget.Clickable),
			}
			u.profileCards[p.ID] = cards
		}

		subtitle := hostConnectionSubtitle(p)

		rows = append(rows, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return compactCard(gtx, theme, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								if !p.ImportProtected {
									return layout.Dimensions{}
								}
								return layout.Inset{Right: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return importProtectedIcon(gtx, theme)
								})
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								lbl := material.Body1(th, p.Name)
								lbl.Color = theme.Text
								return lbl.Layout(gtx)
							}),
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								return layout.Inset{Left: unit.Dp(20), Right: unit.Dp(20)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return mutedLabel(gtx, th, theme, subtitle)
								})
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return badge(gtx, th, theme, normalizeGroup(p.Group))
							}),
							layout.Rigid(hgap(theme)),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return u.layoutCardMoreButton(gtx, th, theme, p, cards)
							}),
						)
					}),
					layout.Rigid(spacer(theme, unit.Dp(8))),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if label := u.pendingUploadLabel(p.ID); label != "" {
							return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return mutedLabel(gtx, th, theme, label)
								}),
								layout.Rigid(vgap(theme)),
							)
						}
						return layout.Dimensions{}
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return fixedWidthSuccessButton(gtx, th, theme, cards.backup, "Backup", unit.Dp(120), func() {
									u.runBackup(p)
								})
							}),
							layout.Rigid(hgap(theme)),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								if !p.FileBackupReady() {
									return layout.Dimensions{}
								}
								return fixedWidthSuccessButton(gtx, th, theme, cards.backupFiles, "Backup Files", unit.Dp(120), func() {
									u.runFileBackup(p)
								})
							}),
							layout.Rigid(hgap(theme)),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								if len(p.RemoteUploadDestinationIDs) == 0 {
									return layout.Dimensions{}
								}
								uploadState := u.hostUploadState(p.ID)
								if uploadState.Running || u.isHostUploadRunning(p.ID) {
									return fixedWidthDisabledButton(gtx, th, theme, "Uploading...", unit.Dp(120))
								}
								return fixedWidthSecondaryButton(gtx, th, theme, cards.uploadRemote, "Upload", unit.Dp(120), func() {
									u.runRemoteUpload(p)
								})
							}),
						)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						uploadState := u.hostUploadState(p.ID)
						if uploadState.Running {
							return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
								layout.Rigid(vgap(theme)),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return progressBar(gtx, theme, uploadState.Progress)
								}),
								layout.Rigid(vgap(theme)),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									if uploadState.Status == "" {
										return layout.Dimensions{}
									}
									return mutedLabel(gtx, th, theme, uploadState.Status)
								}),
							)
						}
						if uploadState.Result != "" {
							return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
								layout.Rigid(vgap(theme)),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return uploadResultLabel(gtx, th, theme, uploadState.Result, uploadState.IsError)
								}),
							)
						}
						return layout.Dimensions{}
					}),
				)
			})
		}))
		rows = append(rows, layout.Rigid(spacer(theme, unit.Dp(8))))
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, rows...)
}

func (u *UI) layoutCardMoreButton(gtx layout.Context, th *material.Theme, theme *AppTheme, p models.Profile, cards profileCardWidgets) layout.Dimensions {
	if cards.more.Clicked(gtx) {
		if u.menuOpenID == p.ID {
			u.menuOpenID = ""
		} else {
			u.menuOpenID = p.ID
		}
		u.invalidate()
	}

	btnDims := renderButton(gtx, th, theme, cards.more, "⋮", btnSecondary, false)

	if u.menuOpenID != p.ID {
		return btnDims
	}

	items := []menuPopupItem{
		{
			label: "Edit",
			btn:   cards.edit,
			onClick: func() {
				u.menuOpenID = ""
				u.openProfileEditor(p)
			},
		},
		{
			label: "Duplicate",
			btn:   cards.duplicate,
			onClick: func() {
				u.menuOpenID = ""
				u.duplicateProfile(p)
			},
		},
		{
			label:  "Delete",
			danger: true,
			btn:    cards.delete,
			onClick: func() {
				u.menuOpenID = ""
				profile := p
				u.showConfirm("Delete host", "Delete "+profile.Name+"?", func() {
					if err := u.core.DeleteProfile(profile.ID); err != nil {
						u.showError(err)
						return
					}
					delete(u.profileCards, profile.ID)
					u.openHosts()
				})
			},
		},
	}

	// Dismiss menu when backdrop is clicked
	if u.menuCloseArea.Clicked(gtx) {
		u.menuOpenID = ""
		u.invalidate()
	}

	// Record the entire overlay (backdrop + popup) to be deferred.
	// op.Defer saves the current transform stack, so the popup is
	// correctly positioned relative to the ⋮ button on screen.
	popupW := gtx.Dp(unit.Dp(160))
	popupOffX := btnDims.Size.X - popupW // right-align popup edge with button right edge
	popupOffY := btnDims.Size.Y + gtx.Dp(unit.Dp(4))

	macro := op.Record(gtx.Ops)

	// Full-screen transparent backdrop to catch outside clicks
	backdropStack := op.Offset(image.Pt(-9999, -9999)).Push(gtx.Ops)
	backdropClip := clip.Rect{Max: image.Pt(99999, 99999)}.Push(gtx.Ops)
	u.menuCloseArea.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Dimensions{Size: image.Pt(99999, 99999)}
	})
	backdropClip.Pop()
	backdropStack.Pop()

	// Popup menu card
	popupStack := op.Offset(image.Pt(popupOffX, popupOffY)).Push(gtx.Ops)
	menuPopup(gtx, th, theme, items)
	popupStack.Pop()

	op.Defer(gtx.Ops, macro.Stop())

	return btnDims
}

func (u *UI) duplicateProfile(profile models.Profile) {
	clone := profile
	clone.ID = fmt.Sprintf("%d", time.Now().UnixNano())
	clone.Name = nextDuplicateName(profile.Name, u.core.Profiles())
	clone.ExportSettings = nil
	clone.ImportSettings = nil

	if err := u.core.SaveProfile(clone); err != nil {
		u.showError(err)
		return
	}
	u.profileCards[clone.ID] = profileCardWidgets{
		backup:       new(widget.Clickable),
		backupFiles:  new(widget.Clickable),
		uploadRemote: new(widget.Clickable),
		edit:         new(widget.Clickable),
		duplicate:    new(widget.Clickable),
		delete:       new(widget.Clickable),
		more:         new(widget.Clickable),
	}
	u.openHosts()
}

func nextDuplicateName(name string, profiles []models.Profile) string {
	base := strings.TrimSpace(name)
	if base == "" {
		base = "Host"
	}
	exists := map[string]bool{}
	for _, p := range profiles {
		exists[p.Name] = true
	}
	for i := 1; ; i++ {
		candidate := fmt.Sprintf("%s %d", base, i)
		if !exists[candidate] {
			return candidate
		}
	}
}

func (u *UI) runBackup(p models.Profile) {
	ctx, cancel := context.WithCancel(context.Background())
	job := u.addJob("Backup", p.Name, cancel)
	u.backupTab = 1
	u.openBackups()
	go func() {
		defer cancel()
		record, err := u.core.Backup(ctx, p, func(message string, current int64, total int64) {
			progress := float64(0)
			if total > 0 {
				progress = float64(current) / float64(total)
			}
			verifyPhase := strings.Contains(message, "Capturing fingerprint") ||
				strings.Contains(message, "Verifying backup integrity")
			u.setBackupJobProgress(job.ID, message, progress, verifyPhase)
			if verifyPhase {
				u.invalidateBackupCache()
			}
		})
		u.invalidateBackupCache()
		if err != nil {
			if errors.Is(err, context.Canceled) {
				u.finishJob(job.ID, "Backup canceled", nil)
				return
			}
			u.finishJob(job.ID, "Backup failed", err)
			return
		}
		u.setBackupJobRecord(job.ID, record.ID)
		u.finishJob(job.ID, "Backup complete: "+filepath.Base(record.FilePath), nil)
		u.maybeAutoRemoteUpload(p, []models.ExportRecord{record})
	}()
}

func (u *UI) runFileBackup(p models.Profile) {
	if !p.FileBackupReady() {
		u.showInfo("Backup Files", "Enable file backup and add at least one path in host settings.")
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	job := u.addFileBackupJob(p, cancel)
	u.backupTab = 1
	u.openBackups()
	go func() {
		defer cancel()
		result, err := u.core.BackupFiles(ctx, p, func(prog coreapp.FileBackupProgress) {
			u.setFileBackupJobProgress(job.ID, prog)
		})
		u.invalidateBackupCache()
		if err != nil {
			if errors.Is(err, context.Canceled) {
				u.finishFileBackupJob(job.ID, "Backup Files canceled", nil)
				return
			}
			status := "Backup Files failed"
			if result.PartialFail {
				status = "Backup Files partial failure"
			}
			u.finishFileBackupJob(job.ID, status, err)
			return
		}
		if len(result.Records) > 0 {
			u.setFileBackupJobRecord(job.ID, result.Records[len(result.Records)-1].ID)
		}
		u.finishFileBackupJob(job.ID, fmt.Sprintf("Backup Files complete (%d archives)", len(result.Records)), nil)
		u.maybeAutoRemoteUpload(p, result.Records)
	}()
}

func (u *UI) maybeAutoRemoteUpload(p models.Profile, records []models.ExportRecord) {
	if len(p.RemoteUploadDestinationIDs) == 0 {
		return
	}
	if !p.RemoteAutoUploadDB && !p.RemoteAutoUploadFiles {
		return
	}
	u.runRemoteUploadRecords(p, records)
}

func (u *UI) runRemoteUpload(p models.Profile) {
	if len(p.RemoteUploadDestinationIDs) == 0 {
		u.showInfo("Remote Upload", "Select remote destinations in host settings first.")
		return
	}
	u.runRemoteUploadRecords(p, nil)
}

func (u *UI) runRemoteUploadRecords(p models.Profile, records []models.ExportRecord) {
	if u.isHostUploadRunning(p.ID) {
		u.finishHostUpload(p.ID, "Upload already running", true)
		u.invalidate()
		return
	}

	var filterIDs []string
	for _, rec := range records {
		filterIDs = append(filterIDs, rec.ID)
	}

	u.setHostUploadRunning(p.ID)
	u.updateHostUploadProgress(p.ID, -1, "Checking remote backups...")
	u.invalidate()
	debug.Log("DEBUG", "RemoteUpload.UI", "prepare_start", fmt.Sprintf("host=%q profile_id=%s filter_records=%d timeout=%s", p.Name, p.ID, len(filterIDs), remote.PrepareUploadTimeout), p.Name, "", "")

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), remote.PrepareUploadTimeout)
		defer cancel()

		plan, err := u.core.PrepareProfileUpload(ctx, p.ID, filterIDs)
		u.invalidateBackupCache()
		if err != nil {
			msg := "Could not check remote backups: " + err.Error()
			if errors.Is(err, context.DeadlineExceeded) {
				msg = "Remote backup check timed out. Check your S3 connection and try again."
			}
			debug.Log("DEBUG", "RemoteUpload.UI", "prepare_failed", fmt.Sprintf("host=%q profile_id=%s", p.Name, p.ID), p.Name, "", err.Error())
			u.finishHostUpload(p.ID, msg, true)
			u.invalidate()
			return
		}

		debug.Log(
			"DEBUG",
			"RemoteUpload.UI",
			"prepare_done",
			fmt.Sprintf("retry=%d stale=%d latest_stale=%q", len(plan.RetryRecordIDs), len(plan.StaleRemoteRecordIDs), plan.LatestStaleRecordID),
			p.Name,
			"",
			"",
		)

		if len(plan.RetryRecordIDs) == 0 && len(plan.StaleRemoteRecordIDs) == 0 {
			u.finishHostUpload(p.ID, "All backups already uploaded", false)
			u.invalidate()
			return
		}

		if len(plan.StaleRemoteRecordIDs) > 1 {
			debug.Log("DEBUG", "RemoteUpload.UI", "prompt_missing", fmt.Sprintf("stale_records=%d", len(plan.StaleRemoteRecordIDs)), p.Name, "", "")
			u.showRemoteUploadMissingDialog(p, plan)
			return
		}

		uploadIDs := mergeRecordIDs(plan.RetryRecordIDs, plan.StaleRemoteRecordIDs)
		debug.Log("DEBUG", "RemoteUpload.UI", "upload_start", fmt.Sprintf("record_ids=%v", uploadIDs), p.Name, "", "")
		u.startRemoteUpload(p, uploadIDs)
	}()
}

func (u *UI) showRemoteUploadMissingDialog(p models.Profile, plan coreapp.RemoteUploadPlan) {
	u.hostUploadMu.Lock()
	delete(u.hostUploadStates, p.ID)
	u.hostUploadMu.Unlock()

	count := len(plan.StaleRemoteRecordIDs)
	msg := fmt.Sprintf("%d backups exist locally but are missing on remote.", count)
	if count == 1 {
		msg = "1 backup exists locally but is missing on remote."
	}
	u.pendingRemoteUpload = &pendingRemoteUploadChoice{
		profile:  p,
		retryIDs: append([]string(nil), plan.RetryRecordIDs...),
		staleIDs: append([]string(nil), plan.StaleRemoteRecordIDs...),
		latestID: plan.LatestStaleRecordID,
	}
	u.showDialog(DialogState{
		Kind:    DialogRemoteUploadMissing,
		Title:   "Missing on remote",
		Message: msg + " Upload the latest backup only, or upload all?",
	})
	u.invalidate()
}

func (u *UI) confirmRemoteUploadChoice(uploadAll bool) {
	choice := u.pendingRemoteUpload
	u.pendingRemoteUpload = nil
	if choice == nil {
		return
	}
	var uploadIDs []string
	if uploadAll {
		uploadIDs = mergeRecordIDs(choice.retryIDs, choice.staleIDs)
	} else {
		uploadIDs = mergeRecordIDs(choice.retryIDs, []string{choice.latestID})
	}
	u.startRemoteUpload(choice.profile, uploadIDs)
}

func mergeRecordIDs(groups ...[]string) []string {
	seen := map[string]bool{}
	var out []string
	for _, group := range groups {
		for _, id := range group {
			if id == "" || seen[id] {
				continue
			}
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}

func (u *UI) startRemoteUpload(p models.Profile, recordIDs []string) {
	u.setHostUploadRunning(p.ID)
	u.updateHostUploadProgress(p.ID, -1, "Starting upload...")
	u.invalidate()
	debug.Log("DEBUG", "RemoteUpload.UI", "upload_exec", fmt.Sprintf("record_ids=%v", recordIDs), p.Name, "", "")

	ctx, cancel := context.WithCancel(context.Background())
	job := u.addJob("Remote Upload", p.Name, cancel)
	go func() {
		defer cancel()
		result, err := u.core.UploadProfileBackups(ctx, p.ID, recordIDs, func(prog coreapp.RemoteUploadProgress) {
			status := formatRemoteUploadStatus(prog)
			progress := float64(-1)
			if prog.Total > 0 {
				progress = float64(prog.Current) / float64(prog.Total)
			}
			u.updateHostUploadProgress(p.ID, progress, status)
			u.updateJob(job.ID, status, progress, prog.Error)
			u.invalidate()
		})
		u.invalidateBackupCache()

		message := coreapp.FormatRemoteUploadResultMessage(result)

		if err != nil {
			if errors.Is(err, context.Canceled) {
				u.finishHostUpload(p.ID, "Remote upload canceled", false)
				u.finishJob(job.ID, "Remote upload canceled", nil)
				u.invalidate()
				return
			}
			if errors.Is(err, coreapp.ErrRemoteUploadRunning) {
				u.finishHostUpload(p.ID, "Upload already running", true)
				u.finishJob(job.ID, "Remote upload already running", err)
				u.invalidate()
				return
			}
			u.finishHostUpload(p.ID, message, true)
			u.finishJob(job.ID, "Remote upload failed", err)
			u.invalidate()
			return
		}
		u.finishHostUpload(p.ID, message, false)
		u.finishJob(job.ID, message, nil)
		u.invalidate()
	}()
}

func formatRemoteUploadStatus(prog coreapp.RemoteUploadProgress) string {
	if prog.Total > 0 {
		if prog.RecordIndex > 0 && prog.RecordTotal > 0 {
			return fmt.Sprintf("Uploading backup %d/%d (%d/%d)...", prog.RecordIndex, prog.RecordTotal, prog.Current, prog.Total)
		}
		return fmt.Sprintf("Uploading (%d/%d)...", prog.Current, prog.Total)
	}
	if prog.Status == models.RemoteUploadUploading {
		return "Uploading..."
	}
	if prog.Error != "" {
		return prog.Error
	}
	return "Uploading..."
}

func uploadResultLabel(gtx layout.Context, th *material.Theme, theme *AppTheme, message string, isError bool) layout.Dimensions {
	lbl := material.Body2(th, message)
	if isError {
		lbl.Color = theme.Danger
	} else {
		lbl.Color = theme.Success
	}
	return lbl.Layout(gtx)
}

func (u *UI) openProfileEditor(p models.Profile) {
	u.editingProfile = p
	setEditorText(&u.profileName, p.Name)
	setEditorText(&u.profileGroup, p.Group)
	defaultDest := paths.DefaultBackupDestination()
	destinations, _ := u.core.ListRemoteDestinations()
	u.hostForm = newSettingsForm(p, defaultDest, destinations)
	u.queryForm = newQueryForm(p)
	u.profileTab = 0
	u.view = ViewProfileEditor
	u.invalidate()
}

// sortedBackupHostOptions returns host filter values and labels for backups view.
func sortedBackupHostOptions(profiles []models.Profile) (values, labels []string) {
	type hostOpt struct {
		id    string
		label string
	}
	opts := make([]hostOpt, 0, len(profiles))
	for _, p := range profiles {
		label := p.Name
		if p.Group != "" {
			label += " (" + normalizeGroup(p.Group) + ")"
		}
		opts = append(opts, hostOpt{id: p.ID, label: label})
	}
	sort.Slice(opts, func(i, j int) bool { return opts[i].label < opts[j].label })
	values = append(values, backupFilterAll)
	labels = append(labels, "All hosts")
	for _, o := range opts {
		values = append(values, o.id)
		labels = append(labels, o.label)
	}
	return values, labels
}
