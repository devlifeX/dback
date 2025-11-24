package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func (u *UI) createHistoryTab() fyne.CanvasObject {
	// Table columns
	headers := []string{"Time", "Action", "Status", "Size", "Details"}

	// We need a way to refresh the table when logs change.
	// widget.Table doesn't have a simple binding like List yet in standard use without data binding package.
	// We'll use the standard callback approach.

	table := widget.NewTable(
		func() (int, int) {
			return len(u.logs), len(headers)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("Cell Content")
		},
		func(i widget.TableCellID, o fyne.CanvasObject) {
			// i.Row is index in logs. i.Col is column index.
			// Show newest first?
			idx := len(u.logs) - 1 - i.Row
			if idx < 0 {
				return
			}
			entry := u.logs[idx]

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

	// Refresh logic hooks
	// We can't easily hook into u.logs append from here without observer.
	// But u.log() calls u.logList.Refresh(). We should also refresh table.
	// I'll add a reference to table in UI struct or just refresh it on tab selection?
	// Better: Add table to UI struct.
	u.historyTable = table

	return container.NewBorder(headerContainer, nil, nil, nil, table)
}
