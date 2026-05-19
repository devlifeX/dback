package ui

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"dback/backend/wordpress"
	coreapp "dback/internal/app"
	"dback/models"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type UI struct {
	app               fyne.App
	window            fyne.Window
	core              *coreapp.App
	content           *fyne.Container
	sidebar           *fyne.Container
	search            *widget.Entry
	selectedProfileID string
	currentSection    string
	backupTab         string
	jobsMu            sync.Mutex
	jobs              []*operationJob
}

type operationJob struct {
	ID          string
	Kind        string
	ProfileName string
	Status      string
	Progress    float64
	Done        bool
	Err         string
	Cancel      context.CancelFunc
}

type tableCell struct {
	label  *widget.Label
	button *widget.Button
	box    *fyne.Container
}

func NewUI(app fyne.App) *UI {
	return &UI{app: app, currentSection: "hosts"}
}

func (u *UI) Run() {
	var err error
	u.core, err = coreapp.New(".")
	if err != nil {
		panic(err)
	}

	u.window = u.app.NewWindow("DBack")
	u.window.Resize(fyne.NewSize(1200, 800))
	u.content = container.NewMax()
	u.sidebar = u.createSidebar()
	if fyne.CurrentDevice().IsMobile() {
		u.window.SetContent(container.NewBorder(nil, u.createBottomNav(), nil, nil, u.content))
	} else {
		u.window.SetContent(container.NewBorder(nil, nil, u.sidebar, nil, u.content))
	}
	u.showHosts()
	u.window.ShowAndRun()
}

