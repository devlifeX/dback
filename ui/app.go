package ui

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/png"
	"log"
	"os"
	"strings"
	"sync"
	"time"
	"unsafe"

	coreapp "dback/internal/app"
	"dback/internal/debug"
	"dback/models"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/explorer"
)

type UI struct {
	platform Platform
	core     *coreapp.App
	theme    *AppTheme
	logo     image.Image
	version  string

	window   *app.Window
	explorer *explorer.Explorer

	x11Display           unsafe.Pointer
	x11Window            uintptr
	pendingCenterSize    image.Point
	windowCentered       bool
	windowCenterAttempts int

	section Section
	view    View

	searchEditor  widget.Editor
	search        string
	selectedGroup string

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

	backupTab          int
	settingsTab        int
	selectedBackup     *models.ExportRecord
	selectedBackupID   string
	backupHostFilter   string
	backupHostSelect   widget.Enum
	backupHostDropdown DropdownState
	backupTypeFilter   string
	backupTypeSelect   widget.Enum
	backupTypeDropdown DropdownState
	destSelect         widget.Enum
	destHostDropdown   DropdownState
	backupList         widget.List
	jobsList           widget.List

	includeSecrets   widget.Bool
	passphraseEditor widget.Editor

	dialog DialogState

	pendingPick *PendingFilePick
	pickReqID   string

	jobsMu             sync.Mutex
	jobs               []*operationJob
	jobsUIMu           sync.Mutex
	jobTickerMu        sync.Mutex
	jobTickerRunning   bool
	lastBackupsRefresh time.Time

	navHosts                widget.Clickable
	navBackups              widget.Clickable
	navTemplates            widget.Clickable
	navSettings             widget.Clickable
	navAbout                widget.Clickable
	addHostBtn              widget.Clickable
	addTemplateBtn          widget.Clickable
	saveProfileBtn          widget.Clickable
	saveTemplateBtn         widget.Clickable
	backBtn                 widget.Clickable
	testExportBtn           widget.Clickable
	exportAppDataBtn        widget.Clickable
	importAppDataBtn        widget.Clickable
	tabSettingsExport       widget.Clickable
	tabSettingsSync         widget.Clickable
	saveSyncBtn             widget.Clickable
	testSyncBtn             widget.Clickable
	syncPushBtn             widget.Clickable
	syncPullBtn             widget.Clickable
	syncDestForm            *RemoteDestinationForm
	syncDestinations        []models.RemoteDestination
	syncAppSettingsDestID   string
	syncAppSettingsSelect   *widget.Enum
	syncAppSettingsDropdown DropdownState
	syncShowDestEditor      bool
	syncEditingDestID       string
	syncConnectionOK        bool
	syncPushPending         bool
	syncActivity            models.SyncActivity
	syncAddDestBtn          widget.Clickable
	syncSaveDestBtn         widget.Clickable
	syncCancelDestBtn       widget.Clickable
	syncTestDestBtn         widget.Clickable
	syncDestEditBtns        map[string]*widget.Clickable
	syncDestDeleteBtns      map[string]*widget.Clickable
	settingsList            widget.List
	tabConnection           widget.Clickable
	tabQuery                widget.Clickable
	tabBackupFiles          widget.Clickable
	tabBackupJobs           widget.Clickable
	restoreBtn              widget.Clickable
	verifyBackupBtn         widget.Clickable
	deepVerifySelect        widget.Enum
	deepVerifyDropdown      DropdownState
	openBackupFolderBtn     widget.Clickable
	dialogOKBtn             widget.Clickable
	dialogCancelBtn         widget.Clickable
	dialogSyncPullBtn       widget.Clickable
	dialogForcePushBtn      widget.Clickable
	dialogUploadLatestBtn   widget.Clickable
	dialogUploadAllBtn      widget.Clickable
	dialogHostList          widget.List
	connectionTestCancelBtn widget.Clickable
	connectionTestCloseBtn  widget.Clickable
	connectionTestCopyBtn   widget.Clickable
	deleteTemplateBtn       widget.Clickable
	aboutProjectBtn         widget.Clickable
	aboutCheckUpdateBtn     widget.Clickable

	updateStatus      string
	pendingUpdateInfo coreapp.UpdateInfo
	updateApplyCancel context.CancelFunc

	menuOpenID       string
	backupMenuOpenID string
	menuCloseArea    widget.Clickable

	profileCards   map[string]profileCardWidgets
	groupChips     map[string]*widget.Clickable
	templateRows   map[string]*widget.Clickable
	backupRowMenus map[string]backupRowMenuWidgets
	jobCancelBtns  map[string]*widget.Clickable

	unlocked             bool
	loginFocusPending    bool
	loginPassword        widget.Editor
	loginConfirmPassword widget.Editor
	loginPasswordVisible bool
	loginConfirmVisible  bool
	loginPasswordToggle  widget.Clickable
	loginConfirmToggle   widget.Clickable
	loginError           string
	loginBtn             widget.Clickable
	passphraseVisible    bool
	passphraseToggle     widget.Clickable
	templateCache        templateOptionCache
	backupCache          backupViewCache
	hostConnTest         hostConnectionTestState

	hostUploadMu     sync.Mutex
	hostUploadStates map[string]hostUploadUIState

	pendingRemoteUpload *pendingRemoteUploadChoice

	invalidate func()
}

