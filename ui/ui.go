package ui

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"dback/models"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// UI holds the application state and UI components
type UI struct {
	app      fyne.App
	window   fyne.Window
	logs     []models.LogEntry
	logList  *widget.List
	profiles []models.Profile

	// Export Tab Widgets (exposed for Profile saving)
	expConnectionTypeSelect *widget.Select
	expWPUrlEntry           *widget.Entry
	expWPKeyEntry           *widget.Entry
	// expWPPluginPathEntry    *widget.Entry // Removed as not in model

	expHostEntry        *widget.Entry
	expPortEntry        *widget.Entry
	expSSHUserEntry     *widget.Entry
	expSSHPassEntry     *widget.Entry
	expAuthTypeSelect   *widget.Select
	expKeyPathEntry     *widget.Entry
	expDBHostEntry      *widget.Entry
	expDBPortEntry      *widget.Entry
	expDBUserEntry      *widget.Entry
	expDBPassEntry      *widget.Entry
	expDBTypeSelect     *widget.Select
	expIsDockerCheck    *widget.Check
	expContainerIDEntry *widget.Entry
	expTargetDBEntry    *widget.Entry
	expDestPathLabel    *widget.Label // To bind destination

	// Import Tab Widgets
	impConnectionTypeSelect *widget.Select
	impWPUrlEntry           *widget.Entry
	impWPKeyEntry           *widget.Entry
	impHostEntry            *widget.Entry
	impPortEntry            *widget.Entry
	impSSHUserEntry         *widget.Entry
	impSSHPassEntry         *widget.Entry
	impAuthTypeSelect       *widget.Select
	impKeyPathEntry         *widget.Entry
	impDBHostEntry          *widget.Entry
	impDBPortEntry          *widget.Entry
	impDBUserEntry          *widget.Entry
	impDBPassEntry          *widget.Entry
	impDBTypeSelect         *widget.Select
	impIsDockerCheck        *widget.Check
	impContainerIDEntry     *widget.Entry
	impTargetDBEntry        *widget.Entry

	currentTab   string
	historyTable *widget.Table
}

// NewUI creates a new UI instance
func NewUI(app fyne.App) *UI {
	return &UI{
		app:  app,
		logs: []models.LogEntry{},
	}
}