func (u *UI) createSidebar() *fyne.Container {
	logoImage := canvas.NewImageFromFile("logo.png")
	logoImage.SetMinSize(fyne.NewSize(56, 56))
	logoImage.FillMode = canvas.ImageFillContain
	logo := container.NewVBox(
		logoImage,
		widget.NewLabelWithStyle("DBack", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
	)
	return container.NewBorder(logo, nil, nil, nil, container.NewVBox(
		u.navButton("Hosts", theme.ComputerIcon(), u.showHosts),
		u.navButton("Backups", theme.StorageIcon(), u.showBackups),
		u.navButton("Settings", theme.SettingsIcon(), u.showSettings),
		u.navButton("About", theme.InfoIcon(), u.showAbout),
	))
}

func (u *UI) createBottomNav() fyne.CanvasObject {
	return container.NewGridWithColumns(4,
		u.navButton("Hosts", theme.ComputerIcon(), u.showHosts),
		u.navButton("Backups", theme.StorageIcon(), u.showBackups),
		u.navButton("Settings", theme.SettingsIcon(), u.showSettings),
		u.navButton("About", theme.InfoIcon(), u.showAbout),
	)
}

func (u *UI) navButton(label string, icon fyne.Resource, tapped func()) *widget.Button {
	return widget.NewButtonWithIcon(label, icon, func() {
		u.currentSection = strings.ToLower(label)
		tapped()
	})
}

func (u *UI) setContent(content fyne.CanvasObject) {
	u.content.Objects = []fyne.CanvasObject{content}
	u.content.Refresh()
}

func (u *UI) cardGrid() *fyne.Container {
	if fyne.CurrentDevice().IsMobile() {
		return container.NewGridWithColumns(1)
	}
	return container.NewAdaptiveGrid(3)
}

func (u *UI) actionBox(objects ...fyne.CanvasObject) fyne.CanvasObject {
	if fyne.CurrentDevice().IsMobile() {
		return container.NewVBox(objects...)
	}
	return container.NewHBox(objects...)
}

func (u *UI) showHosts() {
	u.search = widget.NewEntry()
	u.search.SetPlaceHolder("Search hosts...")
	u.search.OnChanged = func(_ string) {
		u.showHosts()
	}

	profiles := u.filteredProfiles()
	groupCards := u.groupSummaryCards(profiles)
	hostCards := u.cardGrid()
	for _, profile := range profiles {
		p := profile
		hostCards.Add(u.profileCard(p))
	}

	top := container.NewBorder(nil, nil, nil,
		widget.NewButtonWithIcon("Host", theme.ContentAddIcon(), u.showProfileEditor),
		container.NewVBox(widget.NewLabelWithStyle("Hosts", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), u.search),
	)
	u.setContent(container.NewBorder(top, nil, nil, nil, container.NewVScroll(container.NewVBox(
		widget.NewLabel("Groups"),
		groupCards,
		widget.NewSeparator(),
		widget.NewLabel("Hosts"),
		hostCards,
	))))
}

func (u *UI) filteredProfiles() []models.Profile {
	profiles := u.core.Profiles()
	if u.search == nil || strings.TrimSpace(u.search.Text) == "" {
		return profiles
	}
	q := strings.ToLower(strings.TrimSpace(u.search.Text))
	var filtered []models.Profile
	for _, p := range profiles {
		if strings.Contains(strings.ToLower(p.Name), q) || strings.Contains(strings.ToLower(p.Host), q) || strings.Contains(strings.ToLower(p.Group), q) {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

func (u *UI) groupSummaryCards(profiles []models.Profile) fyne.CanvasObject {
	counts := map[string]int{}
	for _, p := range profiles {
		group := p.Group
		if group == "" {
			group = "Default"
		}
		counts[group]++
	}
	grid := u.cardGrid()
	if len(counts) == 0 {
		grid.Add(widget.NewCard("No groups", "", widget.NewLabel("Create a host to start.")))
		return grid
	}
	for group, count := range counts {
		grid.Add(widget.NewCard(group, fmt.Sprintf("%d hosts", count), widget.NewIcon(theme.FolderIcon())))
	}
	return grid
}

func (u *UI) profileCard(p models.Profile) fyne.CanvasObject {
	settings := p.EffectiveExport()
	subtitle := fmt.Sprintf("%s@%s:%s - %s", settings.SSHUser, settings.Host, settings.Port, settings.TargetDBName)
	if settings.ConnectionType == models.ConnectionTypeWordPress {
		subtitle = settings.WPUrl + " - WordPress"
	}
	backupBtn := widget.NewButtonWithIcon("Backup", theme.UploadIcon(), func() {
		u.runBackup(p)
	})
	editBtn := widget.NewButtonWithIcon("Edit", theme.DocumentCreateIcon(), func() {
		u.showProfileEditorWith(p)
	})
	deleteBtn := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
		dialog.ShowConfirm("Delete host", "Delete "+p.Name+"?", func(ok bool) {
			if ok {
				if err := u.core.DeleteProfile(p.ID); err != nil {
					dialog.ShowError(err, u.window)
					return
				}
				u.showHosts()
			}
		}, u.window)
	})
	body := container.NewVBox(
		widget.NewLabel(subtitle),
		u.actionBox(backupBtn, editBtn, deleteBtn),
	)
	return widget.NewCard(p.Name, p.Group, body)
}

func (u *UI) showProfileEditor() {
	p := defaultProfile()
	u.showProfileEditorWith(p)
}

func (u *UI) showProfileEditorWith(p models.Profile) {
	name := widget.NewEntry()
	name.SetText(p.Name)
	group := widget.NewEntry()
	group.SetText(p.Group)
	exportEditor := newSettingsEditor(p.EffectiveExport())
	importEditor := newSettingsEditor(p.EffectiveImport())

	save := widget.NewButtonWithIcon("Save Profile", theme.DocumentSaveIcon(), func() {
		p.Name = strings.TrimSpace(name.Text)
		p.Group = strings.TrimSpace(group.Text)
		if p.Name == "" {
			dialog.ShowError(fmt.Errorf("profile name is required"), u.window)
			return
		}
		exportSettings := exportEditor.settings()
		importSettings := importEditor.settings()
		p.ExportSettings = &exportSettings
		p.ImportSettings = &importSettings
		legacy := withLegacy(p, exportSettings)
		if err := u.core.SaveProfile(legacy); err != nil {
			dialog.ShowError(err, u.window)
			return
		}
		u.showHosts()
	})
	testExport := widget.NewButtonWithIcon("Test Export Connection", theme.ConfirmIcon(), func() {
		u.testProfileConnection(p, name.Text, group.Text, exportEditor, importEditor, false)
	})
	testImport := widget.NewButtonWithIcon("Test Import Connection", theme.ConfirmIcon(), func() {
		u.testProfileConnection(p, name.Text, group.Text, exportEditor, importEditor, true)
	})
	copyExportToImport := widget.NewButton("Copy Export to Import", func() {
		importEditor.apply(exportEditor.settings())
	})
	copyImportToExport := widget.NewButton("Copy Import to Export", func() {
		exportEditor.apply(importEditor.settings())
	})

	tabs := container.NewAppTabs(
		container.NewTabItem("Export", exportEditor.form(u.window)),
		container.NewTabItem("Import", importEditor.form(u.window)),
	)
	u.setContent(container.NewBorder(
		container.NewVBox(widget.NewLabelWithStyle("Host Profile", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), widget.NewForm(
			widget.NewFormItem("Name", name),
			widget.NewFormItem("Group", group),
		)),
		container.NewVBox(
			u.actionBox(copyExportToImport, copyImportToExport),
			u.actionBox(save, testExport, testImport),
		),
		nil,
		nil,
		tabs,
	))
}

func (u *UI) runBackup(p models.Profile) {
	ctx, cancel := context.WithCancel(context.Background())
	job := u.addJob("Backup", p.Name, cancel)
	u.backupTab = "jobs"
	u.showBackups()
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
			u.finishJob(job.ID, "Backup canceled or failed", err)
			return
		}
		u.finishJob(job.ID, "Backup complete: "+filepath.Base(record.FilePath), nil)
		u.showBackups()
	}()
}

