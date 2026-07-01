package ui

import "testing"

func TestHostUploadStateTransitions(t *testing.T) {
	u := &UI{}

	if u.isHostUploadRunning("p1") {
		t.Fatal("expected no running upload initially")
	}

	u.setHostUploadRunning("p1")
	state := u.hostUploadState("p1")
	if !state.Running {
		t.Fatal("expected running after setHostUploadRunning")
	}
	if state.Progress != -1 {
		t.Fatalf("expected indeterminate progress, got %v", state.Progress)
	}
	if !u.isHostUploadRunning("p1") {
		t.Fatal("expected isHostUploadRunning true")
	}

	u.updateHostUploadProgress("p1", 0.5, "Uploading backup 1/2...")
	state = u.hostUploadState("p1")
	if state.Progress != 0.5 {
		t.Fatalf("expected progress 0.5, got %v", state.Progress)
	}
	if state.Status != "Uploading backup 1/2..." {
		t.Fatalf("unexpected status: %q", state.Status)
	}

	u.finishHostUpload("p1", "2 backups uploaded", false)
	state = u.hostUploadState("p1")
	if state.Running {
		t.Fatal("expected not running after finish")
	}
	if state.Result != "2 backups uploaded" {
		t.Fatalf("unexpected result: %q", state.Result)
	}
	if state.IsError {
		t.Fatal("expected success result")
	}
	if u.isHostUploadRunning("p1") {
		t.Fatal("expected isHostUploadRunning false after finish")
	}

	u.setHostUploadRunning("p1")
	state = u.hostUploadState("p1")
	if state.Result != "" {
		t.Fatal("expected previous result cleared when starting new upload")
	}
}
