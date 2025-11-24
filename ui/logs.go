package ui

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
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

func (u *UI) log(action, details, fileSize, status, errStr string) {
	entry := models.LogEntry{
		Timestamp: time.Now(),
		Action:    action,
		Details:   details,
		FileSize:  fileSize,
		Status:    status,
		Error:     errStr,
	}
	u.logs = append(u.logs, entry)

	// Refresh List
	if u.logList != nil {
		u.logList.Refresh()
	}

	// Refresh Table
	if u.historyTable != nil {
		u.historyTable.Refresh()
	}

	u.saveLogs()
}

func (u *UI) loadLogs() {
	file, err := os.Open("logs.json")
	if err != nil {
		return
	}
	defer file.Close()

	bytes, _ := ioutil.ReadAll(file)
	json.Unmarshal(bytes, &u.logs)
}

func (u *UI) saveLogs() {
	bytes, _ := json.MarshalIndent(u.logs, "", "  ")
	ioutil.WriteFile("logs.json", bytes, 0644)
}