// Run initializes and starts the UI
func (u *UI) Run() {
	u.window = u.app.NewWindow("DB Sync Manager")
	u.window.Resize(fyne.NewSize(1200, 800))
	// u.window.SetFullScreen(true) // Removed per user request (hides controls)

	// Load data
	u.loadProfiles()
	u.loadLogs()

	// Create Tabs
	exportTab := container.NewTabItem("Export (Backup)", u.createExportTab(u.window))
	importTab := container.NewTabItem("Import (Restore)", u.createImportTab(u.window))
	historyTab := container.NewTabItem("History", u.createHistoryTab())
	logsTab := container.NewTabItem("Activity Logs", u.createLogsTab())

	tabs := container.NewAppTabs(exportTab, importTab, historyTab, logsTab)
	tabs.OnSelected = func(t *container.TabItem) {
		u.currentTab = t.Text
	}
	// Default
	u.currentTab = "Export (Backup)"

	// Sidebar (Saved Profiles)
	var sidebar *widget.List
	sidebar = widget.NewList(
		func() int {
			return len(u.profiles)
		},
		func() fyne.CanvasObject {
			label := widget.NewLabel("Profile Name")
			saveBtn := widget.NewButtonWithIcon("", theme.DocumentSaveIcon(), nil)
			duplicateBtn := widget.NewButtonWithIcon("", theme.ContentCopyIcon(), nil)
			renameBtn := widget.NewButtonWithIcon("", theme.DocumentCreateIcon(), nil) // Pencil/Edit icon
			deleteBtn := widget.NewButtonWithIcon("", theme.DeleteIcon(), nil)
			buttons := container.NewHBox(saveBtn, duplicateBtn, renameBtn, deleteBtn)
			return container.NewBorder(nil, nil, nil, buttons, label)
		},
		func(i int, o fyne.CanvasObject) {
			c := o.(*fyne.Container)
			var label *widget.Label
			var btnContainer *fyne.Container

			for _, obj := range c.Objects {
				if l, ok := obj.(*widget.Label); ok {
					label = l
				} else if cont, ok := obj.(*fyne.Container); ok {
					btnContainer = cont
				}
			}

			saveBtn := btnContainer.Objects[0].(*widget.Button)
			duplicateBtn := btnContainer.Objects[1].(*widget.Button)
			renameBtn := btnContainer.Objects[2].(*widget.Button)
			deleteBtn := btnContainer.Objects[3].(*widget.Button)

			p := u.profiles[i]
			label.SetText(p.Name)

			saveBtn.OnTapped = func() {
				if i >= len(u.profiles) {
					return
				}

				// Save based on current tab
				if u.currentTab == "Import (Restore)" {
					u.profiles[i].ConnectionType = models.ConnectionType(u.impConnectionTypeSelect.Selected)
					u.profiles[i].WPUrl = u.impWPUrlEntry.Text
					u.profiles[i].WPKey = u.impWPKeyEntry.Text
					u.profiles[i].Host = u.impHostEntry.Text
					u.profiles[i].Port = u.impPortEntry.Text
					u.profiles[i].SSHUser = u.impSSHUserEntry.Text
					u.profiles[i].SSHPassword = u.impSSHPassEntry.Text
					u.profiles[i].AuthType = models.AuthType(u.impAuthTypeSelect.Selected)
					u.profiles[i].AuthKeyPath = u.impKeyPathEntry.Text
					u.profiles[i].DBHost = u.impDBHostEntry.Text
					u.profiles[i].DBPort = u.impDBPortEntry.Text
					u.profiles[i].DBUser = u.impDBUserEntry.Text
					u.profiles[i].DBPassword = u.impDBPassEntry.Text
					u.profiles[i].DBType = models.DBType(u.impDBTypeSelect.Selected)
					u.profiles[i].IsDocker = u.impIsDockerCheck.Checked
					u.profiles[i].ContainerID = u.impContainerIDEntry.Text
					u.profiles[i].TargetDBName = u.impTargetDBEntry.Text
					// Import tab doesn't have Destination Path binding yet
				} else {
					// Default to Export fields
					u.profiles[i].ConnectionType = models.ConnectionType(u.expConnectionTypeSelect.Selected)
					u.profiles[i].WPUrl = u.expWPUrlEntry.Text
					u.profiles[i].WPKey = u.expWPKeyEntry.Text
					u.profiles[i].Host = u.expHostEntry.Text
					u.profiles[i].Port = u.expPortEntry.Text
					u.profiles[i].SSHUser = u.expSSHUserEntry.Text
					u.profiles[i].SSHPassword = u.expSSHPassEntry.Text
					u.profiles[i].AuthType = models.AuthType(u.expAuthTypeSelect.Selected)
					u.profiles[i].AuthKeyPath = u.expKeyPathEntry.Text
					u.profiles[i].DBHost = u.expDBHostEntry.Text
					u.profiles[i].DBPort = u.expDBPortEntry.Text
					u.profiles[i].DBUser = u.expDBUserEntry.Text
					u.profiles[i].DBPassword = u.expDBPassEntry.Text
					u.profiles[i].DBType = models.DBType(u.expDBTypeSelect.Selected)
					u.profiles[i].IsDocker = u.expIsDockerCheck.Checked
					u.profiles[i].ContainerID = u.expContainerIDEntry.Text
					u.profiles[i].TargetDBName = u.expTargetDBEntry.Text
					u.profiles[i].Destination = u.expDestPathLabel.Text
				}

				u.saveProfiles()
				// Refresh forms to sync both tabs
				u.populateForms(u.profiles[i])
				dialog.ShowInformation("Saved", fmt.Sprintf("Profile '%s' updated from %s tab", p.Name, u.currentTab), u.window)
			}

			duplicateBtn.OnTapped = func() {
				// Clone profile
				newProfile := p
				newProfile.ID = fmt.Sprintf("%d", time.Now().Unix())
				newProfile.Name = p.Name + " (Copy)"

				u.profiles = append(u.profiles, newProfile)
				u.saveProfiles()
				sidebar.Refresh()
				sidebar.Select(len(u.profiles) - 1)
			}

			renameBtn.OnTapped = func() {
				entry := widget.NewEntry()
				entry.SetText(p.Name)
				dialog.ShowCustomConfirm("Rename Profile", "Rename", "Cancel", entry, func(b bool) {
					if b && entry.Text != "" {
						u.profiles[i].Name = entry.Text
						u.saveProfiles()
						sidebar.Refresh()
					}
				}, u.window)
			}

			deleteBtn.OnTapped = func() {
				dialog.ShowConfirm("Delete Profile", fmt.Sprintf("Are you sure you want to delete '%s'?", p.Name), func(b bool) {
					if b {
						// Remove profile
						u.profiles = append(u.profiles[:i], u.profiles[i+1:]...)
						u.saveProfiles()
						sidebar.Refresh()
					}
				}, u.window)
			}
		},
	)

	sidebar.OnSelected = func(id int) {
		p := u.profiles[id]
		u.populateForms(p)
	}

	// New Profile Button
	addProfileBtn := widget.NewButtonWithIcon("New Profile", theme.ContentAddIcon(), func() {
		// Create new empty profile
		nameEntry := widget.NewEntry()
		nameEntry.SetPlaceHolder("Profile Name")

		dialog.ShowCustomConfirm("New Profile", "Create", "Cancel", nameEntry, func(b bool) {
			if b && nameEntry.Text != "" {
				newProfile := models.Profile{
					ID:   fmt.Sprintf("%d", time.Now().Unix()),
					Name: nameEntry.Text,
					// Initialize with current fields or empty? Usually empty or defaults.
					// For convenience, let's use current fields as 'clone' or defaults?
					// User asked for "create new", usually implies blank or current state as template.
					// I'll use current state as template to populate it, but user can clear if they want.
					ConnectionType: models.ConnectionType(u.expConnectionTypeSelect.Selected),
					WPUrl:          u.expWPUrlEntry.Text,
					WPKey:          u.expWPKeyEntry.Text,
					// PluginPath removed

					Host:         u.expHostEntry.Text,
					Port:         u.expPortEntry.Text,
					SSHUser:      u.expSSHUserEntry.Text,
					SSHPassword:  u.expSSHPassEntry.Text,
					AuthType:     models.AuthType(u.expAuthTypeSelect.Selected),
					AuthKeyPath:  u.expKeyPathEntry.Text,
					DBHost:       u.expDBHostEntry.Text,
					DBPort:       u.expDBPortEntry.Text,
					DBUser:       u.expDBUserEntry.Text,
					DBPassword:   u.expDBPassEntry.Text,
					DBType:       models.DBType(u.expDBTypeSelect.Selected),
					IsDocker:     u.expIsDockerCheck.Checked,
					ContainerID:  u.expContainerIDEntry.Text,
					TargetDBName: u.expTargetDBEntry.Text,
					Destination:  u.expDestPathLabel.Text,
				}

				u.profiles = append(u.profiles, newProfile)
				u.saveProfiles()
				sidebar.Refresh()
				// Select the new profile
				sidebar.Select(len(u.profiles) - 1)
			}
		}, u.window)
	})

	sidebarContainer := container.NewBorder(nil, addProfileBtn, nil, nil, sidebar)

	split := container.NewHSplit(
		container.NewBorder(widget.NewLabel("Saved Profiles"), nil, nil, nil, sidebarContainer),
		tabs,
	)
	split.SetOffset(0.25)

	u.window.SetContent(split)
	u.window.ShowAndRun()
}

