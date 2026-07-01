package capability

import (
	"dback/backend/builder"
	"dback/backend/shell"
	"dback/models"
)

// ArchiveOptions holds inputs for a filesystem archive plan.
type ArchiveOptions struct {
	RootPath    string
	Excludes    []string
	Compression models.ArchiveCompression
	Mode        shell.ExecutionMode
}

// FilesystemProvider plans filesystem archive operations.
type FilesystemProvider interface {
	PlanArchive(opts ArchiveOptions) (shell.ExecutionPlan, error)
}

type tarArchivePlanner struct{}

// DefaultFilesystemProvider returns the standard tar archive planner.
func DefaultFilesystemProvider() FilesystemProvider {
	return tarArchivePlanner{}
}

func (tarArchivePlanner) PlanArchive(opts ArchiveOptions) (shell.ExecutionPlan, error) {
	return builder.BuildTarArchivePlan(opts.RootPath, opts.Excludes, opts.Compression, opts.Mode)
}
