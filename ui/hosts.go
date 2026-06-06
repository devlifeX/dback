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

	"dback/internal/paths"
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
				backup:    new(widget.Clickable),
				edit:      new(widget.Clickable),
				duplicate: new(widget.Clickable),
				delete:    new(widget.Clickable),
				more:      new(widget.Clickable),
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
						return fixedWidthSuccessButton(gtx, th, theme, cards.backup, "Backup", unit.Dp(150), func() {
							u.runBackup(p)
						})
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
		backup:    new(widget.Clickable),
		edit:      new(widget.Clickable),
		duplicate: new(widget.Clickable),
		delete:    new(widget.Clickable),
		more:      new(widget.Clickable),
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
	}()
}

func (u *UI) openProfileEditor(p models.Profile) {
	u.editingProfile = p
	setEditorText(&u.profileName, p.Name)
	setEditorText(&u.profileGroup, p.Group)
	defaultDest := paths.DefaultBackupDestination()
	u.hostForm = newSettingsForm(p, defaultDest)
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
