package ui

import (
	"dback/models"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func (u *UI) createHistoryTab() fyne.CanvasObject {
	// Table columns: Time, Action, Status, Size, Details, Actions
	headers := []string{"Time", "Action", "Status", "Size", "Details"}

	table := widget.NewTable(
		func() (int, int) {
			return len(u.filteredLogs), len(headers)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("Cell Content")
		},
		func(i widget.TableCellID, o fyne.CanvasObject) {
			// i.Row is index in filteredLogs.
			// Show newest first logic handled in refreshHistory?
			// Or just reverse index here?
			// filteredLogs will be sorted or we just index it.
			if i.Row >= len(u.filteredLogs) {
				return
			}
			entry := u.filteredLogs[len(u.filteredLogs)-1-i.Row]

			label := o.(*widget.Label)

			switch i.Col {
			case 0:
				label.SetText(entry.Timestamp.Format("2006-01-02 15:04:05"))
			case 1:
				label.SetText(entry.Action)
			case 2:
				label.SetText(entry.Status)
			case 3:
				label.SetText(entry.FileSize)
			case 4:
				label.SetText(entry.Details)
			}
		},
	)

	// Set column widths
	table.SetColumnWidth(0, 150)
	table.SetColumnWidth(1, 80)
	table.SetColumnWidth(2, 80)
	table.SetColumnWidth(3, 80)
	table.SetColumnWidth(4, 300)

	// Header
	headerContainer := container.NewGridWithColumns(len(headers))
	for _, h := range headers {
		headerContainer.Add(widget.NewLabelWithStyle(h, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
	}

	u.historyTable = table

	var selectedRow int = -1
	table.OnSelected = func(id widget.TableCellID) {
		selectedRow = id.Row
	}

	// Toolbar for Actions
	importBtn := widget.NewButtonWithIcon("Import", theme.DownloadIcon(), func() {
		if selectedRow < 0 || selectedRow >= len(u.filteredLogs) {
			return
		}
		// Get entry
		// Reverse index match?
		// My table logic: entry := u.filteredLogs[len(u.filteredLogs)-1-i.Row]
		// So selectedRow maps to that.
		realIdx := len(u.filteredLogs) - 1 - selectedRow
		if realIdx < 0 || realIdx >= len(u.filteredLogs) {
			return
		}
		entry := u.filteredLogs[realIdx]

		// Import logic
		// Use Import Tab logic? Or call helper?
		// I need to switch to Import tab and populate?
		// Or just run import?
		// The user said "Import in selected profile".
		// So we use u.currentProfileID (selected in sidebar).
		// And entry.FilePath.
		// Assuming u.profiles has the profile.
		var p models.Profile
		found := false
		for _, prof := range u.profiles {
			if prof.ID == u.currentProfileID {
				p = prof
				found = true
				break
			}
		}
		if !found {
			return
		}

		// Trigger import placeholder
		msg := fmt.Sprintf("Request to import file:\n%s\n\nInto Profile:\n%s", entry.FilePath, p.Name)
		dialog.ShowInformation("Import from History", msg, u.window)
		u.log(&p, "History Import", "Import requested from history", entry.FilePath, entry.FileSize, "Pending", "")
	})

	deleteBtn := widget.NewButtonWithIcon("Remove", theme.DeleteIcon(), func() {
		if selectedRow < 0 {
			return
		}
		realIdx := len(u.filteredLogs) - 1 - selectedRow
		if realIdx < 0 || realIdx >= len(u.filteredLogs) {
			return
		}
		entry := u.filteredLogs[realIdx]

		// Remove from u.logs
		// Find index in u.logs
		for i, log := range u.logs {
			if log.Timestamp == entry.Timestamp && log.Details == entry.Details { // Match unique?
				// Remove
				u.logs = append(u.logs[:i], u.logs[i+1:]...)
				break
			}
		}
		u.saveLogs()
		u.refreshHistory()
	})

	toolbar := container.NewHBox(importBtn, deleteBtn)

	return container.NewBorder(container.NewVBox(headerContainer, toolbar), nil, nil, nil, table)
}

func (u *UI) refreshHistory() {
	// Filter u.logs by u.currentProfileID
	u.filteredLogs = []models.LogEntry{}
	for _, entry := range u.logs {
		if entry.ProfileID == u.currentProfileID {
			u.filteredLogs = append(u.filteredLogs, entry)
		}
	}
	if u.historyTable != nil {
		u.historyTable.Refresh()
	}
}
