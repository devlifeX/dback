package ui

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"dback/models"

	"gioui.org/layout"
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

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return sectionTitle(gtx, th, theme, "Hosts")
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return primaryButton(gtx, th, theme, &u.addHostBtn, "+ Host", func() {
						u.openProfileEditor(defaultProfile())
					})
				}),
			)
		}),
		layout.Rigid(vgap(theme)),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return editorField(gtx, th, theme, &u.searchEditor, "Search hosts...")
		}),
		layout.Rigid(vgap(theme)),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return scrollArea(gtx, th, &hostsList, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return mutedLabel(gtx, th, theme, "Groups")
					}),
					layout.Rigid(vgap(theme)),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return u.layoutGroupCards(gtx, th, theme, profiles)
					}),
					layout.Rigid(spacer(theme, unit.Dp(20))),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return mutedLabel(gtx, th, theme, "Hosts")
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
	profiles := u.core.Profiles()
	q := strings.ToLower(strings.TrimSpace(u.search))
	if q == "" {
		return profiles
	}
	var filtered []models.Profile
	for _, p := range profiles {
		if strings.Contains(strings.ToLower(p.Name), q) ||
			strings.Contains(strings.ToLower(p.Host), q) ||
			strings.Contains(strings.ToLower(p.Group), q) {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

func (u *UI) layoutGroupCards(gtx layout.Context, th *material.Theme, theme *AppTheme, profiles []models.Profile) layout.Dimensions {
	counts := map[string]int{}
	for _, p := range profiles {
		group := p.Group
		if group == "" {
			group = "Default"
		}
		counts[group]++
	}
	if len(counts) == 0 {
		return card(gtx, theme, func(gtx layout.Context) layout.Dimensions {
			return mutedLabel(gtx, th, theme, "Create a host to start.")
		})
	}
	var children []layout.FlexChild
	for group, count := range counts {
		group, count := group, count
		children = append(children, layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return card(gtx, theme, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						lbl := material.Body1(th, group)
						lbl.Color = theme.Text
						return lbl.Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return mutedLabel(gtx, th, theme, fmt.Sprintf("%d hosts", count))
					}),
				)
			})
		}))
		children = append(children, layout.Rigid(hgap(theme)))
	}
	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, children...)
}

func (u *UI) layoutProfileCards(gtx layout.Context, th *material.Theme, theme *AppTheme, profiles []models.Profile) layout.Dimensions {
	if len(profiles) == 0 {
		return card(gtx, theme, func(gtx layout.Context) layout.Dimensions {
			return mutedLabel(gtx, th, theme, "No hosts match your search.")
		})
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
			}
			u.profileCards[p.ID] = cards
		}

		subtitle := fmt.Sprintf("%s@%s:%s - %s", p.SSHUser, p.Host, p.Port, p.TargetDBName)
		if p.ConnectionType == models.ConnectionTypeWordPress {
			subtitle = p.WPUrl + " - WordPress"
		}
		if p.IsDocker {
			subtitle += " (Docker)"
		}

		rows = append(rows, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return card(gtx, theme, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						lbl := material.Body1(th, p.Name)
						lbl.Color = theme.Text
						return lbl.Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return mutedLabel(gtx, th, theme, p.Group)
					}),
					layout.Rigid(vgap(theme)),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return mutedLabel(gtx, th, theme, subtitle)
					}),
					layout.Rigid(vgap(theme)),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return secondaryButton(gtx, th, theme, cards.backup, "Backup", func() {
									u.runBackup(p)
								})
							}),
							layout.Rigid(hgap(theme)),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return secondaryButton(gtx, th, theme, cards.edit, "Edit", func() {
									u.openProfileEditor(p)
								})
							}),
							layout.Rigid(hgap(theme)),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return secondaryButton(gtx, th, theme, cards.duplicate, "Duplicate", func() {
									u.duplicateProfile(p)
								})
							}),
							layout.Rigid(hgap(theme)),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return dangerButton(gtx, th, theme, cards.delete, "Delete", func() {
									profile := p
									u.showConfirm("Delete host", "Delete "+profile.Name+"?", func() {
										if err := u.core.DeleteProfile(profile.ID); err != nil {
											u.showError(err)
											return
										}
										delete(u.profileCards, profile.ID)
										u.openHosts()
									})
								})
							}),
						)
					}),
				)
			})
		}))
		rows = append(rows, layout.Rigid(vgap(theme)))
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, rows...)
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
			progress := job.Progress
			if total > 0 {
				progress = float64(current) / float64(total)
			} else {
				progress += 0.03
				if progress > 0.95 {
					progress = 0.1
				}
			}
			u.updateJob(job.ID, message, progress, "")
		})
		if err != nil {
			if errors.Is(err, context.Canceled) {
				u.finishJob(job.ID, "Backup canceled", nil)
				return
			}
			u.finishJob(job.ID, "Backup failed", err)
			return
		}
		u.finishJob(job.ID, "Backup complete: "+filepath.Base(record.FilePath), nil)
	}()
}

func (u *UI) openProfileEditor(p models.Profile) {
	u.editingProfile = p
	setEditorText(&u.profileName, p.Name)
	setEditorText(&u.profileGroup, p.Group)
	defaultDest := defaultBackupDir(u.platform)
	u.hostForm = newSettingsForm(p, defaultDest)
	u.queryForm = newQueryForm(p)
	u.profileTab = 0
	u.view = ViewProfileEditor
	u.invalidate()
}
