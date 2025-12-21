package ui

import (
	"dback/backend/db"
	"dback/backend/ssh"
	"dback/backend/wordpress"
	"dback/models"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func (u *UI) createImportTab(w fyne.Window) fyne.CanvasObject {
	// --- Source File ---
	sourceFileLabel := widget.NewLabel("No file selected")
	var sourcePath string
	selectFileBtn := widget.NewButton("Select SQL Dump File (.sql, .sql.gz)", func() {
		fd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err == nil && reader != nil {
				sourcePath = reader.URI().Path()
				sourceFileLabel.SetText(sourcePath)
			}
		}, w)
		fd.Show()
	})

	sourceGroup := widget.NewCard("Source File", "", container.NewVBox(
		sourceFileLabel,
		selectFileBtn,
	))

	// --- Destination Server ---
	restoreLocalCheck := widget.NewCheck("Restore to Localhost?", nil)

	u.impConnectionTypeSelect = widget.NewSelect([]string{string(models.ConnectionTypeSSH), string(models.ConnectionTypeWordPress)}, nil)
	u.impConnectionTypeSelect.SetSelected(string(models.ConnectionTypeSSH))

	// SSH Fields
	u.impHostEntry = widget.NewEntry()
	u.impHostEntry.SetPlaceHolder("192.168.1.100")
	u.impPortEntry = widget.NewEntry()
	u.impPortEntry.SetText("22")
	u.impSSHUserEntry = widget.NewEntry()
	u.impSSHUserEntry.SetPlaceHolder("root")

	u.impAuthTypeSelect = widget.NewSelect([]string{string(models.AuthTypePassword), string(models.AuthTypeKeyFile)}, nil)
	u.impAuthTypeSelect.SetSelected(string(models.AuthTypePassword))

	u.impSSHPassEntry = widget.NewPasswordEntry()
	u.impSSHPassEntry.SetPlaceHolder("SSH Password")

	u.impKeyPathEntry = widget.NewEntry()
	u.impKeyPathEntry.SetPlaceHolder("/path/to/private/key")
	keyPathBtn := widget.NewButton("Select Key", func() {
		fd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err == nil && reader != nil {
				u.impKeyPathEntry.SetText(reader.URI().Path())
			}
		}, w)
		fd.Show()
	})
	keyAuthContainer := container.NewBorder(nil, nil, nil, keyPathBtn, u.impKeyPathEntry)
	keyAuthContainer.Hide()

	u.impAuthTypeSelect.OnChanged = func(s string) {
		if s == string(models.AuthTypePassword) {
			u.impSSHPassEntry.Show()
			keyAuthContainer.Hide()
		} else {
			u.impSSHPassEntry.Hide()
			keyAuthContainer.Show()
		}
	}

	sshForm := widget.NewForm(
		widget.NewFormItem("Host", u.impHostEntry),
		widget.NewFormItem("Port", u.impPortEntry),
		widget.NewFormItem("SSH User", u.impSSHUserEntry),
		widget.NewFormItem("Auth Type", u.impAuthTypeSelect),
	)
	sshContainer := container.NewVBox(sshForm, u.impSSHPassEntry, keyAuthContainer)

	// WP Fields
	u.impWPUrlEntry = widget.NewEntry()
	u.impWPUrlEntry.SetPlaceHolder("https://example.com")
	u.impWPKeyEntry = widget.NewEntry()
	u.impWPKeyEntry.SetPlaceHolder("API Key")

	wpContainer := container.NewVBox(
		widget.NewForm(
			widget.NewFormItem("WordPress URL", u.impWPUrlEntry),
			widget.NewFormItem("API Key", u.impWPKeyEntry),
		),
	)
	wpContainer.Hide()

	// Toggle Logic
	restoreLocalCheck.OnChanged = func(b bool) {
		if b {
			u.impConnectionTypeSelect.Hide()
			sshContainer.Hide()
			wpContainer.Hide()
		} else {
			u.impConnectionTypeSelect.Show()
			// Trigger conn type change
			u.impConnectionTypeSelect.OnChanged(u.impConnectionTypeSelect.Selected)
		}
	}

	u.impConnectionTypeSelect.OnChanged = func(s string) {
		if restoreLocalCheck.Checked {
			return
		}

		if s == string(models.ConnectionTypeWordPress) {
			sshContainer.Hide()
			wpContainer.Show()
		} else {
			sshContainer.Show()
			wpContainer.Hide()
		}
	}

	// Test Server Connectivity (Import)
	testServerBtn := widget.NewButton("Test Connectivity", func() {
		connType := models.ConnectionType(u.impConnectionTypeSelect.Selected)
		if restoreLocalCheck.Checked {
			dialog.ShowInformation("Info", "Localhost selected. No connection test needed.", w)
			return
		}

		if connType == models.ConnectionTypeWordPress {
			// Test WP via Ping endpoint
			wpUrl := u.impWPUrlEntry.Text
			wpKey := u.impWPKeyEntry.Text

			if wpUrl == "" {
				dialog.ShowError(fmt.Errorf("WordPress URL is required"), w)
				return
			}
			if wpKey == "" {
				dialog.ShowError(fmt.Errorf("API Key is required"), w)
				return
			}

			loading := u.showLoading("Testing Connection", "Connecting to WordPress...")
			go func() {
				wpClient := wordpress.NewClient(wpUrl, wpKey)
				pingResp, err := wpClient.Ping()
				loading.Hide()

				if err != nil {
					errMsg := fmt.Sprintf("WordPress Connection Failed:\n\n%s\n\nPossible causes:\n• Plugin not installed/activated\n• Wrong URL or API key\n• Site not reachable", err.Error())
					dialog.ShowError(fmt.Errorf(errMsg), w)
					return
				}

				// Format server info
				info := fmt.Sprintf("✓ Connection Successful!\n\n"+
					"Plugin Version: %s\n"+
					"Shell Available: %v\n"+
					"DB Connected: %v",
					pingResp.Version,
					pingResp.CanUseShell,
					pingResp.DBConnected)

				// Warn about import-specific issues
				warnings := ""
				if !pingResp.CanUseShell {
					warnings += "\n\n⚠️ Shell disabled: mysql command not available, import will fail"
				}
				if !pingResp.DBConnected {
					warnings += "\n\n⚠️ Database not connected: import will fail"
				}

				dialog.ShowInformation("WordPress Connection", info+warnings, w)
			}()
			return
		}

		// SSH Test
		loading := u.showLoading("Testing Connection", "Connecting to server...")
		go func() {
			p := models.Profile{
				Host:        strings.TrimSpace(u.impHostEntry.Text),
				Port:        strings.TrimSpace(u.impPortEntry.Text),
				SSHUser:     strings.TrimSpace(u.impSSHUserEntry.Text),
				SSHPassword: strings.TrimSpace(u.impSSHPassEntry.Text),
				AuthType:    models.AuthType(u.impAuthTypeSelect.Selected),
				AuthKeyPath: strings.TrimSpace(u.impKeyPathEntry.Text),
			}

			client, err := ssh.NewClient(p)
			if err != nil {
				loading.Hide()
				u.showErrorAndLog("SSH Connection Failed", err, "Test SSH (Import)")
				return
			}
			client.Close()
			loading.Hide()
			dialog.ShowInformation("Success", "SSH Connection Established Successfully!", w)
		}()
	})

	serverGroup := widget.NewCard("Destination Server", "", container.NewVBox(
		restoreLocalCheck,
		widget.NewForm(widget.NewFormItem("Type", u.impConnectionTypeSelect)),
		sshContainer,
		wpContainer,
		widget.NewSeparator(),
		testServerBtn,
	))

	// --- Destination Database ---
	u.impIsDockerCheck = widget.NewCheck("Is Docker Container?", nil)
	u.impContainerIDEntry = widget.NewEntry()
	u.impContainerIDEntry.SetPlaceHolder("mysql_container_name")
	u.impContainerIDEntry.Disable()

	u.impIsDockerCheck.OnChanged = func(b bool) {
		if b {
			u.impContainerIDEntry.Enable()
		} else {
			u.impContainerIDEntry.Disable()
		}
	}

	u.impDBTypeSelect = widget.NewSelect([]string{string(models.DBTypeMySQL), string(models.DBTypeMariaDB), string(models.DBTypePostgreSQL), string(models.DBTypeCouchDB)}, nil)
	u.impDBTypeSelect.SetSelected(string(models.DBTypeMySQL))

	u.impDBHostEntry = widget.NewEntry()
	u.impDBHostEntry.SetText("127.0.0.1")
	u.impDBPortEntry = widget.NewEntry()
	u.impDBPortEntry.SetText("3306")
	u.impDBUserEntry = widget.NewEntry()
	u.impDBUserEntry.SetPlaceHolder("root")
	u.impDBPassEntry = widget.NewPasswordEntry()
	u.impTargetDBEntry = widget.NewEntry()
	u.impTargetDBEntry.SetPlaceHolder("target_database")

	// Test DB Connectivity (Import)
	testDBBtn := widget.NewButton("Test DB Connectivity", func() {
			p := models.Profile{
				DBHost:      strings.TrimSpace(u.impDBHostEntry.Text),
				DBPort:      strings.TrimSpace(u.impDBPortEntry.Text),
				DBUser:      strings.TrimSpace(u.impDBUserEntry.Text),
				DBPassword:  strings.TrimSpace(u.impDBPassEntry.Text),
				DBType:      models.DBType(u.impDBTypeSelect.Selected),
				IsDocker:    u.impIsDockerCheck.Checked,
				ContainerID: strings.TrimSpace(u.impContainerIDEntry.Text),
			}

		if restoreLocalCheck.Checked {
			// Local DB Test
			loading := u.showLoading("Testing Local DB", "Connecting to database...")
			go func() {
				var cmdStr string
				authArgs := fmt.Sprintf("-u %s -p'%s'", p.DBUser, p.DBPassword)

				if p.DBType == models.DBTypePostgreSQL {
					authEnv := fmt.Sprintf("PGPASSWORD='%s'", p.DBPassword)
					if p.IsDocker {
						cmdStr = fmt.Sprintf("docker exec %s sh -c '%s pg_isready -U %s'", p.ContainerID, authEnv, p.DBUser)
					} else {
						hostArgs := fmt.Sprintf("-h %s -p %s", p.DBHost, p.DBPort)
						cmdStr = fmt.Sprintf("%s pg_isready %s -U %s", authEnv, hostArgs, p.DBUser)
					}
				} else if p.DBType == models.DBTypeCouchDB {
					targetHost := p.DBHost
					if p.IsDocker {
						targetHost = "127.0.0.1"
					}
					url := fmt.Sprintf("http://%s:%s/", targetHost, p.DBPort)
					cmdStr = fmt.Sprintf("curl -s -f -u %s:%s %s", p.DBUser, p.DBPassword, url)
				} else {
					// MySQL/MariaDB - try both commands
					if p.IsDocker {
						cmdStr = fmt.Sprintf("docker exec %s sh -c 'if command -v mariadb-admin >/dev/null 2>&1; then mariadb-admin %s ping; else mysqladmin %s ping; fi'",
							p.ContainerID, authArgs, authArgs)
					} else {
						hostArgs := fmt.Sprintf("-h %s -P %s", p.DBHost, p.DBPort)
						cmdStr = fmt.Sprintf("sh -c 'if command -v mariadb-admin >/dev/null 2>&1; then mariadb-admin %s %s ping; else mysqladmin %s %s ping; fi'",
							hostArgs, authArgs, hostArgs, authArgs)
					}
				}

				cmd := exec.Command("bash", "-c", cmdStr)
				output, err := cmd.CombinedOutput()
				loading.Hide()

				if err != nil {
					errMsg := fmt.Sprintf("Connection Failed\n\nError: %v\nOutput: %s\n\nCommand: %s", err, string(output), cmdStr)
					dialog.ShowError(fmt.Errorf(errMsg), w)
					return
				}

				dialog.ShowInformation("Success", "Database Connection Successful!\n"+string(output), w)
			}()
			return
		}

		// Remote DB Test via SSH
		loading := u.showLoading("Testing DB", "Connecting to Database...")
		go func() {
			p.Host = strings.TrimSpace(u.impHostEntry.Text)
			p.Port = strings.TrimSpace(u.impPortEntry.Text)
			p.SSHUser = strings.TrimSpace(u.impSSHUserEntry.Text)
			p.SSHPassword = strings.TrimSpace(u.impSSHPassEntry.Text)
			p.AuthType = models.AuthType(u.impAuthTypeSelect.Selected)
			p.AuthKeyPath = strings.TrimSpace(u.impKeyPathEntry.Text)

			client, err := ssh.NewClient(p)
			if err != nil {
				loading.Hide()
				u.showErrorAndLog("SSH Connection Failed", err, "Test DB (Import)")
				return
			}
			defer client.Close()

			var cmd string
			if p.DBType == models.DBTypePostgreSQL {
				authEnv := fmt.Sprintf("PGPASSWORD='%s'", p.DBPassword)
				if p.IsDocker {
					cmd = fmt.Sprintf("docker exec -e %s %s pg_isready -U %s", authEnv, p.ContainerID, p.DBUser)
				} else {
					hostArgs := fmt.Sprintf("-h %s -p %s", p.DBHost, p.DBPort)
					cmd = fmt.Sprintf("%s pg_isready %s -U %s", authEnv, hostArgs, p.DBUser)
				}
			} else if p.DBType == models.DBTypeCouchDB {
				targetHost := p.DBHost
				if p.IsDocker {
					targetHost = "127.0.0.1"
				}
				url := fmt.Sprintf("http://%s:%s/", targetHost, p.DBPort)
				auth := fmt.Sprintf("-u %s:%s", p.DBUser, p.DBPassword)
				if p.IsDocker {
					cmd = fmt.Sprintf("docker exec %s curl -s -f %s %s", p.ContainerID, auth, url)
				} else {
					cmd = fmt.Sprintf("curl -s -f %s %s", auth, url)
				}
			} else {
				// MySQL/MariaDB - try both commands
				authArgs := fmt.Sprintf("-u %s -p'%s'", p.DBUser, p.DBPassword)
				if p.IsDocker {
					cmd = fmt.Sprintf("docker exec %s sh -c 'if command -v mariadb-admin >/dev/null 2>&1; then mariadb-admin %s ping; else mysqladmin %s ping; fi'",
						p.ContainerID, authArgs, authArgs)
				} else {
					hostArgs := fmt.Sprintf("-h %s -P %s", p.DBHost, p.DBPort)
					cmd = fmt.Sprintf("sh -c 'if command -v mariadb-admin >/dev/null 2>&1; then mariadb-admin %s %s ping; else mysqladmin %s %s ping; fi'",
						hostArgs, authArgs, hostArgs, authArgs)
				}
			}

			output, err := client.RunCommand(cmd)
			if err != nil {
				loading.Hide()
				u.showErrorAndLog("DB Check Cmd Failed", err, "Test DB (Import)")
				return
			}

			loading.Hide()
			dialog.ShowInformation("Success", "Database Connection Successful!\n"+output, w)
		}()
	})

	dbGroup := widget.NewCard("Destination Database", "", container.NewVBox(
		u.impIsDockerCheck,
		widget.NewForm(
			widget.NewFormItem("DB Type", u.impDBTypeSelect),
			widget.NewFormItem("Container Name/ID", u.impContainerIDEntry),
			widget.NewFormItem("DB Host", u.impDBHostEntry),
			widget.NewFormItem("DB Port", u.impDBPortEntry),
			widget.NewFormItem("DB User", u.impDBUserEntry),
			widget.NewFormItem("DB Password", u.impDBPassEntry),
			widget.NewFormItem("Target DB Name", u.impTargetDBEntry),
		),
		widget.NewSeparator(),
		testDBBtn,
	))

	// --- Action ---
	progressBar := widget.NewProgressBar()
	statusLabel := widget.NewLabel("Ready")

	startBtn := widget.NewButton("Start Upload & Restore", func() {
		if sourcePath == "" {
			dialog.ShowError(fmt.Errorf("please select a source file"), w)
			return
		}

		connType := models.ConnectionType(u.impConnectionTypeSelect.Selected)
		isLocal := restoreLocalCheck.Checked

		if !isLocal && connType == models.ConnectionTypeWordPress {
			// WP Import
			wpUrl := u.impWPUrlEntry.Text
			wpKey := u.impWPKeyEntry.Text

			// Validate inputs
			if wpUrl == "" {
				dialog.ShowError(fmt.Errorf("WordPress URL is required"), w)
				return
			}
			if wpKey == "" {
				dialog.ShowError(fmt.Errorf("API Key is required"), w)
				return
			}

			go func() {
				u.log(nil, "Import (WP)", "Starting import to WordPress: "+wpUrl, "", "", "In Progress", "")
				statusLabel.SetText("Uploading & Restoring...")
				progressBar.SetValue(0)

				wpClient := wordpress.NewClient(wpUrl, wpKey)

				// Get file size for progress
				fileInfo, err := os.Stat(sourcePath)
				if err != nil {
					statusLabel.SetText("Failed to read file")
					u.log(nil, "Import (WP)", "Failed to stat source file", "", "", "Failed", err.Error())
					dialog.ShowError(fmt.Errorf("Failed to read source file: %v", err), w)
					return
				}
				totalSize := fileInfo.Size()

				err = wpClient.Import(sourcePath, func(curr int64) {
					pct := float64(curr) / float64(totalSize)
						progressBar.SetValue(pct)
						statusLabel.SetText(fmt.Sprintf("Uploading: %.1f%%", pct*100))
				})

				if err != nil {
					statusLabel.SetText("Import Failed")
					u.log(nil, "Import (WP)", "WP Import Failed", "", "", "Failed", err.Error())
					dialog.ShowError(fmt.Errorf("WordPress Import Failed:\n%s", err.Error()), w)
					return
				}

				progressBar.SetValue(1.0)
				sizeStr := fmt.Sprintf("%.2f MB", float64(totalSize)/1024/1024)
				statusLabel.SetText(fmt.Sprintf("Success! Restore Completed (%s)", sizeStr))
				u.log(nil, "Import (WP)", "Import completed", sourcePath, sizeStr, "Success", "")
				dialog.ShowInformation("Import Complete", fmt.Sprintf("Database restored successfully!\n\nFile: %s\nSize: %s", sourcePath, sizeStr), w)
			}()
			return
		}

		p := models.Profile{
			Host:         strings.TrimSpace(u.impHostEntry.Text),
			Port:         strings.TrimSpace(u.impPortEntry.Text),
			SSHUser:      strings.TrimSpace(u.impSSHUserEntry.Text),
			SSHPassword:  strings.TrimSpace(u.impSSHPassEntry.Text),
			AuthType:     models.AuthType(u.impAuthTypeSelect.Selected),
			AuthKeyPath:  strings.TrimSpace(u.impKeyPathEntry.Text),
			DBHost:       strings.TrimSpace(u.impDBHostEntry.Text),
			DBPort:       strings.TrimSpace(u.impDBPortEntry.Text),
			DBUser:       strings.TrimSpace(u.impDBUserEntry.Text),
			DBPassword:   strings.TrimSpace(u.impDBPassEntry.Text),
			DBType:       models.DBType(u.impDBTypeSelect.Selected),
			IsDocker:     u.impIsDockerCheck.Checked,
			ContainerID:  strings.TrimSpace(u.impContainerIDEntry.Text),
			TargetDBName: strings.TrimSpace(u.impTargetDBEntry.Text),
		}

		go func() {
			u.log(&p, "Import", fmt.Sprintf("Starting import for DB: %s", p.TargetDBName), "", "", "In Progress", "")
			statusLabel.SetText("Preparing...")
			progressBar.SetValue(0)

			// Build the import command
			cmdStr := db.BuildImportCommand(p)

			inFile, err := os.Open(sourcePath)
			if err != nil {
				statusLabel.SetText("Open File Failed")
				u.log(&p, "Import", "Failed to open source file", "", "", "Failed", err.Error())
				return
			}
			defer inFile.Close()

			fileInfo, _ := inFile.Stat()
			totalSize := fileInfo.Size()

			if isLocal {
				// Execute Locally
				statusLabel.SetText("Executing Local Restore...")
				cmd := exec.Command("bash", "-c", cmdStr)

				stdin, err := cmd.StdinPipe()
				if err != nil {
					statusLabel.SetText("Local Pipe Failed")
					dialog.ShowError(fmt.Errorf("Failed to create stdin pipe: %v", err), w)
					return
				}

				// Capture stderr for error messages
				stderrPipe, _ := cmd.StderrPipe()
				var stderrBuf strings.Builder
				if stderrPipe != nil {
					go func() {
						io.Copy(&stderrBuf, stderrPipe)
					}()
				}

				// Start command
				if err := cmd.Start(); err != nil {
					statusLabel.SetText("Local Command Start Failed")
					dialog.ShowError(fmt.Errorf("Failed to start command: %v\n\nCommand: %s", err, cmdStr), w)
					return
				}

				// Copy with progress (upload phase: 0-50%)
				progressR := &ssh.ProgressReader{
					Reader: inFile,
					Total:  totalSize,
					Callback: func(current int64, total int64) {
						pct := float64(current) / float64(total) * 0.5 // 0-50%
						progressBar.SetValue(pct)
						statusLabel.SetText(fmt.Sprintf("Uploading: %.1f%%", pct*100*2))
					},
				}

				io.Copy(stdin, progressR)
				stdin.Close()

				// Processing phase: show indeterminate progress
				statusLabel.SetText("⏳ Processing... Please wait (DROP + CREATE + IMPORT)")
				progressBar.SetValue(0.5)
				
				// Animate progress bar while waiting
				done := make(chan bool)
				go func() {
					val := 0.5
					for {
						select {
						case <-done:
							return
						default:
							val += 0.05
							if val > 0.95 {
								val = 0.5
							}
							progressBar.SetValue(val)
							time.Sleep(200 * time.Millisecond)
						}
					}
				}()

				err = cmd.Wait()
				done <- true
				
				if err != nil {
					stderrStr := stderrBuf.String()
					errMsg := fmt.Sprintf("Error: %v", err)
					if stderrStr != "" {
						errMsg += fmt.Sprintf("\n\nDetails:\n%s", stderrStr)
					}
					errMsg += fmt.Sprintf("\n\nCommand:\n%s", cmdStr)

					statusLabel.SetText("Restore Failed")
					u.log(&p, "Import", "Local restore failed", "", "", "Failed", errMsg)
					dialog.ShowError(fmt.Errorf("Local Restore Failed:\n\n%s", errMsg), w)
					return
				}

			} else {
				// Execute Remote via SSH
				statusLabel.SetText("Connecting to SSH...")
				client, err := ssh.NewClient(p)
				if err != nil {
					statusLabel.SetText("SSH Connection Failed")
					u.log(&p, "Import", "SSH Connection failed", "", "", "Failed", err.Error())
					return
				}
				defer client.Close()

				statusLabel.SetText("Connected. Starting Stream...")
				stdin, stderr, session, err := client.RunCommandPipeInput(cmdStr)
				if err != nil {
					statusLabel.SetText("Remote Command Failed")
					u.log(&p, "Import", "Remote command failed", "", "", "Failed", err.Error())
					return
				}
				defer session.Close()

				// Capture stderr
				var stderrBuf strings.Builder
				go func() {
					io.Copy(&stderrBuf, stderr)
				}()

				// Upload phase: 0-50%
				progressR := &ssh.ProgressReader{
					Reader: inFile,
					Total:  totalSize,
					Callback: func(current int64, total int64) {
						pct := float64(current) / float64(total) * 0.5
						progressBar.SetValue(pct)
						statusLabel.SetText(fmt.Sprintf("Uploading: %.1f%%", pct*100*2))
					},
				}

				_, err = io.Copy(stdin, progressR)
				if err != nil {
					statusLabel.SetText("Upload Stream Failed")
					u.log(&p, "Import", "Upload stream failed", "", "", "Failed", err.Error())
					return
				}

				// Close stdin to signal EOF to remote process
				stdin.Close()

				// Processing phase: animate progress bar
				statusLabel.SetText("⏳ Processing... Please wait (DROP + CREATE + IMPORT)")
				progressBar.SetValue(0.5)
				
				done := make(chan bool)
				go func() {
					val := 0.5
					for {
						select {
						case <-done:
							return
						default:
							val += 0.05
							if val > 0.95 {
								val = 0.5
							}
							progressBar.SetValue(val)
							time.Sleep(200 * time.Millisecond)
						}
					}
				}()

				// Wait for remote command to finish
				err = session.Wait()
				done <- true
				
				if err != nil {
					errMsg := fmt.Sprintf("Process exited with error: %v. Stderr: %s", err, stderrBuf.String())
					statusLabel.SetText("Restore Process Failed")
					u.log(&p, "Import", "Remote restore process failed", "", "", "Failed", errMsg)
					dialog.ShowError(fmt.Errorf("Restore Failed:\n%s", errMsg), w)
					return
				}
			}

			statusLabel.SetText("Restore Completed Successfully!")
			progressBar.SetValue(1.0)
			sizeStr := fmt.Sprintf("%.2f MB", float64(totalSize)/1024/1024)
			u.log(&p, "Import", "Import completed successfully", sourcePath, sizeStr, "Success", "")
			
			// Show success dialog
			dialog.ShowInformation("Import Complete", 
				fmt.Sprintf("Database restored successfully!\n\nDatabase: %s\nFile: %s\nSize: %s", 
					p.TargetDBName, filepath.Base(sourcePath), sizeStr), w)
		}()
	})
	startBtn.Importance = widget.HighImportance

	actionGroup := widget.NewCard("Action", "", container.NewVBox(
		sourceGroup,
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
