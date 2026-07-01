package archiver

import (
	"fmt"

	"dback/backend/shell"
	"dback/models"
)

// Archiver describes a compression step in an archive pipeline.
type Archiver interface {
	CompressCommand() shell.Command
	Extension() string
}

type zstdArchiver struct{}

func (zstdArchiver) CompressCommand() shell.Command {
	return shell.Command{Binary: "zstd", Args: []string{"-1"}}
}

func (zstdArchiver) Extension() string { return ".tar.zst" }

type gzipArchiver struct{}

func (gzipArchiver) CompressCommand() shell.Command {
	return shell.Command{Binary: "gzip", Args: []string{"-1"}}
}

func (gzipArchiver) Extension() string { return ".tar.gz" }

// For returns the archiver for a compression setting.
func For(c models.ArchiveCompression) (Archiver, error) {
	switch models.NormalizeArchiveCompression(c) {
	case models.ArchiveCompressionGzip:
		return gzipArchiver{}, nil
	case models.ArchiveCompressionZstd:
		return zstdArchiver{}, nil
	default:
		return nil, fmt.Errorf("unsupported compression %q", c)
	}
}