func (u *UI) addJob(kind, profileName string, cancel context.CancelFunc) *operationJob {
	job := &operationJob{
		ID:          fmt.Sprintf("%d", time.Now().UnixNano()),
		Kind:        kind,
		ProfileName: profileName,
		Status:      "Starting...",
		Cancel:      cancel,
	}
	u.jobsMu.Lock()
	u.jobs = append([]*operationJob{job}, u.jobs...)
	u.jobsMu.Unlock()
	return job
}

func (u *UI) updateJob(id, status string, progress float64, errText string) {
	u.jobsMu.Lock()
	for _, job := range u.jobs {
		if job.ID == id {
			job.Status = status
			job.Progress = progress
			job.Err = errText
			break
		}
	}
	u.jobsMu.Unlock()
	if u.currentSection == "backups" {
		u.showBackups()
	}
}

func (u *UI) finishJob(id, status string, err error) {
	u.jobsMu.Lock()
	for _, job := range u.jobs {
		if job.ID == id {
			job.Done = true
			job.Status = status
			job.Progress = 1
			if err != nil {
				job.Err = err.Error()
				job.Progress = 0
			}
			break
		}
	}
	u.jobsMu.Unlock()
	if u.currentSection == "backups" {
		u.showBackups()
	}
}

func (u *UI) currentJobs() []*operationJob {
	u.jobsMu.Lock()
	defer u.jobsMu.Unlock()
	jobs := make([]*operationJob, len(u.jobs))
	copy(jobs, u.jobs)
	return jobs
}

