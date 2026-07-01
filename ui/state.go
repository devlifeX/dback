package ui

import "dback/models"

type Section int

const (
	SectionHosts Section = iota
	SectionBackups
	SectionTemplates
	SectionSettings
	SectionAbout
)

func (s Section) String() string {
	switch s {
	case SectionHosts:
		return "Hosts"
	case SectionBackups:
		return "Backups"
	case SectionTemplates:
		return "Templates"
	case SectionSettings:
		return "Settings"
	case SectionAbout:
		return "About"
	default:
		return ""
	}
}

type View int

const (
	ViewList View = iota
	ViewProfileEditor
	ViewTemplateEditor
	ViewBackupDetail
)

type DialogKind int

const (
	DialogNone DialogKind = iota
	DialogConfirm
	DialogError
	DialogInfo
	DialogLoading
	DialogPassword
	DialogTemplateReplace
	DialogSyncPushWarning
	DialogConnectionTest
	DialogUpdateAvailable
	DialogVerifyReport
	DialogDeepVerifyConfirm
	DialogRemoteUploadMissing
)

type DialogState struct {
	Kind                  DialogKind
	Title                 string
	Message               string
	OKLabel               string
	HostUsages            []models.TemplateHostUsage
	VerifyReport          []models.TableVerifyResult
	VerifyPassed          bool
	VerifyFingerprintMode string
	OnOK                  func()
	OnCancel              func()
}

type pendingRemoteUploadChoice struct {
	profile  models.Profile
	retryIDs []string
	staleIDs []string
	latestID string
}

type PendingFilePick struct {
	Kind     filePickKind
	OnFile   func(path string, data []byte)
	OnFolder func(path string)
	SaveName string
}

type filePickKind int

const (
	pickOpenFile filePickKind = iota
	pickSaveFile
	pickFolder
)
