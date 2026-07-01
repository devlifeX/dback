package ui

type hostUploadUIState struct {
	Running  bool
	Progress float64 // 0..1, or -1 for indeterminate
	Status   string
	Result   string
	IsError  bool
}

func (u *UI) hostUploadState(profileID string) hostUploadUIState {
	u.hostUploadMu.Lock()
	defer u.hostUploadMu.Unlock()
	if u.hostUploadStates == nil {
		return hostUploadUIState{}
	}
	return u.hostUploadStates[profileID]
}

func (u *UI) setHostUploadRunning(profileID string) {
	u.hostUploadMu.Lock()
	defer u.hostUploadMu.Unlock()
	if u.hostUploadStates == nil {
		u.hostUploadStates = map[string]hostUploadUIState{}
	}
	u.hostUploadStates[profileID] = hostUploadUIState{
		Running:  true,
		Progress: -1,
		Status:   "Starting upload...",
	}
}

func (u *UI) updateHostUploadProgress(profileID string, progress float64, status string) {
	u.hostUploadMu.Lock()
	defer u.hostUploadMu.Unlock()
	if u.hostUploadStates == nil {
		u.hostUploadStates = map[string]hostUploadUIState{}
	}
	state := u.hostUploadStates[profileID]
	state.Running = true
	state.Progress = progress
	state.Status = status
	u.hostUploadStates[profileID] = state
}

func (u *UI) finishHostUpload(profileID string, message string, isError bool) {
	u.hostUploadMu.Lock()
	defer u.hostUploadMu.Unlock()
	if u.hostUploadStates == nil {
		u.hostUploadStates = map[string]hostUploadUIState{}
	}
	u.hostUploadStates[profileID] = hostUploadUIState{
		Result:  message,
		IsError: isError,
	}
}

func (u *UI) isHostUploadRunning(profileID string) bool {
	u.hostUploadMu.Lock()
	defer u.hostUploadMu.Unlock()
	if u.hostUploadStates == nil {
		return false
	}
	return u.hostUploadStates[profileID].Running
}
