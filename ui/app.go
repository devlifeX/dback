package ui

import (
	"bytes"
	"image"
	_ "image/png"
	"os"
	"sync"
	"time"

	coreapp "dback/internal/app"
	"dback/models"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/explorer"
)

type UI struct {
	platform Platform
	core     *coreapp.App
	theme    *AppTheme
	logo     image.Image

	window   *app.Window
	explorer *explorer.Explorer

	section Section
	view    View

	searchEditor widget.Editor
	search       string

	editingProfile models.Profile
	profileName    widget.Editor
	profileGroup   widget.Editor
	profileTab     int
	hostForm       *SettingsForm
	queryForm      *QueryForm

	editingTemplate models.SQLTemplate
	templateName    widget.Editor
	templateDesc    widget.Editor
	templateBody    widget.Editor

	backupTab        int
	selectedBackup   *models.ExportRecord
	selectedBackupID string
	destSelect       widget.Enum
	backupList       widget.List
	jobsList         widget.List

	includeSecrets   widget.Bool
	passphraseEditor widget.Editor

	dialog DialogState

	pendingPick *PendingFilePick
	pickReqID   string

	jobsMu             sync.Mutex
	jobs               []*operationJob
	jobsUIMu           sync.Mutex
	lastBackupsRefresh time.Time

	navHosts            widget.Clickable
	navBackups          widget.Clickable
	navTemplates        widget.Clickable
	navSettings         widget.Clickable
	navAbout            widget.Clickable
	addHostBtn          widget.Clickable
	addTemplateBtn      widget.Clickable
	saveProfileBtn      widget.Clickable
	saveTemplateBtn     widget.Clickable
	backBtn             widget.Clickable
	testExportBtn       widget.Clickable
	exportProfilesBtn   widget.Clickable
	importProfilesBtn   widget.Clickable
	tabConnection       widget.Clickable
	tabQuery            widget.Clickable
	tabBackupFiles      widget.Clickable
	tabBackupJobs       widget.Clickable
	importSelectedBtn   widget.Clickable
	openFolderBtn       widget.Clickable
	restoreBtn          widget.Clickable
	openBackupFolderBtn widget.Clickable
	dialogOKBtn         widget.Clickable
	dialogCancelBtn     widget.Clickable
	deleteTemplateBtn   widget.Clickable

	profileCards  map[string]profileCardWidgets
	templateRows  map[string]*widget.Clickable
	backupRows    map[string]*widget.Clickable
	jobCancelBtns map[string]*widget.Clickable

	invalidate func()
}

type profileCardWidgets struct {
	backup    *widget.Clickable
	edit      *widget.Clickable
	duplicate *widget.Clickable
	delete    *widget.Clickable
}

func New(logoPNG []byte) *UI {
	var logo image.Image
	if len(logoPNG) > 0 {
		if img, _, err := image.Decode(bytes.NewReader(logoPNG)); err == nil {
			logo = img
		}
	}
	return &UI{
		platform:      DesktopPlatform{},
		theme:         NewAppTheme(),
		logo:          logo,
		section:       SectionHosts,
		view:          ViewList,
		profileCards:  make(map[string]profileCardWidgets),
		templateRows:  make(map[string]*widget.Clickable),
		backupRows:    make(map[string]*widget.Clickable),
		jobCancelBtns: make(map[string]*widget.Clickable),
	}
}

func (u *UI) Run() {
	baseDir := u.platform.AppDataDir()
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		panic(err)
	}
	var err error
	u.core, err = coreapp.New(baseDir)
	if err != nil {
		panic(err)
	}

	u.window = new(app.Window)
	u.window.Option(
		app.Title("DBack"),
		app.Size(unit.Dp(1200), unit.Dp(800)),
		app.MinSize(unit.Dp(900), unit.Dp(600)),
	)
	u.explorer = explorer.NewExplorer(u.window)

	go func() {
		u.loop()
		os.Exit(0)
	}()
	app.Main()
}

func (u *UI) loop() {
	var ops op.Ops
	u.invalidate = func() { u.window.Invalidate() }

	for {
		e := u.window.Event()
		u.explorer.ListenEvents(e)
		switch e := e.(type) {
		case app.DestroyEvent:
			return
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			u.layout(gtx)
			e.Frame(gtx.Ops)
		}
	}
}

func (u *UI) layout(gtx layout.Context) layout.Dimensions {
	th := u.theme.WithPalette()

	fillRect(gtx, gtx.Constraints.Max, u.theme.Bg)

	if u.dialog.Kind != DialogNone {
		return u.layoutDialog(gtx, th)
	}

	if u.platform.IsMobile() {
		return u.layoutMobile(gtx, th)
	}
	return u.layoutDesktop(gtx, th)
}

func (u *UI) layoutDesktop(gtx layout.Context, th *material.Theme) layout.Dimensions {
	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Max.X = gtx.Dp(220)
			gtx.Constraints.Min.X = gtx.Dp(220)
			return u.layoutSidebar(gtx, th)
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return u.layoutContent(gtx, th)
		}),
	)
}

func (u *UI) layoutMobile(gtx layout.Context, th *material.Theme) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return u.layoutContent(gtx, th)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return u.layoutBottomNav(gtx, th)
		}),
	)
}

func (u *UI) layoutContent(gtx layout.Context, th *material.Theme) layout.Dimensions {
	return layout.Inset{
		Top: u.theme.Padding, Bottom: u.theme.Padding,
		Left: u.theme.Padding, Right: u.theme.Padding,
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		switch u.section {
		case SectionHosts:
			return u.layoutHosts(gtx, th)
		case SectionBackups:
			return u.layoutBackups(gtx, th)
		case SectionTemplates:
			return u.layoutTemplates(gtx, th)
		case SectionSettings:
			return u.layoutSettings(gtx, th)
		case SectionAbout:
			return u.layoutAbout(gtx, th)
		default:
			return layout.Dimensions{}
		}
	})
}

func (u *UI) navigate(section Section) {
	u.section = section
	u.view = ViewList
	u.invalidate()
}

func (u *UI) showDialog(d DialogState) {
	u.dialog = d
	u.invalidate()
}

func (u *UI) closeDialog() {
	u.dialog = DialogState{}
	u.invalidate()
}

func (u *UI) openHosts() {
	u.section = SectionHosts
	u.view = ViewList
	u.invalidate()
}

func (u *UI) openBackups() {
	u.section = SectionBackups
	u.view = ViewList
	u.invalidate()
}