func (u *UI) showBackups() {
	u.currentSection = "backups"
	if u.backupTab == "" {
		u.backupTab = "files"
	}
	filesTab := container.NewTabItem("Backup Files", u.createBackupFilesTable())
	jobsTab := container.NewTabItem("Jobs", u.createJobsTable())
	tabs := container.NewAppTabs(filesTab, jobsTab)
	tabs.OnSelected = func(item *container.TabItem) {
		if item == jobsTab {
			u.backupTab = "jobs"
			return
		}
		u.backupTab = "files"
	}
	if u.backupTab == "jobs" {
		tabs.Select(jobsTab)
	} else {
		tabs.Select(filesTab)
	}
	u.setContent(container.NewBorder(
		widget.NewLabelWithStyle("Backups", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		nil, nil, nil,
		tabs,
	))
}

func newTableCell() fyne.CanvasObject {
	label := widget.NewLabel("")
	label.Truncation = fyne.TextTruncateEllipsis
	button := widget.NewButton("", nil)
	button.Hide()
	return container.NewMax(label, button)
}

func unpackTableCell(o fyne.CanvasObject) *tableCell {
	box := o.(*fyne.Container)
	return &tableCell{
		label:  box.Objects[0].(*widget.Label),
		button: box.Objects[1].(*widget.Button),
		box:    box,
	}
}

func setCellLabel(o fyne.CanvasObject, text string, bold bool) {
	cell := unpackTableCell(o)
	cell.button.Hide()
	cell.label.Show()
	cell.label.TextStyle = fyne.TextStyle{Bold: bold}
	cell.label.SetText(text)
}

func setCellButton(o fyne.CanvasObject, text string, enabled bool, tapped func()) {
	cell := unpackTableCell(o)
	cell.label.Hide()
	cell.button.Show()
	cell.button.SetText(text)
	cell.button.OnTapped = tapped
	if enabled {
		cell.button.Enable()
	} else {
		cell.button.Disable()
	}
}

func (u *UI) createBackupFilesTable() fyne.CanvasObject {
	records := u.core.History()
	headers := []string{"Date", "Profile", "Database", "Size", "File", "Action"}
	var selected *models.ExportRecord
	status := widget.NewLabel("Select a backup file to import or open its folder.")
	importBtn := widget.NewButtonWithIcon("Import Selected", theme.DownloadIcon(), func() {
		if selected == nil {
			dialog.ShowInformation("Backup Files", "Select a backup first.", u.window)
			return
		}
		u.showBackupActions(*selected)
	})
	openBtn := widget.NewButtonWithIcon("Open Folder", theme.FolderOpenIcon(), func() {
		if selected == nil {
			dialog.ShowInformation("Backup Files", "Select a backup first.", u.window)
			return
		}
		u.openFolder(filepath.Dir(selected.FilePath))
	})

	table := widget.NewTable(
		func() (int, int) { return len(records) + 1, len(headers) },
		newTableCell,
		func(id widget.TableCellID, o fyne.CanvasObject) {
			if id.Row == 0 {
				setCellLabel(o, headers[id.Col], true)
				return
			}
			record := records[len(records)-id.Row]
			switch id.Col {
			case 0:
				setCellLabel(o, record.ExportDate.Format("2006-01-02 15:04"), false)
			case 1:
				setCellLabel(o, record.ProfileName, false)
			case 2:
				setCellLabel(o, record.DatabaseName, false)
			case 3:
				setCellLabel(o, record.FileSize, false)
			case 4:
				setCellLabel(o, filepath.Base(record.FilePath), false)
			case 5:
				setCellButton(o, "Import", true, func() {
					u.showBackupActions(record)
				})
			}
		},
	)
	table.SetColumnWidth(0, 145)
	table.SetColumnWidth(1, 170)
	table.SetColumnWidth(2, 130)
	table.SetColumnWidth(3, 80)
	table.SetColumnWidth(4, 260)
	table.SetColumnWidth(5, 90)
	table.SetRowHeight(0, 34)
	table.OnSelected = func(id widget.TableCellID) {
		if id.Row == 0 {
			return
		}
		record := records[len(records)-id.Row]
		selected = &record
		status.SetText(filepath.Base(record.FilePath))
	}

	return container.NewBorder(nil, container.NewVBox(status, u.actionBox(importBtn, openBtn)), nil, nil, table)
}

func (u *UI) createJobsTable() fyne.CanvasObject {
	jobs := u.currentJobs()
	headers := []string{"Type", "Profile", "Status", "Progress", "Action"}
	table := widget.NewTable(
		func() (int, int) { return len(jobs) + 1, len(headers) },
		newTableCell,
		func(id widget.TableCellID, o fyne.CanvasObject) {
			if id.Row == 0 {
				setCellLabel(o, headers[id.Col], true)
				return
			}
			job := jobs[id.Row-1]
			status := job.Status
			if job.Err != "" {
				status += " - " + job.Err
			}
			switch id.Col {
			case 0:
				setCellLabel(o, job.Kind, false)
			case 1:
				setCellLabel(o, job.ProfileName, false)
			case 2:
				setCellLabel(o, status, false)
			case 3:
				setCellLabel(o, fmt.Sprintf("%.0f%%", job.Progress*100), false)
			case 4:
				setCellButton(o, "Cancel", !job.Done, func() {
					if job.Cancel != nil && !job.Done {
						job.Cancel()
						u.updateJob(job.ID, "Canceling...", job.Progress, "")
					}
				})
			}
		},
	)
	table.SetColumnWidth(0, 80)
	table.SetColumnWidth(1, 180)
	table.SetColumnWidth(2, 520)
	table.SetColumnWidth(3, 80)
	table.SetColumnWidth(4, 90)
	table.SetRowHeight(0, 34)

	return container.NewBorder(nil, nil, nil, nil, table)
}

func (u *UI) showBackupActions(record models.ExportRecord) {
	profiles := u.core.Profiles()
	names := make([]string, 0, len(profiles))
	byName := map[string]models.Profile{}
	for _, p := range profiles {
		label := p.Name
		names = append(names, label)
		byName[label] = p
	}
	dest := widget.NewSelect(names, nil)
	if len(names) > 0 {
		dest.SetSelected(names[0])
	}
	progress := widget.NewProgressBar()
	status := widget.NewLabel("Choose a destination profile.")
	restoreBtn := widget.NewButtonWithIcon("Import to Selected Host", theme.DownloadIcon(), func() {
		p, ok := byName[dest.Selected]
		if !ok {
			dialog.ShowError(fmt.Errorf("select a destination profile"), u.window)
			return
		}
		ctx, cancel := context.WithCancel(context.Background())
		job := u.addJob("Import", p.Name, cancel)
		u.backupTab = "jobs"
		u.showBackups()
		go func() {
			defer cancel()
			err := u.core.Restore(ctx, record, p, func(message string, current int64, total int64) {
				if total > 0 {
					u.updateJob(job.ID, message, float64(current)/float64(total), "")
				} else {
					u.updateJob(job.ID, message, job.Progress, "")
				}
			})
			if err != nil {
				if errors.Is(err, context.Canceled) {
					u.finishJob(job.ID, "Import canceled", nil)
					return
				}
				u.finishJob(job.ID, "Import canceled or failed", err)
				return
			}
			u.finishJob(job.ID, "Import complete", nil)
			u.showBackups()
		}()
	})
	openBtn := widget.NewButtonWithIcon("Open Folder", theme.FolderOpenIcon(), func() {
		u.openFolder(filepath.Dir(record.FilePath))
	})
	u.setContent(container.NewBorder(
		widget.NewLabelWithStyle("Backup Detail", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewButtonWithIcon("Back", theme.NavigateBackIcon(), u.showBackups),
		nil, nil,
		container.NewVBox(
			widget.NewCard(record.ProfileName, record.FileSize, widget.NewLabel(record.FilePath)),
			widget.NewForm(widget.NewFormItem("Destination", dest)),
			u.actionBox(restoreBtn, openBtn),
			status,
			progress,
		),
	))
}

func (u *UI) testProfileConnection(base models.Profile, name, group string, exportEditor, importEditor *settingsEditor, useImport bool) {
	exportSettings := exportEditor.settings()
	importSettings := importEditor.settings()
	base.Name = defaultString(strings.TrimSpace(name), "Unsaved Profile")
	base.Group = defaultString(strings.TrimSpace(group), "Default")
	base.ExportSettings = &exportSettings
	base.ImportSettings = &importSettings

	loading := dialog.NewCustomWithoutButtons("Testing connection", container.NewVBox(
		widget.NewLabel("Connecting..."),
		widget.NewProgressBarInfinite(),
	), u.window)
	loading.Show()
	go func() {
		err := u.core.TestConnection(withLegacy(base, exportSettings), useImport)
		loading.Hide()
		if err != nil {
			dialog.ShowError(err, u.window)
			return
		}
		dialog.ShowInformation("Connection OK", "Connection test succeeded.", u.window)
	}()
}

func (u *UI) showSettings() {
	includeSecrets := widget.NewCheck("Include saved passwords/API keys", nil)
	exportBtn := widget.NewButtonWithIcon("Export Profiles", theme.UploadIcon(), func() {
		fd := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
			if err != nil || writer == nil {
				return
			}
			path := writer.URI().Path()
			_ = writer.Close()
			if err := u.core.ExportProfiles(path, includeSecrets.Checked); err != nil {
				dialog.ShowError(err, u.window)
				return
			}
			dialog.ShowInformation("Export complete", path, u.window)
		}, u.window)
		fd.SetFileName("dback-profiles.json")
		fd.Show()
	})
	importBtn := widget.NewButtonWithIcon("Import Profiles", theme.DownloadIcon(), func() {
		fd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil || reader == nil {
				return
			}
			path := reader.URI().Path()
			_ = reader.Close()
			if err := u.core.ImportProfiles(path, includeSecrets.Checked); err != nil {
				dialog.ShowError(err, u.window)
				return
			}
			dialog.ShowInformation("Import complete", "Profiles imported.", u.window)
			u.showHosts()
		}, u.window)
		fd.Show()
	})
	u.setContent(container.NewVBox(
		widget.NewLabelWithStyle("Settings", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewCard("Profile Transfer", "Move or backup all host profiles.", container.NewVBox(
			includeSecrets,
			u.actionBox(exportBtn, importBtn),
			widget.NewLabel("Passwords and API keys are excluded unless you explicitly include them."),
		)),
		widget.NewButtonWithIcon("About", theme.InfoIcon(), u.showAbout),
	))
}

