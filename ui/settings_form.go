package ui

import (
	"fmt"
	"strings"

	"dback/models"

	"gioui.org/layout"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

var (
	connTypeValues = []string{
		string(models.ConnectionTypeSSH),
		string(models.ConnectionTypeJumpHost),
		string(models.ConnectionTypeLocalhost),
		string(models.ConnectionTypeWordPress),
	}
	authTypeValues = []string{string(models.AuthTypePassword), string(models.AuthTypeKeyFile)}
	dbTypeValues   = []string{string(models.DBTypeMySQL), string(models.DBTypeMariaDB)}
)

type SettingsForm struct {
	ConnectionType widget.Enum
	Host           widget.Editor
	Port           widget.Editor
	SSHUser        widget.Editor
	SSHPassword    widget.Editor
	AuthType       widget.Enum
	KeyPath        widget.Editor
	AuthKeyPEM     string
	JumpHost       widget.Editor
	JumpPort       widget.Editor
	JumpUser       widget.Editor
	JumpPassword   widget.Editor
	JumpAuthType   widget.Enum
	JumpKeyPath    widget.Editor
	JumpAuthKeyPEM string
	WPUrl          widget.Editor
	WPKey          widget.Editor
	DBHost         widget.Editor
	DBPort         widget.Editor
	DBUser         widget.Editor
	DBPassword     widget.Editor
	DBType         widget.Enum
	IsDocker       widget.Bool
	ContainerID    widget.Editor
	TargetDB       widget.Editor
	Destination    widget.Editor
	ImportProtected widget.Bool

	defaultDestination string
	scrollList         widget.List

	selectKeyBtn        widget.Clickable
	selectJumpKeyBtn    widget.Clickable
	selectFolderBtn     widget.Clickable
	generateWPKeyBtn    widget.Clickable
	downloadPluginBtn   widget.Clickable
	wpPasswordVisible   bool
	wpPasswordToggle    widget.Clickable
	sshPasswordVisible  bool
	jumpPasswordVisible bool
	dbPasswordVisible   bool
	sshPasswordToggle   widget.Clickable
	jumpPasswordToggle  widget.Clickable
	dbPasswordToggle    widget.Clickable
}

func newSettingsForm(p models.Profile, defaultDest string) *SettingsForm {
	f := &SettingsForm{defaultDestination: defaultDest}
	f.ConnectionType.Value = defaultString(string(p.ConnectionType), string(models.ConnectionTypeSSH))
	setEditorText(&f.Host, p.Host)
	setEditorText(&f.Port, defaultString(p.Port, "22"))
	setEditorText(&f.SSHUser, p.SSHUser)
	setEditorText(&f.SSHPassword, p.SSHPassword)
	f.AuthType.Value = defaultString(string(p.AuthType), string(models.AuthTypePassword))
	setEditorText(&f.KeyPath, p.AuthKeyPath)
	f.AuthKeyPEM = p.AuthKeyPEM
	setEditorText(&f.JumpHost, p.JumpHost)
	setEditorText(&f.JumpPort, defaultString(p.JumpPort, "22"))
	setEditorText(&f.JumpUser, p.JumpUser)
	setEditorText(&f.JumpPassword, p.JumpPassword)
	f.JumpAuthType.Value = defaultString(string(p.JumpAuthType), string(models.AuthTypePassword))
	setEditorText(&f.JumpKeyPath, p.JumpAuthKeyPath)
	f.JumpAuthKeyPEM = p.JumpAuthKeyPEM
	wpURL := p.WPUrl
	if wpURL == "" {
		wpURL = p.Host
	}
	setEditorText(&f.WPUrl, wpURL)
	setEditorText(&f.WPKey, p.WPKey)
	setEditorText(&f.DBHost, defaultString(p.DBHost, "127.0.0.1"))
	setEditorText(&f.DBPort, defaultString(p.DBPort, "3306"))
	setEditorText(&f.DBUser, p.DBUser)
	setEditorText(&f.DBPassword, p.DBPassword)
	f.DBType.Value = defaultString(string(p.DBType), string(models.DBTypeMySQL))
	f.IsDocker.Value = p.IsDocker
	setEditorText(&f.ContainerID, p.ContainerID)
	setEditorText(&f.TargetDB, p.TargetDBName)
	dest := p.Destination
	if strings.TrimSpace(dest) == "" && defaultDest != "" {
		dest = defaultDest
	}
	setEditorText(&f.Destination, dest)
	f.ImportProtected.Value = p.ImportProtected
	return f
}

func (f *SettingsForm) supportsSQLQuery() bool {
	if f.ConnectionType.Value == string(models.ConnectionTypeWordPress) {
		return true
	}
	db := f.DBType.Value
	return db == string(models.DBTypeMySQL) || db == string(models.DBTypeMariaDB)
}

func (f *SettingsForm) profile() models.Profile {
	return models.Profile{
		ConnectionType:  models.ConnectionType(f.ConnectionType.Value),
		Host:            strings.TrimSpace(editorText(&f.Host)),
		Port:            strings.TrimSpace(editorText(&f.Port)),
		SSHUser:         strings.TrimSpace(editorText(&f.SSHUser)),
		SSHPassword:     editorText(&f.SSHPassword),
		AuthType:        models.AuthType(f.AuthType.Value),
		AuthKeyPath:     strings.TrimSpace(editorText(&f.KeyPath)),
		AuthKeyPEM:      f.AuthKeyPEM,
		JumpHost:        strings.TrimSpace(editorText(&f.JumpHost)),
		JumpPort:        strings.TrimSpace(editorText(&f.JumpPort)),
		JumpUser:        strings.TrimSpace(editorText(&f.JumpUser)),
		JumpPassword:    editorText(&f.JumpPassword),
		JumpAuthType:    models.AuthType(f.JumpAuthType.Value),
		JumpAuthKeyPath: strings.TrimSpace(editorText(&f.JumpKeyPath)),
		JumpAuthKeyPEM:  f.JumpAuthKeyPEM,
		WPUrl:           strings.TrimSpace(editorText(&f.WPUrl)),
		WPKey:           editorText(&f.WPKey),
		DBHost:          strings.TrimSpace(editorText(&f.DBHost)),
		DBPort:          strings.TrimSpace(editorText(&f.DBPort)),
		DBUser:          strings.TrimSpace(editorText(&f.DBUser)),
		DBPassword:      editorText(&f.DBPassword),
		DBType:          models.DBType(f.DBType.Value),
		IsDocker:        f.IsDocker.Value,
		ContainerID:     strings.TrimSpace(editorText(&f.ContainerID)),
		TargetDBName:    strings.TrimSpace(editorText(&f.TargetDB)),
		Destination:     strings.TrimSpace(editorText(&f.Destination)),
		ImportProtected:   f.ImportProtected.Value,
	}
}

func (f *SettingsForm) layout(gtx layout.Context, th *material.Theme, theme *AppTheme, u *UI) layout.Dimensions {
	isLocal := f.ConnectionType.Value == string(models.ConnectionTypeLocalhost)
	isWordPress := f.ConnectionType.Value == string(models.ConnectionTypeWordPress)
	isJump := f.ConnectionType.Value == string(models.ConnectionTypeJumpHost)
	useKey := f.AuthType.Value == string(models.AuthTypeKeyFile)
	useJumpKey := f.JumpAuthType.Value == string(models.AuthTypeKeyFile)
	isDocker := f.IsDocker.Value

	return scrollArea(gtx, th, &f.scrollList, func(gtx layout.Context) layout.Dimensions {
		var sections []layout.FlexChild

		sections = append(sections, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return card(gtx, theme, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						lbl := material.Subtitle1(th, "Connection")
						lbl.Color = theme.Text
						return lbl.Layout(gtx)
					}),
					layout.Rigid(vgap(theme)),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return enumField(gtx, th, theme, &f.ConnectionType, "Type", connTypeValues)
					}),
					layout.Rigid(vgap(theme)),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if isLocal {
							return mutedLabel(gtx, th, theme, "Runs mysqldump on this machine — no SSH settings needed.")
						}
						if isWordPress {
							return mutedLabel(gtx, th, theme, "Uses the DBack WordPress plugin over REST — install the generated plugin on the site first.")
						}
						return layout.Dimensions{}
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if !isWordPress {
							return layout.Dimensions{}
						}
						return labeledField(gtx, th, theme, "Site URL", func(gtx layout.Context) layout.Dimensions {
							return editorField(gtx, th, theme, &f.WPUrl, "https://example.com")
						})
					}),
					layout.Rigid(vgap(theme)),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if !isWordPress {
							return layout.Dimensions{}
						}
						return labeledField(gtx, th, theme, "API Key", func(gtx layout.Context) layout.Dimensions {
							return passwordField(gtx, th, theme, &f.WPKey, "", &f.wpPasswordVisible, &f.wpPasswordToggle)
						})
					}),
					layout.Rigid(vgap(theme)),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if !isWordPress {
							return layout.Dimensions{}
						}
						return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return secondaryButton(gtx, th, theme, &f.generateWPKeyBtn, "Generate Token", func() {
									key, err := u.core.GenerateWPKey()
									if err != nil {
										u.showError(err)
										return
									}
									setEditorText(&f.WPKey, key)
									u.invalidate()
								})
							}),
							layout.Rigid(hgap(theme)),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return secondaryButton(gtx, th, theme, &f.downloadPluginBtn, "Download Plugin", func() {
									siteURL := strings.TrimSpace(editorText(&f.WPUrl))
									apiKey := strings.TrimSpace(editorText(&f.WPKey))
									if siteURL == "" {
										u.showError(fmt.Errorf("site URL is required"))
										return
									}
									if apiKey == "" {
										u.showError(fmt.Errorf("generate or enter an API key first"))
										return
									}
									data, filename, err := u.core.BuildWordPressPluginZip(siteURL, apiKey)
									if err != nil {
										u.showError(err)
										return
									}
									u.pickSaveBytes(filename, data, func(path string) {
										u.showInfo("Plugin downloaded", path)
									})
								})
							}),
						)
					}),
					layout.Rigid(vgap(theme)),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if isLocal || isWordPress {
							return layout.Dimensions{}
						}
						return labeledField(gtx, th, theme, "Host", func(gtx layout.Context) layout.Dimensions {
							return editorField(gtx, th, theme, &f.Host, "example.com")
						})
					}),
					layout.Rigid(vgap(theme)),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if isLocal || isWordPress {
							return layout.Dimensions{}
						}
						return labeledField(gtx, th, theme, "Port", func(gtx layout.Context) layout.Dimensions {
							return editorField(gtx, th, theme, &f.Port, "22")
						})
					}),
					layout.Rigid(vgap(theme)),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if isLocal || isWordPress {
							return layout.Dimensions{}
						}
						return labeledField(gtx, th, theme, "SSH User", func(gtx layout.Context) layout.Dimensions {
							return editorField(gtx, th, theme, &f.SSHUser, "root")
						})
					}),
					layout.Rigid(vgap(theme)),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if isLocal || isWordPress || useKey {
							return layout.Dimensions{}
						}
						return labeledField(gtx, th, theme, "SSH Password", func(gtx layout.Context) layout.Dimensions {
							return passwordField(gtx, th, theme, &f.SSHPassword, "", &f.sshPasswordVisible, &f.sshPasswordToggle)
						})
					}),
					layout.Rigid(vgap(theme)),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if isLocal || isWordPress {
							return layout.Dimensions{}
						}
						return enumField(gtx, th, theme, &f.AuthType, "Auth Type", authTypeValues)
					}),
					layout.Rigid(vgap(theme)),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if isLocal || isWordPress || !useKey {
							return layout.Dimensions{}
						}
						return layout.Flex{Axis: layout.Horizontal, Alignment: layout.End}.Layout(gtx,
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								return labeledField(gtx, th, theme, "Key Path", func(gtx layout.Context) layout.Dimensions {
									return editorField(gtx, th, theme, &f.KeyPath, "")
								})
							}),
							layout.Rigid(hgap(theme)),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return secondaryButton(gtx, th, theme, &f.selectKeyBtn, "Select Key", func() {
									u.pickOpenFile(func(path string, data []byte) {
										f.AuthKeyPEM = string(data)
										setEditorText(&f.KeyPath, path)
										u.invalidate()
									})
								})
							}),
						)
					}),
				)
			})
		}))

		if isJump {
			sections = append(sections, layout.Rigid(vgap(theme)))
			sections = append(sections, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return card(gtx, theme, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							lbl := material.Subtitle2(th, "Jump Host")
							lbl.Color = theme.Text
							return lbl.Layout(gtx)
						}),
						layout.Rigid(vgap(theme)),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return labeledField(gtx, th, theme, "Jump Host", func(gtx layout.Context) layout.Dimensions {
								return editorField(gtx, th, theme, &f.JumpHost, "")
							})
						}),
						layout.Rigid(vgap(theme)),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return labeledField(gtx, th, theme, "Jump Port", func(gtx layout.Context) layout.Dimensions {
								return editorField(gtx, th, theme, &f.JumpPort, "22")
							})
						}),
						layout.Rigid(vgap(theme)),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return labeledField(gtx, th, theme, "Jump User", func(gtx layout.Context) layout.Dimensions {
								return editorField(gtx, th, theme, &f.JumpUser, "")
							})
						}),
						layout.Rigid(vgap(theme)),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							if useJumpKey {
								return layout.Dimensions{}
							}
							return labeledField(gtx, th, theme, "Jump Password", func(gtx layout.Context) layout.Dimensions {
								return passwordField(gtx, th, theme, &f.JumpPassword, "", &f.jumpPasswordVisible, &f.jumpPasswordToggle)
							})
						}),
						layout.Rigid(vgap(theme)),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return enumField(gtx, th, theme, &f.JumpAuthType, "Jump Auth Type", authTypeValues)
						}),
						layout.Rigid(vgap(theme)),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							if !useJumpKey {
								return layout.Dimensions{}
							}
							return layout.Flex{Axis: layout.Horizontal, Alignment: layout.End}.Layout(gtx,
								layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
									return labeledField(gtx, th, theme, "Jump Key Path", func(gtx layout.Context) layout.Dimensions {
										return editorField(gtx, th, theme, &f.JumpKeyPath, "")
									})
								}),
								layout.Rigid(hgap(theme)),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return secondaryButton(gtx, th, theme, &f.selectJumpKeyBtn, "Select Jump Key", func() {
										u.pickOpenFile(func(path string, data []byte) {
											f.JumpAuthKeyPEM = string(data)
											setEditorText(&f.JumpKeyPath, path)
											u.invalidate()
										})
									})
								}),
							)
						}),
					)
				})
			}))
		}

		sections = append(sections, layout.Rigid(vgap(theme)))
		sections = append(sections, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return card(gtx, theme, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						lbl := material.Subtitle1(th, "Database")
						lbl.Color = theme.Text
						return lbl.Layout(gtx)
					}),
					layout.Rigid(vgap(theme)),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if !isWordPress {
							return layout.Dimensions{}
						}
						return mutedLabel(gtx, th, theme, "Database credentials come from wp-config.php on the WordPress site.")
					}),
					layout.Rigid(vgap(theme)),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if isWordPress {
							return layout.Dimensions{}
						}
						return checkboxField(gtx, th, theme, &f.IsDocker, "Docker container")
					}),
					layout.Rigid(vgap(theme)),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if isWordPress || !isDocker {
							return layout.Dimensions{}
						}
						return labeledField(gtx, th, theme, "Container ID", func(gtx layout.Context) layout.Dimensions {
							return editorField(gtx, th, theme, &f.ContainerID, "mysql_container")
						})
					}),
					layout.Rigid(vgap(theme)),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if isWordPress {
							return layout.Dimensions{}
						}
						return enumField(gtx, th, theme, &f.DBType, "DB Type", dbTypeValues)
					}),
					layout.Rigid(vgap(theme)),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if isWordPress || isDocker {
							return layout.Dimensions{}
						}
						return labeledField(gtx, th, theme, "DB Host", func(gtx layout.Context) layout.Dimensions {
							return editorField(gtx, th, theme, &f.DBHost, "127.0.0.1")
						})
					}),
					layout.Rigid(vgap(theme)),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if isWordPress || isDocker {
							return layout.Dimensions{}
						}
						return labeledField(gtx, th, theme, "DB Port", func(gtx layout.Context) layout.Dimensions {
							return editorField(gtx, th, theme, &f.DBPort, "3306")
						})
					}),
					layout.Rigid(vgap(theme)),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if isWordPress {
							return layout.Dimensions{}
						}
						return labeledField(gtx, th, theme, "DB User", func(gtx layout.Context) layout.Dimensions {
							return editorField(gtx, th, theme, &f.DBUser, "")
						})
					}),
					layout.Rigid(vgap(theme)),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if isWordPress {
							return layout.Dimensions{}
						}
						return labeledField(gtx, th, theme, "DB Password", func(gtx layout.Context) layout.Dimensions {
							return passwordField(gtx, th, theme, &f.DBPassword, "", &f.dbPasswordVisible, &f.dbPasswordToggle)
						})
					}),
					layout.Rigid(vgap(theme)),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						label := "Database"
						placeholder := ""
						if isWordPress {
							label = "Import Database"
							placeholder = "Leave empty for WordPress default"
						}
						return labeledField(gtx, th, theme, label, func(gtx layout.Context) layout.Dimensions {
							return editorField(gtx, th, theme, &f.TargetDB, placeholder)
						})
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if !isWordPress {
							return layout.Dimensions{}
						}
						return mutedLabel(gtx, th, theme, "Empty uses the WordPress database from wp-config.php. A custom name creates/selects that database before import.")
					}),
				)
			})
		}))

		sections = append(sections, layout.Rigid(vgap(theme)))
		sections = append(sections, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return card(gtx, theme, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						lbl := material.Subtitle1(th, "Backup Files")
						lbl.Color = theme.Text
						return lbl.Layout(gtx)
					}),
					layout.Rigid(vgap(theme)),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal, Alignment: layout.End}.Layout(gtx,
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								return labeledField(gtx, th, theme, "Destination Folder", func(gtx layout.Context) layout.Dimensions {
									return editorField(gtx, th, theme, &f.Destination, f.defaultDestination)
								})
							}),
							layout.Rigid(hgap(theme)),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return secondaryButton(gtx, th, theme, &f.selectFolderBtn, "Browse", func() {
									u.pickFolder(func(path string) {
										if isDocumentURIPath(path) {
											setEditorText(&f.Destination, f.defaultDestination)
										} else {
											setEditorText(&f.Destination, path)
										}
										u.invalidate()
									})
								})
							}),
						)
					}),
				)
			})
		}))

		sections = append(sections, layout.Rigid(vgap(theme)))
		sections = append(sections, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return card(gtx, theme, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						lbl := material.Subtitle1(th, "Security")
						lbl.Color = theme.Text
						return lbl.Layout(gtx)
					}),
					layout.Rigid(vgap(theme)),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return checkboxField(gtx, th, theme, &f.ImportProtected, "Block import to this host")
					}),
					layout.Rigid(vgap(theme)),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return mutedLabel(gtx, th, theme, "When enabled, this host cannot be selected as an import destination. Use this to protect production servers from accidental database overwrites.")
					}),
				)
			})
		}))

		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, sections...)
	})
}
