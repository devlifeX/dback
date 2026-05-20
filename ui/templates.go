package ui

import (
	"fmt"
	"strings"
	"time"

	"dback/models"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

var templatesList widget.List

func (u *UI) layoutTemplates(gtx layout.Context, th *material.Theme) layout.Dimensions {
	theme := u.theme

	if u.view == ViewTemplateEditor {
		return u.layoutTemplateEditor(gtx, th)
	}

	templates := u.core.Templates()
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return sectionTitle(gtx, th, theme, "SQL Templates")
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return primaryButton(gtx, th, theme, &u.addTemplateBtn, "+ Template", func() {
						u.openTemplateEditor(models.SQLTemplate{
							ID:   fmt.Sprintf("%d", time.Now().UnixNano()),
							Name: "New Template",
						})
					})
				}),
			)
		}),
		layout.Rigid(vgap(theme)),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return scrollArea(gtx, th, &templatesList, func(gtx layout.Context) layout.Dimensions {
				var rows []layout.FlexChild
				for _, t := range templates {
					t := t
					btn, ok := u.templateRows[t.ID]
					if !ok {
						btn = new(widget.Clickable)
						u.templateRows[t.ID] = btn
					}
					editBtn := btn
					rows = append(rows, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return card(gtx, theme, func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
								layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
									return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											lbl := material.Body1(th, t.Name)
											lbl.Color = theme.Text
											return lbl.Layout(gtx)
										}),
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											return mutedLabel(gtx, th, theme, t.Description)
										}),
									)
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return secondaryButton(gtx, th, theme, editBtn, "Edit", func() {
										u.openTemplateEditor(t)
									})
								}),
							)
						})
					}))
					rows = append(rows, layout.Rigid(vgap(theme)))
				}
				if len(templates) == 0 {
					rows = append(rows, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return mutedLabel(gtx, th, theme, "No templates yet.")
					}))
				}
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx, rows...)
			})
		}),
	)
}

func (u *UI) openTemplateEditor(t models.SQLTemplate) {
	u.editingTemplate = t
	setEditorText(&u.templateName, t.Name)
	setEditorText(&u.templateDesc, t.Description)
	setEditorText(&u.templateBody, t.Body)
	u.view = ViewTemplateEditor
	u.invalidate()
}

func (u *UI) layoutTemplateEditor(gtx layout.Context, th *material.Theme) layout.Dimensions {
	theme := u.theme
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
					return sectionTitle(gtx, th, theme, "Edit Template")
				}),
			)
		}),
		layout.Rigid(vgap(theme)),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return labeledField(gtx, th, theme, "Name", func(gtx layout.Context) layout.Dimensions {
				return editorField(gtx, th, theme, &u.templateName, "Template name")
			})
		}),
		layout.Rigid(vgap(theme)),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return labeledField(gtx, th, theme, "Description", func(gtx layout.Context) layout.Dimensions {
				return editorField(gtx, th, theme, &u.templateDesc, "Optional description")
			})
		}),
		layout.Rigid(vgap(theme)),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return mutedLabel(gtx, th, theme, "Placeholders: {databasename}, {host}, {profile}, {dbuser}")
		}),
		layout.Rigid(vgap(theme)),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.Y = gtx.Dp(unit.Dp(200))
			return editorMultiline(gtx, th, theme, &u.templateBody, "SQL body...")
		}),
		layout.Rigid(vgap(theme)),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return dangerButton(gtx, th, theme, &u.deleteTemplateBtn, "Delete", func() {
						id := u.editingTemplate.ID
						u.showConfirm("Delete template", "Delete this template?", func() {
							_ = u.core.DeleteTemplate(id)
							u.view = ViewList
							u.invalidate()
						})
					})
				}),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{} }),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return primaryButton(gtx, th, theme, &u.saveTemplateBtn, "Save", func() {
						t := u.editingTemplate
						t.Name = strings.TrimSpace(editorText(&u.templateName))
						t.Description = strings.TrimSpace(editorText(&u.templateDesc))
						t.Body = editorText(&u.templateBody)
						if t.Name == "" {
							u.showError(fmt.Errorf("template name is required"))
							return
						}
						if err := u.core.SaveTemplate(t); err != nil {
							u.showError(err)
							return
						}
						u.view = ViewList
						u.invalidate()
					})
				}),
			)
		}),
	)
}