func (u *UI) showAbout() {
	githubURL, _ := url.Parse("https://github.com/devlifeX/dback")
	logo := canvas.NewImageFromFile("logo.png")
	logo.SetMinSize(fyne.NewSize(96, 96))
	logo.FillMode = canvas.ImageFillContain

	u.setContent(container.NewCenter(widget.NewCard("About DBack", "DB Sync Manager", container.NewVBox(
		container.NewCenter(logo),
		widget.NewLabelWithStyle("dariush vesal", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("dariush.vesal@gmail.com", fyne.TextAlignCenter, fyne.TextStyle{}),
		container.NewCenter(widget.NewHyperlink("https://github.com/devlifeX/dback", githubURL)),
	))))
}

func (u *UI) getExecutableDir() string {
	cwd, err := os.Getwd()
	if err == nil && cwd != "" {
		return cwd
	}
	exe, err := os.Executable()
	if err == nil {
		dir := filepath.Dir(exe)
		if !strings.Contains(dir, "go-build") && !strings.Contains(dir, "/tmp/") {
			return dir
		}
	}
	home, err := os.UserHomeDir()
	if err == nil {
		return home
	}

	return "."
}

type settingsEditor struct {
	connectionType *widget.Select
	host           *widget.Entry
	port           *widget.Entry
	sshUser        *widget.Entry
	sshPassword    *widget.Entry
	authType       *widget.Select
	keyPath        *widget.Entry
	jumpHost       *widget.Entry
	jumpPort       *widget.Entry
	jumpUser       *widget.Entry
	jumpPassword   *widget.Entry
	jumpAuthType   *widget.Select
	jumpKeyPath    *widget.Entry
	wpURL          *widget.Entry
	wpKey          *widget.Entry
	dbHost         *widget.Entry
	dbPort         *widget.Entry
	dbUser         *widget.Entry
	dbPassword     *widget.Entry
	dbType         *widget.Select
	isDocker       *widget.Check
	containerID    *widget.Entry
	targetDB       *widget.Entry
	destination    *widget.Entry
	refresh        func()
}

func newSettingsEditor(p models.Profile) *settingsEditor {
	e := &settingsEditor{
		connectionType: widget.NewSelect([]string{string(models.ConnectionTypeSSH), string(models.ConnectionTypeJumpHost), string(models.ConnectionTypeWordPress)}, nil),
		host:           widget.NewEntry(),
		port:           widget.NewEntry(),
		sshUser:        widget.NewEntry(),
		sshPassword:    widget.NewPasswordEntry(),
		authType:       widget.NewSelect([]string{string(models.AuthTypePassword), string(models.AuthTypeKeyFile)}, nil),
		keyPath:        widget.NewEntry(),
		jumpHost:       widget.NewEntry(),
		jumpPort:       widget.NewEntry(),
		jumpUser:       widget.NewEntry(),
		jumpPassword:   widget.NewPasswordEntry(),
		jumpAuthType:   widget.NewSelect([]string{string(models.AuthTypePassword), string(models.AuthTypeKeyFile)}, nil),
		jumpKeyPath:    widget.NewEntry(),
		wpURL:          widget.NewEntry(),
		wpKey:          widget.NewPasswordEntry(),
		dbHost:         widget.NewEntry(),
		dbPort:         widget.NewEntry(),
		dbUser:         widget.NewEntry(),
		dbPassword:     widget.NewPasswordEntry(),
		dbType:         widget.NewSelect([]string{string(models.DBTypeMySQL), string(models.DBTypeMariaDB), string(models.DBTypePostgreSQL), string(models.DBTypeCouchDB)}, nil),
		isDocker:       widget.NewCheck("Docker container", nil),
		containerID:    widget.NewEntry(),
		targetDB:       widget.NewEntry(),
		destination:    widget.NewEntry(),
	}
	e.connectionType.SetSelected(defaultString(string(p.ConnectionType), string(models.ConnectionTypeSSH)))
	e.host.SetText(p.Host)
	e.port.SetText(defaultString(p.Port, "22"))
	e.sshUser.SetText(p.SSHUser)
	e.sshPassword.SetText(p.SSHPassword)
	e.authType.SetSelected(defaultString(string(p.AuthType), string(models.AuthTypePassword)))
	e.keyPath.SetText(p.AuthKeyPath)
	e.jumpHost.SetText(p.JumpHost)
	e.jumpPort.SetText(defaultString(p.JumpPort, "22"))
	e.jumpUser.SetText(p.JumpUser)
	e.jumpPassword.SetText(p.JumpPassword)
	e.jumpAuthType.SetSelected(defaultString(string(p.JumpAuthType), string(models.AuthTypePassword)))
	e.jumpKeyPath.SetText(p.JumpAuthKeyPath)
	e.wpURL.SetText(p.WPUrl)
	e.wpKey.SetText(p.WPKey)
	e.dbHost.SetText(defaultString(p.DBHost, "127.0.0.1"))
	e.dbPort.SetText(defaultString(p.DBPort, "3306"))
	e.dbUser.SetText(p.DBUser)
	e.dbPassword.SetText(p.DBPassword)
	e.dbType.SetSelected(defaultString(string(p.DBType), string(models.DBTypeMySQL)))
	e.isDocker.SetChecked(p.IsDocker)
	e.containerID.SetText(p.ContainerID)
	e.targetDB.SetText(p.TargetDBName)
	e.destination.SetText(p.Destination)
	return e
}

func (e *settingsEditor) form(w fyne.Window) fyne.CanvasObject {
	keyBtn := widget.NewButton("Select Key", func() {
		fd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err == nil && reader != nil {
				e.keyPath.SetText(reader.URI().Path())
				_ = reader.Close()
			}
		}, w)
		fd.Show()
	})
	jumpKeyBtn := widget.NewButton("Select Jump Key", func() {
		fd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err == nil && reader != nil {
				e.jumpKeyPath.SetText(reader.URI().Path())
				_ = reader.Close()
			}
		}, w)
		fd.Show()
	})
	folderBtn := widget.NewButton("Select Destination", func() {
		fd := dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
			if err == nil && uri != nil {
				e.destination.SetText(uri.Path())
			}
		}, w)
		fd.Show()
	})
	generatePluginBtn := widget.NewButton("Generate WordPress Plugin", func() {
		fd := dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil || uri == nil {
				return
			}
			key, path, err := wordpress.GeneratePlugin("plugin_template/dback-sync.php", uri.Path())
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			e.wpKey.SetText(key)
			dialog.ShowInformation("Plugin Generated", path, w)
		}, w)
		fd.Show()
	})
	sshRows := []fyne.CanvasObject{
		labeledField("Target SSH Host", e.host),
		labeledField("Port", e.port),
		labeledField("SSH User", e.sshUser),
		labeledField("SSH Password", e.sshPassword),
		labeledField("Auth Type", e.authType),
		labeledField("Key Path", container.NewBorder(nil, nil, nil, keyBtn, e.keyPath)),
	}
	jumpRows := []fyne.CanvasObject{
		labeledField("Jump Host", e.jumpHost),
		labeledField("Jump Port", e.jumpPort),
		labeledField("Jump User", e.jumpUser),
		labeledField("Jump Password", e.jumpPassword),
		labeledField("Jump Auth Type", e.jumpAuthType),
		labeledField("Jump Key Path", container.NewBorder(nil, nil, nil, jumpKeyBtn, e.jumpKeyPath)),
	}
	wpRows := []fyne.CanvasObject{
		labeledField("WordPress URL", e.wpURL),
		labeledField("WordPress API Key", e.wpKey),
		generatePluginBtn,
	}
	dbRows := []fyne.CanvasObject{
		e.isDocker,
		labeledField("Container", e.containerID),
		labeledField("DB Type", e.dbType),
		labeledField("DB Host", e.dbHost),
		labeledField("DB Port", e.dbPort),
		labeledField("DB User", e.dbUser),
		labeledField("DB Password", e.dbPassword),
		labeledField("Database", e.targetDB),
	}
	sshGrid := responsiveGrid(sshRows...)
	jumpGrid := responsiveGrid(jumpRows...)
	wpGrid := responsiveGrid(wpRows...)
	dbCard := widget.NewCard("Database", "", responsiveGrid(dbRows...))
	keyRow := sshRows[5]
	passwordRow := sshRows[3]
	jumpKeyRow := jumpRows[5]
	jumpPasswordRow := jumpRows[3]

	e.refresh = func() {
		isWP := e.connectionType.Selected == string(models.ConnectionTypeWordPress)
		isJump := e.connectionType.Selected == string(models.ConnectionTypeJumpHost)
		if isWP {
			sshGrid.Hide()
			jumpGrid.Hide()
			wpGrid.Show()
			dbCard.Hide()
		} else {
			sshGrid.Show()
			if isJump {
				jumpGrid.Show()
			} else {
				jumpGrid.Hide()
			}
			wpGrid.Hide()
			dbCard.Show()
		}
		if e.authType.Selected == string(models.AuthTypeKeyFile) {
			keyRow.Show()
			passwordRow.Hide()
		} else {
			keyRow.Hide()
			passwordRow.Show()
		}
		if e.jumpAuthType.Selected == string(models.AuthTypeKeyFile) {
			jumpKeyRow.Show()
			jumpPasswordRow.Hide()
		} else {
			jumpKeyRow.Hide()
			jumpPasswordRow.Show()
		}
		sshGrid.Refresh()
		jumpGrid.Refresh()
		wpGrid.Refresh()
		dbCard.Refresh()
	}
	e.connectionType.OnChanged = func(string) { e.refresh() }
	e.authType.OnChanged = func(string) { e.refresh() }
	e.jumpAuthType.OnChanged = func(string) { e.refresh() }
	e.refresh()

	return container.NewVScroll(container.NewVBox(
		widget.NewCard("Connection", "", container.NewVBox(
			responsiveGrid(labeledField("Type", e.connectionType)),
			sshGrid,
			jumpGrid,
			wpGrid,
		)),
		dbCard,
		widget.NewCard("Files", "", responsiveGrid(
			labeledField("Destination Folder", container.NewBorder(nil, nil, nil, folderBtn, e.destination)),
		)),
	))
}

