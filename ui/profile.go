package ui

import (
	"fmt"
	"strings"

	"dback/models"

	"gioui.org/layout"
	"gioui.org/widget/material"
)

func (u *UI) layoutProfileEditor(gtx layout.Context, th *material.Theme) layout.Dimensions {
	theme := u.theme
	showQuery := u.importForm.supportsSQLQuery()

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return secondaryButton(gtx, th, theme, &u.backBtn, "← Back", func() {
						u.view = ViewList
						u.invalidate()
					})
				}),
				layout.Rigid(hgap(theme)),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return sectionTitle(gtx, th, theme, "Host Profile")
				}),
			)
		}),
		layout.Rigid(vgap(theme)),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return card(gtx, theme, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return labeledField(gtx, th, theme, "Name", func(gtx layout.Context) layout.Dimensions {
							return editorField(gtx, th, theme, &u.profileName, "Host name")
						})
					}),
					layout.Rigid(hgap(theme)),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return labeledField(gtx, th, theme, "Group", func(gtx layout.Context) layout.Dimensions {
							return editorField(gtx, th, theme, &u.profileGroup, "Default")
						})
					}),
				)
			})
		}),
		layout.Rigid(vgap(theme)),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return tabButton(gtx, th, theme, &u.tabExport, "Export", u.profileTab == 0, func() {
						u.profileTab = 0
						u.invalidate()
					})
				}),
				layout.Rigid(hgap(theme)),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return tabButton(gtx, th, theme, &u.tabImport, "Import", u.profileTab == 1, func() {
						u.profileTab = 1
						u.invalidate()
					})
				}),
				layout.Rigid(hgap(theme)),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if !showQuery {
						return layout.Dimensions{}
					}
					return tabButton(gtx, th, theme, &u.tabQuery, "Query", u.profileTab == 2, func() {
						u.profileTab = 2
						u.invalidate()
					})
				}),
			)
		}),
		layout.Rigid(vgap(theme)),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			switch u.profileTab {
			case 1:
				return u.importForm.layout(gtx, th, theme, u)
			case 2:
				if showQuery {
					return u.queryForm.layout(gtx, th, theme, u, u.profileFromEditors, strings.TrimSpace(u.exportForm.settings().TargetDBName))
				}
				fallthrough
			default:
				return u.exportForm.layout(gtx, th, theme, u)
			}
		}),
		layout.Rigid(vgap(theme)),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return secondaryButton(gtx, th, theme, &u.copyExpToImp, "Copy Export → Import", func() {
						u.importForm.apply(u.exportForm.settings())
					})
				}),
				layout.Rigid(hgap(theme)),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return secondaryButton(gtx, th, theme, &u.copyImpToExp, "Copy Import → Export", func() {
						u.exportForm.apply(u.importForm.settings())
					})
				}),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{} }),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return secondaryButton(gtx, th, theme, &u.testExportBtn, "Test Export", func() {
						u.testProfileConnection(false)
					})
				}),
				layout.Rigid(hgap(theme)),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return secondaryButton(gtx, th, theme, &u.testImportBtn, "Test Import", func() {
						u.testProfileConnection(true)
					})
				}),
				layout.Rigid(hgap(theme)),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return primaryButton(gtx, th, theme, &u.saveProfileBtn, "Save Profile", func() {
						u.saveProfile()
					})
				}),
			)
		}),
	)
}

func (u *UI) profileFromEditors() models.Profile {
	p := u.editingProfile
	p.Name = strings.TrimSpace(editorText(&u.profileName))
	p.Group = strings.TrimSpace(editorText(&u.profileGroup))
	exportSettings := u.exportForm.settings()
	importSettings := mergeImportQuerySettings(u.importForm.settings(), u.queryForm.settings())
	p.ExportSettings = ptrSettings(exportSettings)
	p.ImportSettings = &importSettings
	return withLegacy(p, importSettings)
}

func (u *UI) saveProfile() {
	p := u.profileFromEditors()
	if p.Name == "" {
		u.showError(fmt.Errorf("profile name is required"))
		return
	}
	if err := u.core.SaveProfile(p); err != nil {
		u.showError(err)
		return
	}
	u.view = ViewList
	u.invalidate()
}

func (u *UI) testProfileConnection(useImport bool) {
	base := u.profileFromEditors()
	if strings.TrimSpace(base.Name) == "" {
		base.Name = "Unsaved Profile"
	}
	if strings.TrimSpace(base.Group) == "" {
		base.Group = "Default"
	}
	u.showLoading("Testing connection", "Connecting...")
	go func() {
		err := u.core.TestConnection(withLegacy(base, u.exportForm.settings()), useImport)
		u.invalidate()
		u.closeDialog()
		if err != nil {
			u.showError(err)
			return
		}
		u.showInfo("Connection OK", "Connection test succeeded.")
	}()
}
