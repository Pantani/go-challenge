package filestore

import (
	"context"
	"io"
	"time"
)

// FileProvider is the contract every storage backend implements.
//
// Errors are reported through the sentinels in this package (ErrNotFound,
// ErrFileExists). Providers may wrap them with additional context, so
// callers must compare them with errors.Is. Every operation checks its
// context before doing any work and returns ctx.Err() when it is done.
type FileProvider interface {
	// Get opens the named file. The caller owns the returned io.ReadCloser
	// and must close it. The second value is the file's content type.
	// It returns ErrNotFound when the file does not exist.
	Get(ctx context.Context, filename string) (io.ReadCloser, string, error)

	// Set stores fileBytes under filename with the given content type,
	// replacing any previous content.
	Set(ctx context.Context, filename string, fileBytes []byte, contentType string) error

	// Purge removes the named file. Purging a file that does not exist is
	// not an error.
	Purge(ctx context.Context, filename string) error

	// Move renames oldFilename to newFilename. It returns ErrNotFound when
	// the source is missing and ErrFileExists when the destination is
	// already taken (which includes moving a file onto itself), leaving
	// both files untouched in either case. When both conditions hold,
	// ErrFileExists takes precedence.
	Move(ctx context.Context, oldFilename, newFilename string) error

	// Copy duplicates oldFilename to newFilename, including its content
	// type, leaving the source in place. It returns ErrNotFound when the
	// source is missing and ErrFileExists when the destination is already
	// taken (which includes copying a file onto itself), leaving both files
	// untouched in either case. When both conditions hold, ErrFileExists
	// takes precedence.
	Copy(ctx context.Context, oldFilename, newFilename string) error
}

// PresignedFileProvider is a FileProvider that can also hand out time-limited
// URLs granting direct read access to a stored file.
type PresignedFileProvider interface {
	FileProvider

	// GetPresignedURL returns a URL that allows reading filename without
	// further authentication until expireAfter has elapsed. It returns
	// ErrNotFound when the file does not exist.
	GetPresignedURL(ctx context.Context, filename string, expireAfter time.Duration) (string, error)
}
