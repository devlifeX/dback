package ui

import (
	"fmt"

	"dback/models"

	"gioui.org/layout"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

type remoteUploadDestRow struct {
	id       string
	name     string
	enabled  widget.Bool
	moveUp   widget.Clickable
	moveDown widget.Clickable
}

type RemoteUploadForm struct {
	profileID       string
	rows            []remoteUploadDestRow
	autoUploadDB    widget.Bool
	autoUploadFiles widget.Bool
	allDestinations []models.RemoteDestination
}

func newRemoteUploadForm(p models.Profile, destinations []models.RemoteDestination) *RemoteUploadForm {
	f := &RemoteUploadForm{profileID: p.ID, allDestinations: destinations}
	f.autoUploadDB.Value = p.RemoteAutoUploadDB
	f.autoUploadFiles.Value = p.RemoteAutoUploadFiles

	selected := map[string]bool{}
	order := append([]string(nil), p.RemoteUploadDestinationIDs...)
	for _, id := range order {
		selected[id] = true
	}
	for _, id := range order {
		f.rows = append(f.rows, f.rowForDestination(id, true))
	}
	for _, dest := range destinations {
		if selected[dest.ID] {
			continue
		}
		f.rows = append(f.rows, f.rowForDestination(dest.ID, false))
	}
	return f
}

func (f *RemoteUploadForm) rowForDestination(id string, enabled bool) remoteUploadDestRow {
	name := id
	for _, d := range f.allDestinations {
		if d.ID == id {
			name = d.Name
			break
		}
	}
	row := remoteUploadDestRow{id: id, name: name}
	row.enabled.Value = enabled
	return row
}

func (f *RemoteUploadForm) applyToProfile(p models.Profile) models.Profile {
	var ids []string
	for _, row := range f.rows {
		if row.enabled.Value {
			ids = append(ids, row.id)
		}
	}
	p.RemoteUploadDestinationIDs = ids
	p.RemoteAutoUploadDB = f.autoUploadDB.Value
	p.RemoteAutoUploadFiles = f.autoUploadFiles.Value
	return p
}

func (f *RemoteUploadForm) moveRow(index, delta int) {
	next := index + delta
	if index < 0 || index >= len(f.rows) || next < 0 || next >= len(f.rows) {
		return
	}
	f.rows[index], f.rows[next] = f.rows[next], f.rows[index]
}

func (f *RemoteUploadForm) layout(gtx layout.Context, th *material.Theme, theme *AppTheme, invalidate func()) layout.Dimensions {
	if len(f.allDestinations) == 0 {
		return mutedLabel(gtx, th, theme, "Add remote destinations in Settings → Sync first.")
	}
	f.autoUploadDB.Update(gtx)
	f.autoUploadFiles.Update(gtx)

	var rows []layout.FlexChild
	for i := range f.rows {
		i := i
		row := &f.rows[i]
		row.enabled.Update(gtx)
		rows = append(rows, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return checkboxField(gtx, th, theme, &row.enabled, row.name)
				}),
				layout.Flexed(1, layout.Spacer{}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if i == 0 {
						return layout.Dimensions{}
					}
					return secondaryButton(gtx, th, theme, &row.moveUp, "↑", func() {
						f.moveRow(i, -1)
						if invalidate != nil {
							invalidate()
						}
					})
				}),
				layout.Rigid(hgap(theme)),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if i >= len(f.rows)-1 {
						return layout.Dimensions{}
					}
					return secondaryButton(gtx, th, theme, &row.moveDown, "↓", func() {
						f.moveRow(i, 1)
						if invalidate != nil {
							invalidate()
						}
					})
				}),
			)
		}))
		rows = append(rows, layout.Rigid(vgap(theme)))
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return f.layoutS3FolderID(gtx, th, theme)
		}),
		layout.Rigid(vgap(theme)),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return mutedLabel(gtx, th, theme, "Select destinations in upload order. Each backup is mirrored to all selected destinations.")
		}),
		layout.Rigid(vgap(theme)),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, rows...)
		}),
		layout.Rigid(vgap(theme)),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return checkboxField(gtx, th, theme, &f.autoUploadDB, "Auto upload after DB backup")
		}),
		layout.Rigid(vgap(theme)),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return checkboxField(gtx, th, theme, &f.autoUploadFiles, "Auto upload after File backup")
		}),
	)
}

func (f *RemoteUploadForm) layoutS3FolderID(gtx layout.Context, th *material.Theme, theme *AppTheme) layout.Dimensions {
	if f.profileID == "" {
		return mutedLabel(gtx, th, theme, "S3 folder ID will be created after saving this host.")
	}
	prefix := fmt.Sprintf("dback/backups/%s/", f.profileID)
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			lbl := material.Body2(th, "S3 folder ID: "+f.profileID)
			lbl.Color = theme.Text
			return lbl.Layout(gtx)
		}),
		layout.Rigid(vgap(theme)),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			lbl := material.Body2(th, "Path prefix: "+prefix)
			lbl.Color = theme.Text
			return lbl.Layout(gtx)
		}),
		layout.Rigid(vgap(theme)),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return mutedLabel(gtx, th, theme, "Backups for this host are stored under this ID on S3, not the host name.")
		}),
	)
}

func (u *UI) pendingUploadLabel(profileID string) string {
	count, _, err := u.core.PendingUploads(profileID)
	if err != nil || count == 0 {
		return ""
	}
	if count == 1 {
		return "1 backup not fully uploaded"
	}
	return fmt.Sprintf("%d backups not fully uploaded", count)
}
