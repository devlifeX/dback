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
	showQuery := u.hostForm.supportsSQLQuery()

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
			if !showQuery {
				return u.hostForm.layout(gtx, th, theme, u)
			}
			return tabBar(gtx, th, theme,
				func(gtx layout.Context) layout.Dimensions {
					return tabButton(gtx, th, theme, &u.tabConnection, "Connection", u.profileTab == 0, func() {
						u.profileTab = 0
						u.invalidate()
					})
				},
				func(gtx layout.Context) layout.Dimensions {
					return tabButton(gtx, th, theme, &u.tabQuery, "Queries", u.profileTab == 1, func() {
						u.profileTab = 1
						u.invalidate()
					})
				},
			)
		}),
		layout.Rigid(vgap(theme)),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			if showQuery && u.profileTab == 1 {
				return u.queryForm.layout(gtx, th, theme, u, u.profileFromEditors, u.core.Templates())
			}
			return u.hostForm.layout(gtx, th, theme, u)
		}),
		layout.Rigid(vgap(theme)),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{} }),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return secondaryButton(gtx, th, theme, &u.testExportBtn, "Test Connection", func() {
						u.testProfileConnection()
					})
				}),
				layout.Rigid(hgap(theme)),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return primaryButton(gtx, th, theme, &u.saveProfileBtn, "Save Host", func() {
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
	host := u.hostForm.profile()
	p.ConnectionType = host.ConnectionType
	p.Host = host.Host
	p.Port = host.Port
	p.SSHUser = host.SSHUser
	p.SSHPassword = host.SSHPassword
	p.AuthType = host.AuthType
	p.AuthKeyPath = host.AuthKeyPath
	p.AuthKeyPEM = host.AuthKeyPEM
	p.JumpHost = host.JumpHost
	p.JumpPort = host.JumpPort
	p.JumpUser = host.JumpUser
	p.JumpPassword = host.JumpPassword
	p.JumpAuthType = host.JumpAuthType
	p.JumpAuthKeyPath = host.JumpAuthKeyPath
	p.JumpAuthKeyPEM = host.JumpAuthKeyPEM
	p.DBHost = host.DBHost
	p.DBPort = host.DBPort
	p.DBUser = host.DBUser
	p.DBPassword = host.DBPassword
	p.DBType = host.DBType
	p.IsDocker = host.IsDocker
	p.ContainerID = host.ContainerID
	p.TargetDBName = host.TargetDBName
	p.Destination = host.Destination
	p.ImportProtected = host.ImportProtected
	qs := u.queryForm.settings()
	p.PreImportQuery = qs.PreImportQuery
	p.RunQueryBeforeImport = qs.RunQueryBeforeImport
	p.PostImportQuery = qs.PostImportQuery
	p.RunQueryAfterImport = qs.RunQueryAfterImport
	return p
}

func (u *UI) saveProfile() {
	p := u.profileFromEditors()
	if p.Name == "" {
		u.showError(fmt.Errorf("host name is required"))
		return
	}
	if err := u.core.SaveProfile(p); err != nil {
		u.showError(err)
		return
	}
	u.view = ViewList
	u.invalidate()
}

func (u *UI) testProfileConnection() {
	p := u.profileFromEditors()
	if strings.TrimSpace(p.Name) == "" {
		p.Name = "Unsaved Host"
	}
	u.showLoading("Testing connection", "Connecting...")
	go func() {
		err := u.core.TestConnection(p)
		u.invalidate()
		u.closeDialog()
		if err != nil {
			u.showError(err)
			return
		}
		u.showInfo("Connection OK", "Connection test succeeded.")
	}()
}