type profileCardWidgets struct {
	backup       *widget.Clickable
	backupFiles  *widget.Clickable
	uploadRemote *widget.Clickable
	edit         *widget.Clickable
	duplicate    *widget.Clickable
	delete       *widget.Clickable
	more         *widget.Clickable
}

type backupRowMenuWidgets struct {
	more    *widget.Clickable
	import_ *widget.Clickable
	verify  *widget.Clickable
	folder  *widget.Clickable
}

func New(logoPNG []byte, version string) *UI {
	var logo image.Image
	if len(logoPNG) > 0 {
		if img, _, err := image.Decode(bytes.NewReader(logoPNG)); err == nil {
			logo = img
		}
	}
	if strings.TrimSpace(version) == "" {
		version = "dev"
	}
	return &UI{
		platform:          DesktopPlatform{},
		theme:             NewAppTheme(),
		logo:              logo,
		version:           version,
		section:           SectionHosts,
		view:              ViewList,
		profileCards:      make(map[string]profileCardWidgets),
		groupChips:        make(map[string]*widget.Clickable),
		templateRows:      make(map[string]*widget.Clickable),
		backupRowMenus:    make(map[string]backupRowMenuWidgets),
		jobCancelBtns:     make(map[string]*widget.Clickable),
		loginFocusPending: true,
	}
}

func (u *UI) Run() {
	baseDir := u.platform.AppDataDir()
	log.Printf("startup: app data dir = %q", baseDir)

	if _, statErr := os.Stat(baseDir); os.IsNotExist(statErr) {
		log.Printf("startup: data dir does not exist, will create")
	}
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		log.Printf("startup: MkdirAll failed: %v", err)
		panic(err)
	}
	log.Printf("startup: data dir ready")

	var err error
	u.core, err = coreapp.New(baseDir)
	if err != nil {
		log.Printf("startup: coreapp.New failed: %v", err)
		panic(err)
	}
	log.Printf("startup: coreapp initialized (hasVault=%v hasLegacy=%v)", u.core.HasVault(), u.core.HasLegacyPlaintext())
	if debug.Enabled {
		debug.Log("INFO", "startup", "ready", fmt.Sprintf("baseDir=%q vault=%v legacy=%v", baseDir, u.core.HasVault(), u.core.HasLegacyPlaintext()), "", "", "")
	}

	log.Printf("startup: creating window")
	u.window = new(app.Window)
	u.configureWindow()
	u.explorer = explorer.NewExplorer(u.window)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("loop: PANIC: %v\n%s", r, debug.Stack())
			}
		}()
		log.Printf("loop: starting event loop")
		u.loop()
		log.Printf("loop: event loop ended, exiting")
		os.Exit(0)
	}()
	log.Printf("startup: calling app.Main()")
	app.Main()
	log.Printf("startup: app.Main() returned")
}

