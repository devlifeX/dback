package ui

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"dback/backend/db"
	"dback/backend/ssh"
	"dback/backend/wordpress"
	"dback/models"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func (u *UI) createExportTab(w fyne.Window) fyne.CanvasObject {
	// --- Connection Type ---
	u.expConnectionTypeSelect = widget.NewSelect([]string{string(models.ConnectionTypeSSH), string(models.ConnectionTypeWordPress)}, nil)
	u.expConnectionTypeSelect.SetSelected(string(models.ConnectionTypeSSH))

	// --- SSH Widgets ---
	u.expHostEntry = widget.NewEntry()
	u.expHostEntry.SetPlaceHolder("192.168.1.100")
	u.expPortEntry = widget.NewEntry()
	u.expPortEntry.SetText("22")
	u.expSSHUserEntry = widget.NewEntry()
	u.expSSHUserEntry.SetPlaceHolder("root")

	u.expAuthTypeSelect = widget.NewSelect([]string{string(models.AuthTypePassword), string(models.AuthTypeKeyFile)}, nil)
	u.expAuthTypeSelect.SetSelected(string(models.AuthTypePassword))

	u.expSSHPassEntry = widget.NewPasswordEntry()
	u.expSSHPassEntry.SetPlaceHolder("SSH Password")

	u.expKeyPathEntry = widget.NewEntry()
	u.expKeyPathEntry.SetPlaceHolder("/path/to/private/key")
	keyPathBtn := widget.NewButton("Select Key", func() {
		fd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err == nil && reader != nil {
				u.expKeyPathEntry.SetText(reader.URI().Path())
			}
		}, w)
		fd.Show()
	})
	keyAuthContainer := container.NewBorder(nil, nil, nil, keyPathBtn, u.expKeyPathEntry)
	keyAuthContainer.Hide()

	u.expAuthTypeSelect.OnChanged = func(s string) {
		if s == string(models.AuthTypePassword) {
			u.expSSHPassEntry.Show()
			keyAuthContainer.Hide()
		} else {
			u.expSSHPassEntry.Hide()
			keyAuthContainer.Show()
		}
	}

	// --- WordPress Widgets ---
	u.expWPUrlEntry = widget.NewEntry()
	u.expWPUrlEntry.SetPlaceHolder("https://example.com")
	u.expWPKeyEntry = widget.NewEntry()
	u.expWPKeyEntry.SetPlaceHolder("API Key")

	generatePluginBtn := widget.NewButton("Generate Plugin", func() {
		// Ask for save location
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil || uri == nil {
				return
			}

			key, path, err := wordpress.GeneratePlugin("plugin_template/dback-sync.php", uri.Path())
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			u.expWPKeyEntry.SetText(key)
			dialog.ShowInformation("Plugin Generated", fmt.Sprintf("Plugin saved to %s\nAPI Key has been set.", path), w)
		}, w)
	})

	// Containers
	sshContainer := container.NewVBox(
		widget.NewForm(
			widget.NewFormItem("Host", u.expHostEntry),
			widget.NewFormItem("Port", u.expPortEntry),
			widget.NewFormItem("SSH User", u.expSSHUserEntry),
			widget.NewFormItem("Auth Type", u.expAuthTypeSelect),
		),
		u.expSSHPassEntry,
		keyAuthContainer,
	)

	wpContainer := container.NewVBox(
		widget.NewForm(
			widget.NewFormItem("WordPress URL", u.expWPUrlEntry),
			widget.NewFormItem("API Key", u.expWPKeyEntry),
		),
		generatePluginBtn,
	)
	wpContainer.Hide() // Default hidden

	// Toggle Logic
	u.expConnectionTypeSelect.OnChanged = func(s string) {
		if s == string(models.ConnectionTypeWordPress) {
			sshContainer.Hide()
			wpContainer.Show()
		} else {
			sshContainer.Show()
			wpContainer.Hide()
		}
	}

	// Test Server Connectivity
	testServerBtn := widget.NewButton("Test Connectivity", func() {
		connType := models.ConnectionType(u.expConnectionTypeSelect.Selected)

		if connType == models.ConnectionTypeWordPress {
			// Test WP
			// Use a simple check (e.g. try export but cancel? or create a ping endpoint?)
			// Since we don't have a ping endpoint in template, we can't easily check without triggering dump.
			// But user can check if URL is reachable.
			// Ideally we add a ping route. I'll skip for now and rely on user trying export.
			// Or I can update template to add ping.
			// Let's just assume success if URL is valid for MVP or try a GET.
			dialog.ShowInformation("Info", "To test, try starting backup. Errors will be reported.", w)
			return
		}

		// SSH Test
		loading := u.showLoading("Testing Connection", "Connecting to server...")
		go func() {
			p := models.Profile{
				Host:        u.expHostEntry.Text,
				Port:        u.expPortEntry.Text,
				SSHUser:     u.expSSHUserEntry.Text,
				SSHPassword: u.expSSHPassEntry.Text,
				AuthType:    models.AuthType(u.expAuthTypeSelect.Selected),
				AuthKeyPath: u.expKeyPathEntry.Text,
			}

			client, err := ssh.NewClient(p)
			if err != nil {
				loading.Hide()
				dialog.ShowError(fmt.Errorf("Connection Failed: %v", err), w)
				return
			}
			client.Close()
			loading.Hide()
			dialog.ShowInformation("Success", "SSH Connection Established Successfully!", w)
		}()
	})

	serverGroup := widget.NewCard("Connection", "", container.NewVBox(
		widget.NewForm(widget.NewFormItem("Type", u.expConnectionTypeSelect)),
		sshContainer,
		wpContainer,
		widget.NewSeparator(),
		testServerBtn,
	))

	// --- Source Database ---
	u.expIsDockerCheck = widget.NewCheck("Is Docker Container?", nil)
	u.expContainerIDEntry = widget.NewEntry()
	u.expContainerIDEntry.SetPlaceHolder("mysql_container_name")
	u.expContainerIDEntry.Disable()

	u.expIsDockerCheck.OnChanged = func(b bool) {
		if b {
			u.expContainerIDEntry.Enable()
		} else {
			u.expContainerIDEntry.Disable()
		}
	}

	u.expDBTypeSelect = widget.NewSelect([]string{string(models.DBTypeMySQL), string(models.DBTypeMariaDB)}, nil)
	u.expDBTypeSelect.SetSelected(string(models.DBTypeMySQL))

	u.expDBHostEntry = widget.NewEntry()
	u.expDBHostEntry.SetText("127.0.0.1")
	u.expDBPortEntry = widget.NewEntry()
	u.expDBPortEntry.SetText("3306")
	u.expDBUserEntry = widget.NewEntry()
	u.expDBUserEntry.SetPlaceHolder("root")
	u.expDBPassEntry = widget.NewPasswordEntry()
	u.expTargetDBEntry = widget.NewEntry()
	u.expTargetDBEntry.SetPlaceHolder("my_database")

	// Test DB Connectivity
	testDBBtn := widget.NewButton("Test DB Connectivity", func() {
		loading := u.showLoading("Testing DB", "Connecting to Database...")

		go func() {
			p := models.Profile{
				Host:        u.expHostEntry.Text,
				Port:        u.expPortEntry.Text,
				SSHUser:     u.expSSHUserEntry.Text,
				SSHPassword: u.expSSHPassEntry.Text,
				AuthType:    models.AuthType(u.expAuthTypeSelect.Selected),
				AuthKeyPath: u.expKeyPathEntry.Text,
				DBHost:      u.expDBHostEntry.Text,
				DBPort:      u.expDBPortEntry.Text,
				DBUser:      u.expDBUserEntry.Text,
				DBPassword:  u.expDBPassEntry.Text,
				IsDocker:    u.expIsDockerCheck.Checked,
				ContainerID: u.expContainerIDEntry.Text,
			}

			client, err := ssh.NewClient(p)
			if err != nil {
				loading.Hide()
				dialog.ShowError(fmt.Errorf("SSH Connection Failed: %v", err), w)
				return
			}
			defer client.Close()

			// Construct a ping command
			authArgs := fmt.Sprintf("-u %s -p'%s'", p.DBUser, p.DBPassword)
			var cmd string
			if p.IsDocker {
				cmd = fmt.Sprintf("docker exec -i %s mysqladmin %s ping", p.ContainerID, authArgs)
			} else {
				hostArgs := fmt.Sprintf("-h %s -P %s", p.DBHost, p.DBPort)
				cmd = fmt.Sprintf("mysqladmin %s %s ping", hostArgs, authArgs)
			}

			_, session, err := client.RunCommandStream(cmd)
			if err != nil {
				loading.Hide()
				dialog.ShowError(fmt.Errorf("DB Connection Failed (Cmd Error): %v", err), w)
				return
			}
			defer session.Close()

			if err := session.Wait(); err != nil {
				loading.Hide()
				dialog.ShowError(fmt.Errorf("DB Connection Failed (Ping Failed): %v", err), w)
				return
			}

			loading.Hide()
			dialog.ShowInformation("Success", "Database Connection Successful!", w)
		}()
	})

	dbForm := widget.NewForm(
		widget.NewFormItem("DB Type", u.expDBTypeSelect),
		widget.NewFormItem("Container Name/ID", u.expContainerIDEntry),
		widget.NewFormItem("DB Host", u.expDBHostEntry),
		widget.NewFormItem("DB Port", u.expDBPortEntry),
		widget.NewFormItem("DB User", u.expDBUserEntry),
		widget.NewFormItem("DB Password", u.expDBPassEntry),
		widget.NewFormItem("Target DB Name", u.expTargetDBEntry),
	)

	// Hide DB Form if WP is selected
	dbContainer := container.NewVBox(
		u.expIsDockerCheck,
		dbForm,
		widget.NewSeparator(),
		testDBBtn,
	)

	// Hook into connection type change to hide DB fields
	u.expConnectionTypeSelect.OnChanged = func(s string) {
		if s == string(models.ConnectionTypeWordPress) {
			sshContainer.Hide()
			wpContainer.Show()
			dbContainer.Hide() // WP plugin handles DB connection internally
		} else {
			sshContainer.Show()
			wpContainer.Hide()
			dbContainer.Show()
		}
	}

	dbGroup := widget.NewCard("Source Database", "", dbContainer)

	// --- Action ---
	u.expDestPathLabel = widget.NewLabel("No folder selected")

	selectFolderBtn := widget.NewButton("Select Destination Folder", func() {
		fd := dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
			if err == nil && uri != nil {
				u.expDestPathLabel.SetText(uri.Path())
			}
		}, w)
		fd.Show()
	})

	progressBar := widget.NewProgressBar()
	statusLabel := widget.NewLabel("Ready")

	startBtn := widget.NewButton("Start Backup & Download", func() {
		if u.expDestPathLabel.Text == "No folder selected" || u.expDestPathLabel.Text == "" {
			dialog.ShowError(fmt.Errorf("please select a destination folder"), w)
			return
		}
		destPath := u.expDestPathLabel.Text
		connType := models.ConnectionType(u.expConnectionTypeSelect.Selected)

		if connType == models.ConnectionTypeWordPress {
			// WP Flow
			wpUrl := u.expWPUrlEntry.Text
			wpKey := u.expWPKeyEntry.Text

			go func() {
				u.log("Export (WP)", "Starting export from WordPress", "", "In Progress", "")
				statusLabel.SetText("Requesting Export...")
				progressBar.SetValue(0) // Indeterminate?

				wpClient := wordpress.NewClient(wpUrl, wpKey)

				fileName := fmt.Sprintf("wp_dump_%s.sql.gz", time.Now().Format("20060102_150405"))
				fullPath := filepath.Join(destPath, fileName)

				err := wpClient.Export(fullPath, func(curr int64) {
					// We don't know total usually unless header provided.
					// Client handles progress callback.
					// Update UI
					mb := float64(curr) / 1024 / 1024
					statusLabel.SetText(fmt.Sprintf("Downloading: %.2f MB", mb))
				})

				if err != nil {
					statusLabel.SetText("Export Failed")
					u.log("Export (WP)", "WP Export Failed", "", "Failed", err.Error())
					return
				}

				statusLabel.SetText("Success! Saved to " + fileName)
				u.log("Export (WP)", "Export completed", "", "Success", "")
			}()
			return
		}

		// SSH Flow
		p := models.Profile{
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
			IsDocker:     u.expIsDockerCheck.Checked,
			ContainerID:  u.expContainerIDEntry.Text,
			TargetDBName: u.expTargetDBEntry.Text,
		}

		go func() {
			u.log("Export", fmt.Sprintf("Starting export for DB: %s on %s", p.TargetDBName, p.Host), "", "In Progress", "")
			statusLabel.SetText("Connecting...")
			progressBar.SetValue(0)

			client, err := ssh.NewClient(p)
			if err != nil {
				statusLabel.SetText("Connection Failed")
				u.log("Export", "Connection failed", "", "Failed", err.Error())
				return
			}
			defer client.Close()

			statusLabel.SetText("Connected. Generating command...")
			cmd := db.BuildExportCommand(p)

			statusLabel.SetText("Executing Dump & Streaming...")
			stdout, session, err := client.RunCommandStream(cmd)
			if err != nil {
				statusLabel.SetText("Command Failed")
				u.log("Export", "Command failed", "", "Failed", err.Error())
				return
			}
			defer session.Close()

			// Create local file
			fileName := fmt.Sprintf("%s_%s.sql.gz", p.TargetDBName, time.Now().Format("20060102_150405"))
			fullPath := filepath.Join(destPath, fileName)
			outFile, err := os.Create(fullPath)
			if err != nil {
				statusLabel.SetText("File Creation Failed")
				u.log("Export", "Local file creation failed", "", "Failed", err.Error())
				return
			}
			defer outFile.Close()

			progressR := &ssh.ProgressReader{
				Reader: stdout,
				Callback: func(current int64, total int64) {
					mb := float64(current) / 1024 / 1024
					statusLabel.SetText(fmt.Sprintf("Downloading: %.2f MB", mb))
					progressBar.SetValue(progressBar.Value + 0.01)
					if progressBar.Value >= 1.0 {
						progressBar.SetValue(0)
					}
				},
			}

			written, err := io.Copy(outFile, progressR)
			if err != nil {
				statusLabel.SetText("Download Failed")
				u.log("Export", "Stream download failed", "", "Failed", err.Error())
				return
			}

			statusLabel.SetText("Success! Saved to " + fileName)
			progressBar.SetValue(1.0)
			sizeStr := fmt.Sprintf("%.2f MB", float64(written)/1024/1024)
			u.log("Export", "Export completed successfully", sizeStr, "Success", "")
		}()
	})
	startBtn.Importance = widget.HighImportance

	actionGroup := widget.NewCard("Action", "", container.NewVBox(
		u.expDestPathLabel,
		selectFolderBtn,
		widget.NewSeparator(),
		startBtn,
		statusLabel,
		progressBar,
	))

	return container.NewVScroll(container.NewVBox(
		serverGroup,
		dbGroup,
		actionGroup,
	))
}
