package ui

type hostFileBackupUIState struct {
	Running   bool
	Progress  float64
	Status    string
	Result    string
	IsError   bool
	PathIndex int
	PathTotal int
}

func (u *UI) hostFileBackupState(profileID string) hostFileBackupUIState {
	u.hostFileBackupMu.Lock()
	defer u.hostFileBackupMu.Unlock()
	if u.hostFileBackupStates == nil {
		return hostFileBackupUIState{}
	}
	return u.hostFileBackupStates[profileID]
}

func (u *UI) setHostFileBackupRunning(profileID string, pathTotal int) {
	u.hostFileBackupMu.Lock()
	defer u.hostFileBackupMu.Unlock()
	if u.hostFileBackupStates == nil {
		u.hostFileBackupStates = map[string]hostFileBackupUIState{}
	}
	u.hostFileBackupStates[profileID] = hostFileBackupUIState{
		Running:   true,
		Progress:  0,
		Status:    "Starting file backup...",
		PathTotal: pathTotal,
	}
}

func (u *UI) updateHostFileBackupProgress(profileID string, progress float64, status string, pathIndex, pathTotal int) {
	u.hostFileBackupMu.Lock()
	defer u.hostFileBackupMu.Unlock()
	if u.hostFileBackupStates == nil {
		u.hostFileBackupStates = map[string]hostFileBackupUIState{}
	}
	state := u.hostFileBackupStates[profileID]
	state.Running = true
	state.Progress = progress
	state.Status = status
	state.PathIndex = pathIndex
	state.PathTotal = pathTotal
	u.hostFileBackupStates[profileID] = state
}

func (u *UI) finishHostFileBackup(profileID string, message string, isError bool) {
	u.hostFileBackupMu.Lock()
	defer u.hostFileBackupMu.Unlock()
	if u.hostFileBackupStates == nil {
		u.hostFileBackupStates = map[string]hostFileBackupUIState{}
	}
	u.hostFileBackupStates[profileID] = hostFileBackupUIState{
		Result:  message,
		IsError: isError,
	}
}

func (u *UI) isHostFileBackupRunning(profileID string) bool {
	u.hostFileBackupMu.Lock()
	defer u.hostFileBackupMu.Unlock()
	if u.hostFileBackupStates == nil {
		return false
	}
	return u.hostFileBackupStates[profileID].Running
}
