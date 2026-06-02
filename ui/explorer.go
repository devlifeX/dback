package ui

import (
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

func (u *UI) pickOpenFile(onFile func(path string, data []byte)) {
	go func() {
		rc, err := u.explorer.ChooseFile()
		if err != nil {
			return
		}
		defer rc.Close()
		data, err := io.ReadAll(rc)
		if err != nil {
			return
		}
		path := filePathFromReadCloser(rc)
		onFile(path, data)
		u.invalidate()
	}()
}

func (u *UI) pickFolder(onFolder func(path string)) {
	go func() {
		path, err := chooseFolderDialog()
		if err != nil || path == "" {
			return
		}
		onFolder(path)
		u.invalidate()
	}()
}

func (u *UI) pickSaveFile(name string, onSave func(path string)) {
	go func() {
		wc, err := u.explorer.CreateFile(name)
		if err != nil {
			return
		}
		path := filePathFromWriteCloser(wc)
		_ = wc.Close()
		onSave(path)
		u.invalidate()
	}()
}

func (u *UI) pickSaveBytes(name string, data []byte, onSave func(path string)) {
	go func() {
		wc, err := u.explorer.CreateFile(name)
		if err != nil {
			return
		}
		path := filePathFromWriteCloser(wc)
		if path != "" {
			_ = wc.Close()
			if err := os.WriteFile(path, data, 0644); err != nil {
				return
			}
		} else {
			_, _ = wc.Write(data)
			_ = wc.Close()
		}
		onSave(path)
		u.invalidate()
	}()
}

func filePathFromReadCloser(rc io.ReadCloser) string {
	if f, ok := rc.(*os.File); ok {
		return f.Name()
	}
	return "selected file"
}

func filePathFromWriteCloser(wc io.WriteCloser) string {
	if f, ok := wc.(*os.File); ok {
		return f.Name()
	}
	return ""
}

func chooseFolderDialog() (string, error) {
	switch runtime.GOOS {
	case "linux":
		if path, err := runDialog("zenity", "--file-selection", "--directory"); err == nil {
			return path, nil
		}
		if path, err := runDialog("kdialog", "--getexistingdirectory"); err == nil {
			return path, nil
		}
	case "darwin":
		return runDialog("osascript", "-e", `POSIX path of (choose folder with prompt "Select folder")`)
	case "windows":
		return runDialog("powershell", "-NoProfile", "-Command", `(New-Object -ComObject Shell.Application).BrowseForFolder(0,'Select folder',0).Self.Path`)
	}
	return "", os.ErrInvalid
}

func runDialog(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