func labeledField(label string, object fyne.CanvasObject) fyne.CanvasObject {
	return container.NewBorder(widget.NewLabel(label), nil, nil, nil, object)
}

func responsiveGrid(items ...fyne.CanvasObject) *fyne.Container {
	if fyne.CurrentDevice().IsMobile() {
		return container.NewGridWithColumns(1, items...)
	}
	return container.NewGridWithColumns(2, items...)
}

func (e *settingsEditor) settings() models.TransferSettings {
	return models.TransferSettings{
		ConnectionType:  models.ConnectionType(e.connectionType.Selected),
		Host:            strings.TrimSpace(e.host.Text),
		Port:            strings.TrimSpace(e.port.Text),
		SSHUser:         strings.TrimSpace(e.sshUser.Text),
		SSHPassword:     e.sshPassword.Text,
		AuthType:        models.AuthType(e.authType.Selected),
		AuthKeyPath:     strings.TrimSpace(e.keyPath.Text),
		JumpHost:        strings.TrimSpace(e.jumpHost.Text),
		JumpPort:        strings.TrimSpace(e.jumpPort.Text),
		JumpUser:        strings.TrimSpace(e.jumpUser.Text),
		JumpPassword:    e.jumpPassword.Text,
		JumpAuthType:    models.AuthType(e.jumpAuthType.Selected),
		JumpAuthKeyPath: strings.TrimSpace(e.jumpKeyPath.Text),
		WPUrl:           strings.TrimSpace(e.wpURL.Text),
		WPKey:           e.wpKey.Text,
		DBHost:          strings.TrimSpace(e.dbHost.Text),
		DBPort:          strings.TrimSpace(e.dbPort.Text),
		DBUser:          strings.TrimSpace(e.dbUser.Text),
		DBPassword:      e.dbPassword.Text,
		DBType:          models.DBType(e.dbType.Selected),
		IsDocker:        e.isDocker.Checked,
		ContainerID:     strings.TrimSpace(e.containerID.Text),
		TargetDBName:    strings.TrimSpace(e.targetDB.Text),
		Destination:     strings.TrimSpace(e.destination.Text),
	}
}

