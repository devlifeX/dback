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
)

type DialogState struct {
	Kind       DialogKind
	Title      string
	Message    string
	OKLabel    string
	HostUsages []models.TemplateHostUsage
	OnOK       func()
	OnCancel   func()
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
