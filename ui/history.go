package ui

import (
	"dback/backend/db"
	"dback/backend/ssh"
	"dback/models"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func (u *UI) createHistoryTab() fyne.CanvasObject {
	// Use a List instead of Table for better layout
	list := widget.NewList(
		func() int {
			return len(u.exportRecords)
		},
		func() fyne.CanvasObject {
			// Each row: Date | Profile | Database | Size | Filename
			dateLabel := widget.NewLabel("2006-01-02 15:04")
			dateLabel.TextStyle = fyne.TextStyle{Bold: true}
			profileLabel := widget.NewLabel("Profile Name")
			dbLabel := widget.NewLabel("Database")
			sizeLabel := widget.NewLabel("0.00 MB")
			fileLabel := widget.NewLabel("filename.sql.gz")
			fileLabel.Truncation = fyne.TextTruncateEllipsis

			return container.NewHBox(
				container.NewGridWithColumns(5,
					dateLabel,
					profileLabel,
					dbLabel,
					sizeLabel,
					fileLabel,
				),
			)
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			if i >= len(u.exportRecords) {
				return
			}
			// Show newest first
			record := u.exportRecords[len(u.exportRecords)-1-i]

			hbox := o.(*fyne.Container)
			grid := hbox.Objects[0].(*fyne.Container)

			dateLabel := grid.Objects[0].(*widget.Label)
			profileLabel := grid.Objects[1].(*widget.Label)
			dbLabel := grid.Objects[2].(*widget.Label)
			sizeLabel := grid.Objects[3].(*widget.Label)
			fileLabel := grid.Objects[4].(*widget.Label)

			dateLabel.SetText(record.ExportDate.Format("2006-01-02 15:04"))
			profileLabel.SetText(record.ProfileName)
			dbLabel.SetText(record.DatabaseName)
			sizeLabel.SetText(record.FileSize)
			fileLabel.SetText(filepath.Base(record.FilePath))
		},
	)

	u.historyList = list

	var selectedIdx int = -1
	list.OnSelected = func(id widget.ListItemID) {
		selectedIdx = id
	}

	// Helper to get selected record
	getSelectedRecord := func() (*models.ExportRecord, int) {
		if selectedIdx < 0 || selectedIdx >= len(u.exportRecords) {
			return nil, -1
		}
		realIdx := len(u.exportRecords) - 1 - selectedIdx
		if realIdx < 0 || realIdx >= len(u.exportRecords) {
			return nil, -1
		}
		return &u.exportRecords[realIdx], realIdx
	}

	// Action Buttons
	importBtn := widget.NewButtonWithIcon("Import to Profile", theme.DownloadIcon(), func() {
		record, _ := getSelectedRecord()
		if record == nil {
			dialog.ShowInformation("Info", "Please select a backup first", u.window)
			return
		}

		// Check if file exists
		if _, err := os.Stat(record.FilePath); os.IsNotExist(err) {
			dialog.ShowError(fmt.Errorf("File not found: %s", record.FilePath), u.window)
			return
		}

		// Find the profile (on-demand, from current profiles list)
		var profile *models.Profile
		
		// First try to match by ID
		for i := range u.profiles {
			if u.profiles[i].ID == record.ProfileID {
				profile = &u.profiles[i]
				break
			}
		}
		
		// If not found by ID, try to match by name
		if profile == nil {
			for i := range u.profiles {
				if u.profiles[i].Name == record.ProfileName {
					profile = &u.profiles[i]
					break
				}
			}
		}

		if profile == nil {
			// Check if it's a temp profile (starts with "temp_")
			if strings.HasPrefix(record.ProfileID, "temp_") {
				dialog.ShowError(fmt.Errorf("This backup was created without a saved profile.\n\nPlease use the Import tab to import this file:\n%s", record.FilePath), u.window)
			} else {
				dialog.ShowError(fmt.Errorf("Profile '%s' not found. It may have been deleted.", record.ProfileName), u.window)
			}
			return
		}

		// Show confirmation dialog
		dialog.ShowConfirm("Import Database",
			fmt.Sprintf("Import this backup?\n\nProfile: %s\nFile: %s\nDatabase: %s",
				profile.Name, filepath.Base(record.FilePath), record.DatabaseName),
			func(confirmed bool) {
				if confirmed {
					// Start import in background
					go u.runImportFromHistory(record, profile)
				}
			}, u.window)
	})

	openFolderBtn := widget.NewButtonWithIcon("Open Folder", theme.FolderOpenIcon(), func() {
		record, _ := getSelectedRecord()
		if record == nil {
			dialog.ShowInformation("Info", "Please select a backup first", u.window)
			return
		}

		folderPath := filepath.Dir(record.FilePath)

		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command("open", folderPath)
		case "windows":
			cmd = exec.Command("explorer", folderPath)
		default:
			cmd = exec.Command("xdg-open", folderPath)
		}
		cmd.Start()
	})

	deleteFileBtn := widget.NewButtonWithIcon("Delete File", theme.DeleteIcon(), func() {
		record, realIdx := getSelectedRecord()
		if record == nil {
			dialog.ShowInformation("Info", "Please select a backup first", u.window)
			return
		}

		dialog.ShowConfirm("Delete File",
			fmt.Sprintf("Delete this file permanently?\n\n%s", filepath.Base(record.FilePath)),
			func(confirmed bool) {
				if confirmed {
					if err := os.Remove(record.FilePath); err != nil && !os.IsNotExist(err) {
						dialog.ShowError(fmt.Errorf("Failed to delete: %v", err), u.window)
						return
					}
					u.exportRecords = append(u.exportRecords[:realIdx], u.exportRecords[realIdx+1:]...)
					u.saveExportHistory()
					u.refreshHistory()
					selectedIdx = -1
					dialog.ShowInformation("Deleted", "File deleted", u.window)
				}
			}, u.window)
	})

	removeRecordBtn := widget.NewButtonWithIcon("Remove from List", theme.ContentClearIcon(), func() {
		record, realIdx := getSelectedRecord()
		if record == nil {
			dialog.ShowInformation("Info", "Please select a backup first", u.window)
			return
		}

		dialog.ShowConfirm("Remove",
			"Remove from list? (File stays on disk)",
			func(confirmed bool) {
				if confirmed {
					u.exportRecords = append(u.exportRecords[:realIdx], u.exportRecords[realIdx+1:]...)
					u.saveExportHistory()
		u.refreshHistory()
					selectedIdx = -1
				}
			}, u.window)
	})

	toolbar := container.NewHBox(importBtn, openFolderBtn, deleteFileBtn, removeRecordBtn)

	// Header row
	headerGrid := container.NewGridWithColumns(5,
		widget.NewLabelWithStyle("Date/Time", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("Profile", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("Database", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("Size", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("File", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
	)

	return container.NewBorder(
		container.NewVBox(toolbar, headerGrid),
		nil, nil, nil,
		list,
	)
}

// runImportFromHistory executes the actual import using profile settings
func (u *UI) runImportFromHistory(record *models.ExportRecord, p *models.Profile) {
	// Show progress dialog
	progressDialog := dialog.NewCustomWithoutButtons("Importing...",
		widget.NewLabel(fmt.Sprintf("Importing to %s...", p.Name)), u.window)
	progressDialog.Show()

	defer progressDialog.Hide()

	// Open the file
	inFile, err := os.Open(record.FilePath)
	if err != nil {
		dialog.ShowError(fmt.Errorf("Failed to open file: %v", err), u.window)
		return
	}
	defer inFile.Close()

	fileInfo, _ := inFile.Stat()
	totalSize := fileInfo.Size()

	// Build import command based on profile type
	if p.ConnectionType == models.ConnectionTypeWordPress {
		// WordPress import not supported from history yet
		dialog.ShowError(fmt.Errorf("WordPress import from history not supported yet.\nPlease use the Import tab."), u.window)
		return
	}

	// SSH Import
	cmdStr := db.BuildImportCommand(*p)

	client, err := ssh.NewClient(*p)
	if err != nil {
		dialog.ShowError(fmt.Errorf("SSH Connection failed: %v", err), u.window)
		return
	}
	defer client.Close()

	stdin, stderr, session, err := client.RunCommandPipeInput(cmdStr)
	if err != nil {
		dialog.ShowError(fmt.Errorf("Command failed: %v", err), u.window)
		return
	}
	defer session.Close()

	// Capture stderr
	var stderrBuf strings.Builder
	go func() {
		io.Copy(&stderrBuf, stderr)
	}()

	// Copy file to stdin
	_, err = io.Copy(stdin, inFile)
	if err != nil {
		dialog.ShowError(fmt.Errorf("Upload failed: %v", err), u.window)
		return
	}
	stdin.Close()

	// Wait for command to finish
	if err := session.Wait(); err != nil {
		errMsg := fmt.Sprintf("Import failed: %v\n%s", err, stderrBuf.String())
		dialog.ShowError(fmt.Errorf(errMsg), u.window)
		u.log(p, "Import (History)", "Import failed", record.FilePath, record.FileSize, "Failed", errMsg)
		return
	}

	// Success
	sizeStr := fmt.Sprintf("%.2f MB", float64(totalSize)/1024/1024)
	u.log(p, "Import (History)", "Import completed from history", record.FilePath, sizeStr, "Success", "")
	dialog.ShowInformation("Success", fmt.Sprintf("Import completed!\n\nProfile: %s\nDatabase: %s", p.Name, p.TargetDBName), u.window)
}

func (u *UI) refreshHistory() {
	if u.historyList != nil {
		u.historyList.Refresh()
	}
}