func (e *settingsEditor) apply(settings models.TransferSettings) {
	e.connectionType.SetSelected(defaultString(string(settings.ConnectionType), string(models.ConnectionTypeSSH)))
	e.host.SetText(settings.Host)
	e.port.SetText(defaultString(settings.Port, "22"))
	e.sshUser.SetText(settings.SSHUser)
	e.sshPassword.SetText(settings.SSHPassword)
	e.authType.SetSelected(defaultString(string(settings.AuthType), string(models.AuthTypePassword)))
	e.keyPath.SetText(settings.AuthKeyPath)
	e.jumpHost.SetText(settings.JumpHost)
	e.jumpPort.SetText(defaultString(settings.JumpPort, "22"))
	e.jumpUser.SetText(settings.JumpUser)
	e.jumpPassword.SetText(settings.JumpPassword)
	e.jumpAuthType.SetSelected(defaultString(string(settings.JumpAuthType), string(models.AuthTypePassword)))
	e.jumpKeyPath.SetText(settings.JumpAuthKeyPath)
	e.wpURL.SetText(settings.WPUrl)
	e.wpKey.SetText(settings.WPKey)
	e.dbHost.SetText(defaultString(settings.DBHost, "127.0.0.1"))
	e.dbPort.SetText(defaultString(settings.DBPort, "3306"))
	e.dbUser.SetText(settings.DBUser)
	e.dbPassword.SetText(settings.DBPassword)
	e.dbType.SetSelected(defaultString(string(settings.DBType), string(models.DBTypeMySQL)))
	e.isDocker.SetChecked(settings.IsDocker)
	e.containerID.SetText(settings.ContainerID)
	e.targetDB.SetText(settings.TargetDBName)
	e.destination.SetText(settings.Destination)
	if e.refresh != nil {
		e.refresh()
	}
}

