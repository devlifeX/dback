package app

import (
	"sort"
	"time"

	"dback/models"
)

func sortLogsNewestFirst(logs []models.LogEntry) {
	sort.Slice(logs, func(i, j int) bool {
		return logs[i].Timestamp.After(logs[j].Timestamp)
	})
}

func sortHistoryNewestFirst(history []models.ExportRecord) {
	sort.Slice(history, func(i, j int) bool {
		return history[i].ExportDate.After(history[j].ExportDate)
	})
}

func sortProfilesNewestFirst(profiles []models.Profile) {
	sort.Slice(profiles, func(i, j int) bool {
		return profiles[i].ModifiedAt().After(profiles[j].ModifiedAt())
	})
}

func sortUsersNewestFirst(users []models.User) {
	sort.Slice(users, func(i, j int) bool {
		return userSortTime(users[i]).After(userSortTime(users[j]))
	})
}

func userSortTime(u models.User) time.Time {
	if !u.UpdatedAt.IsZero() {
		return u.UpdatedAt
	}
	return u.CreatedAt
}