func (u *UI) loop() {
	var ops op.Ops
	u.invalidate = func() { u.window.Invalidate() }

	for {
		e := u.window.Event()
		u.explorer.ListenEvents(e)
		u.handlePlatformEvent(e)
		switch e := e.(type) {
		case app.DestroyEvent:
			if e.Err != nil {
				log.Printf("loop: DestroyEvent err=%v", e.Err)
				errMsg := e.Err.Error()
				if strings.Contains(errMsg, "egl") || strings.Contains(errMsg, "EGL") || strings.Contains(errMsg, "opengl") || strings.Contains(errMsg, "GL") {
					log.Printf("loop: GPU/EGL initialization failed. Try: LIBGL_ALWAYS_SOFTWARE=1 ./run.sh")
					log.Printf("loop: Or reboot if you recently updated GPU drivers (driver/library version mismatch).")
					_, _ = fmt.Fprintf(os.Stderr, "\n[FATAL] Cannot initialize display renderer: %v\n", e.Err)
					_, _ = fmt.Fprintf(os.Stderr, "  Fix option 1: reboot your system (if GPU driver was recently updated)\n")
					_, _ = fmt.Fprintf(os.Stderr, "  Fix option 2: LIBGL_ALWAYS_SOFTWARE=1 ./run.sh\n")
					_, _ = fmt.Fprintf(os.Stderr, "  Fix option 3: __EGL_VENDOR_LIBRARY_FILENAMES=/usr/share/glvnd/egl_vendor.d/50_mesa.json ./run.sh\n\n")
				}
			} else {
				log.Printf("loop: DestroyEvent (clean exit)")
			}
			return
		case app.ConfigEvent:
			if validWindowSize(e.Config.Size) {
				u.centerWindowForSize(e.Config.Size)
			}
		case app.FrameEvent:
			if !u.windowCentered && validWindowSize(e.Size) {
				u.centerWindowForSize(e.Size)
			}
			gtx := app.NewContext(&ops, e)
			u.layout(gtx)
			e.Frame(gtx.Ops)
		default:
			if debug.Enabled {
				log.Printf("loop: event %T", e)
			}
		}
	}
}

func (u *UI) layout(gtx layout.Context) layout.Dimensions {
	th := u.theme.WithPalette()

	fillRect(gtx, gtx.Constraints.Max, u.theme.Bg)

	if !u.unlocked {
		dims := u.layoutLogin(gtx, th)
		if u.dialog.Kind != DialogNone {
			return layout.Stack{}.Layout(gtx,
				layout.Expanded(func(gtx layout.Context) layout.Dimensions { return dims }),
				layout.Stacked(func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min = gtx.Constraints.Max
					return u.layoutDialog(gtx, th)
				}),
			)
		}
		return dims
	}

	return layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			if u.platform.IsMobile() {
				return u.layoutMobile(gtx, th)
			}
			return u.layoutDesktop(gtx, th)
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			if u.dialog.Kind == DialogNone {
				return layout.Dimensions{}
			}
			gtx.Constraints.Min = gtx.Constraints.Max
			return u.layoutDialog(gtx, th)
		}),
	)
}

func (u *UI) layoutDesktop(gtx layout.Context, th *material.Theme) layout.Dimensions {
	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Max.X = gtx.Dp(248)
			gtx.Constraints.Min.X = gtx.Dp(248)
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
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		if u.section == SectionAbout {
			gtx.Constraints.Min.Y = gtx.Constraints.Max.Y
		}
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
