package ui

import (
	"fmt"
	"time"

	"dback/models"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func (u *UI) createLogsTab() fyne.CanvasObject {
	// Initial logs display
	u.logList = widget.NewList(
		func() int {
			return len(u.logs)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("Log Entry")
		},
		func(i int, o fyne.CanvasObject) {
			entry := u.logs[len(u.logs)-1-i] // Newest first
			label := o.(*widget.Label)
			label.SetText(fmt.Sprintf("[%s] %s: %s (%s) %s", 
				entry.Timestamp.Format("15:04:05"), 
				entry.Action, 
				entry.Status, 
				entry.Details,
				entry.Error,
			))
		},
	)

	return container.NewBorder(nil, nil, nil, nil, u.logList)
}

func (u *UI) log(action, details, status, errStr string) {
	entry := models.LogEntry{
		Timestamp: time.Now(),
		Action:    action,
		Details:   details,
		Status:    status,
		Error:     errStr,
	}
	u.logs = append(u.logs, entry)
	if u.logList != nil {
		u.logList.Refresh()
	}
	// In a real app, save to disk here
}