func defaultProfile() models.Profile {
	id := fmt.Sprintf("%d", time.Now().UnixNano())
	p := models.Profile{
		ID:             id,
		Name:           "New Host",
		Group:          "Default",
		ConnectionType: models.ConnectionTypeSSH,
		Port:           "22",
		AuthType:       models.AuthTypePassword,
		JumpPort:       "22",
		JumpAuthType:   models.AuthTypePassword,
		DBHost:         "127.0.0.1",
		DBPort:         "3306",
		DBType:         models.DBTypeMySQL,
		Destination:    ".",
	}
	settings := models.SettingsFromProfile(p)
	p.ExportSettings = &settings
	p.ImportSettings = &settings
	return p
}

func withLegacy(p models.Profile, settings models.TransferSettings) models.Profile {
	p.ConnectionType = settings.ConnectionType
	p.Host = settings.Host
	p.Port = settings.Port
	p.SSHUser = settings.SSHUser
	p.SSHPassword = settings.SSHPassword
	p.AuthType = settings.AuthType
	p.AuthKeyPath = settings.AuthKeyPath
	p.JumpHost = settings.JumpHost
	p.JumpPort = settings.JumpPort
	p.JumpUser = settings.JumpUser
	p.JumpPassword = settings.JumpPassword
	p.JumpAuthType = settings.JumpAuthType
	p.JumpAuthKeyPath = settings.JumpAuthKeyPath
	p.WPUrl = settings.WPUrl
	p.WPKey = settings.WPKey
	p.DBHost = settings.DBHost
	p.DBPort = settings.DBPort
	p.DBUser = settings.DBUser
	p.DBPassword = settings.DBPassword
	p.DBType = settings.DBType
	p.IsDocker = settings.IsDocker
	p.ContainerID = settings.ContainerID
	p.TargetDBName = settings.TargetDBName
	p.Destination = settings.Destination
	return p
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func (u *UI) openFolder(path string) {
	if path == "" {
		return
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", path)
	case "windows":
		cmd = exec.Command("explorer", path)
	default:
		cmd = exec.Command("xdg-open", path)
	}
	if err := cmd.Start(); err != nil {
		dialog.ShowInformation("Folder", path, u.window)
	}
}