func (u *UI) loadProfiles() {
	// Load from profiles.json
	file, err := os.Open("profiles.json")
	if err != nil {
		return
	}
	defer file.Close()

	bytes, _ := ioutil.ReadAll(file)
	var config models.AppConfig
	json.Unmarshal(bytes, &config)
	u.profiles = config.Profiles
}

func (u *UI) saveProfiles() {
	config := models.AppConfig{
		Profiles: u.profiles,
	}
	bytes, _ := json.MarshalIndent(config, "", "  ")
	ioutil.WriteFile("profiles.json", bytes, 0644)
}

func (u *UI) showLoading(title, message string) *dialog.CustomDialog {
	content := container.NewVBox(
		widget.NewLabel(message),
		widget.NewProgressBarInfinite(),
	)
	d := dialog.NewCustom(title, "Cancel", content, u.window)
	d.Show()
	return d
}

func (u *UI) showErrorAndLog(title string, err error, action string) {
	if err == nil {
		return
	}
	u.log(action, fmt.Sprintf("%s: %v", title, err), "", "Failed", err.Error())
	dialog.ShowError(err, u.window)
}

func (u *UI) populateForms(p models.Profile) {
	// Export Tab
	u.expConnectionTypeSelect.SetSelected(string(p.ConnectionType))
	u.expWPUrlEntry.SetText(p.WPUrl)
	u.expWPKeyEntry.SetText(p.WPKey)
	u.expHostEntry.SetText(p.Host)
	u.expPortEntry.SetText(p.Port)
	u.expSSHUserEntry.SetText(p.SSHUser)
	u.expSSHPassEntry.SetText(p.SSHPassword)
	u.expAuthTypeSelect.SetSelected(string(p.AuthType))
	u.expKeyPathEntry.SetText(p.AuthKeyPath)
	u.expDBHostEntry.SetText(p.DBHost)
	u.expDBPortEntry.SetText(p.DBPort)
	u.expDBUserEntry.SetText(p.DBUser)
	u.expDBPassEntry.SetText(p.DBPassword)
	u.expDBTypeSelect.SetSelected(string(p.DBType))
	u.expIsDockerCheck.SetChecked(p.IsDocker)
	u.expContainerIDEntry.SetText(p.ContainerID)
	u.expTargetDBEntry.SetText(p.TargetDBName)
	if u.expDestPathLabel != nil {
		u.expDestPathLabel.SetText(p.Destination)
	}

	// Import Tab (if initialized)
	if u.impHostEntry != nil {
		u.impConnectionTypeSelect.SetSelected(string(p.ConnectionType))
		u.impWPUrlEntry.SetText(p.WPUrl)
		u.impWPKeyEntry.SetText(p.WPKey)
		u.impHostEntry.SetText(p.Host)
		u.impPortEntry.SetText(p.Port)
		u.impSSHUserEntry.SetText(p.SSHUser)
		u.impSSHPassEntry.SetText(p.SSHPassword)
		u.impAuthTypeSelect.SetSelected(string(p.AuthType))
		u.impKeyPathEntry.SetText(p.AuthKeyPath)
		u.impDBHostEntry.SetText(p.DBHost)
		u.impDBPortEntry.SetText(p.DBPort)
		u.impDBUserEntry.SetText(p.DBUser)
		u.impDBPassEntry.SetText(p.DBPassword)
		u.impDBTypeSelect.SetSelected(string(p.DBType))
		u.impIsDockerCheck.SetChecked(p.IsDocker)
		u.impContainerIDEntry.SetText(p.ContainerID)
		u.impTargetDBEntry.SetText(p.TargetDBName)
	}
}

func (u *UI) getExecutableDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(exe)
}
