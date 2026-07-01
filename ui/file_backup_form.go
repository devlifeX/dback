package ui

import (
	"fmt"
	"strings"
	"time"

	"dback/models"

	"gioui.org/layout"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

var fileCompressionValues = []string{
	string(models.ArchiveCompressionZstd),
	string(models.ArchiveCompressionGzip),
}

type fileBackupPathRow struct {
	id         string
	name       widget.Editor
	remotePath widget.Editor
	remove     widget.Clickable
}

type FileBackupForm struct {
	enabled            widget.Bool
	destination        widget.Editor
	compression        widget.Enum
	exclude            widget.Editor
	paths              []fileBackupPathRow
	addPath            widget.Clickable
	selectFolderBtn    widget.Clickable
	defaultDestination string
}

func newFileBackupForm(p models.Profile, defaultDest string) *FileBackupForm {
	f := &FileBackupForm{defaultDestination: defaultDest}
	f.enabled.Value = p.FileBackupEnabled
	f.compression.Value = string(models.NormalizeArchiveCompression(p.FileBackupCompression))
	dest := p.FileBackupDestination
	if strings.TrimSpace(dest) == "" {
		dest = p.Destination
	}
	if strings.TrimSpace(dest) == "" {
		dest = defaultDest
	}
	setEditorText(&f.destination, dest)
	setEditorMultiline(&f.exclude, strings.Join(p.FileBackupExclude, "\n"))
	for _, path := range p.FileBackupPaths {
		f.paths = append(f.paths, newFileBackupPathRow(path))
	}
	return f
}

func newFileBackupPathRow(p models.FileBackupPath) fileBackupPathRow {
	row := fileBackupPathRow{id: p.ID}
	if row.id == "" {
		row.id = fmt.Sprintf("%d", time.Now().UnixNano())
	}
	setEditorText(&row.name, p.Name)
	setEditorText(&row.remotePath, p.RemotePath)
	return row
}

func (f *FileBackupForm) addEmptyPath() {
	f.paths = append(f.paths, newFileBackupPathRow(models.FileBackupPath{
		ID: fmt.Sprintf("%d", time.Now().UnixNano()),
	}))
}

func (f *FileBackupForm) applyToProfile(p models.Profile) models.Profile {
	p.FileBackupEnabled = f.enabled.Value
	p.FileBackupDestination = strings.TrimSpace(editorText(&f.destination))
	p.FileBackupCompression = models.ArchiveCompression(f.compression.Value)
	if p.FileBackupCompression == "" {
		p.FileBackupCompression = models.ArchiveCompressionZstd
	}
	p.FileBackupExclude = parseExcludeLines(editorText(&f.exclude))
	p.FileBackupPaths = nil
	for _, row := range f.paths {
		name := strings.TrimSpace(editorText(&row.name))
		remote := strings.TrimSpace(editorText(&row.remotePath))
		if name == "" && remote == "" {
			continue
		}
		path := models.FileBackupPath{
			ID:         row.id,
			Name:       name,
			RemotePath: remote,
		}
		_ = path.Normalize()
		p.FileBackupPaths = append(p.FileBackupPaths, path)
	}
	return p
}

func parseExcludeLines(text string) []string {
	var out []string
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		out = append(out, line)
	}
	return out
}

func setEditorMultiline(e *widget.Editor, text string) {
	e.SingleLine = false
	setEditorText(e, text)
}

func (f *FileBackupForm) layout(gtx layout.Context, th *material.Theme, theme *AppTheme, u *UI) layout.Dimensions {
	if f.addPath.Clicked(gtx) {
		f.addEmptyPath()
		u.invalidate()
	}
	for i := range f.paths {
		i := i
		if f.paths[i].remove.Clicked(gtx) {
			f.paths = append(f.paths[:i], f.paths[i+1:]...)
			u.invalidate()
			break
		}
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return card(gtx, theme, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						lbl := material.Subtitle1(th, "File Backup")
						lbl.Color = theme.Text
						return lbl.Layout(gtx)
					}),
					layout.Rigid(vgap(theme)),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return checkboxField(gtx, th, theme, &f.enabled, "Enabled")
					}),
					layout.Rigid(vgap(theme)),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal, Alignment: layout.End}.Layout(gtx,
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								return labeledField(gtx, th, theme, "Destination Folder", func(gtx layout.Context) layout.Dimensions {
									return editorField(gtx, th, theme, &f.destination, f.defaultDestination)
								})
							}),
							layout.Rigid(hgap(theme)),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return secondaryButton(gtx, th, theme, &f.selectFolderBtn, "Browse", func() {
									u.pickFolder(func(path string) {
										if isDocumentURIPath(path) {
											setEditorText(&f.destination, f.defaultDestination)
										} else {
											setEditorText(&f.destination, path)
										}
										u.invalidate()
									})
								})
							}),
						)
					}),
					layout.Rigid(vgap(theme)),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return enumField(gtx, th, theme, &f.compression, "Compression", fileCompressionValues)
					}),
					layout.Rigid(vgap(theme)),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return labeledField(gtx, th, theme, "Exclude patterns", func(gtx layout.Context) layout.Dimensions {
							f.exclude.SingleLine = false
							return editorField(gtx, th, theme, &f.exclude, "One pattern per line")
						})
					}),
				)
			})
		}),
		layout.Rigid(vgap(theme)),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return card(gtx, theme, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						lbl := material.Subtitle1(th, "Paths")
						lbl.Color = theme.Text
						return lbl.Layout(gtx)
					}),
					layout.Rigid(vgap(theme)),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						var rows []layout.FlexChild
						for i := range f.paths {
							row := &f.paths[i]
							rows = append(rows, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
									layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
										return labeledField(gtx, th, theme, "Name", func(gtx layout.Context) layout.Dimensions {
											return editorField(gtx, th, theme, &row.name, "Website")
										})
									}),
									layout.Rigid(hgap(theme)),
									layout.Flexed(2, func(gtx layout.Context) layout.Dimensions {
										return labeledField(gtx, th, theme, "Remote Path", func(gtx layout.Context) layout.Dimensions {
											return editorField(gtx, th, theme, &row.remotePath, "/var/www/html")
										})
									}),
									layout.Rigid(hgap(theme)),
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										return secondaryButton(gtx, th, theme, &row.remove, "−", func() {
											u.invalidate()
										})
									}),
								)
							}))
							rows = append(rows, layout.Rigid(vgap(theme)))
						}
						if len(rows) == 0 {
							rows = append(rows, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return mutedLabel(gtx, th, theme, "No paths yet. Add one to include directories in file backup.")
							}))
						}
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx, rows...)
					}),
					layout.Rigid(vgap(theme)),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return secondaryButton(gtx, th, theme, &f.addPath, "+ Add path", func() {
							u.invalidate()
						})
					}),
				)
			})
		}),
	)
}
