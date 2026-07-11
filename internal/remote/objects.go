package remote

import "time"

const BackupPrefix = "dback/backups/"

// ObjectEntry is one row in a remote storage listing.
type ObjectEntry struct {
	Key          string    `json:"key"`
	Name         string    `json:"name"`
	Size         int64     `json:"size"`
	IsDir        bool      `json:"is_dir"`
	LastModified time.Time `json:"last_modified,omitempty"`
}

// ObjectMeta describes a downloadable remote object.
type ObjectMeta struct {
	Key          string
	Size         int64
	ContentType  string
	LastModified time.Time
}
