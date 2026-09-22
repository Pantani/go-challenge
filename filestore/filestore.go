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
// context on entry and returns ctx.Err() if already cancelled. Cancellation
// during later I/O depends on the provider and returned reader.
type FileProvider interface {
	// Get opens the named file. The caller owns the returned io.ReadCloser
	// and must close it. The second value is the file's content type.
	// It returns ErrNotFound when the file does not exist.
	// Cancellation during subsequent reads depends on the provider and returned
	// reader; the caller must close it even when reading fails.
	Get(ctx context.Context, filename string) (io.ReadCloser, string, error)

	// Set stores fileBytes under filename with the given content type,
	// replacing any previous content.
	Set(ctx context.Context, filename string, fileBytes []byte, contentType string) error

	// Purge removes the named file. Purging a file that does not exist is
	// not an error.
	Purge(ctx context.Context, filename string) error

	// Move transfers oldFilename to newFilename. A detected destination conflict
	// returns ErrFileExists before checking the source; an absent source returns
	// ErrNotFound. Atomicity is provider-specific. The S3-shaped implementation
	// performs a best-effort destination probe, copy, then source deletion; a
	// deletion failure can leave both objects. See the provider's contract.
	Move(ctx context.Context, oldFilename, newFilename string) error

	// Copy duplicates bytes and content type, leaving the source in place.
	// A detected destination conflict returns ErrFileExists before checking the
	// source; an absent source returns ErrNotFound. The S3-shaped provider's
	// destination check is best-effort and cannot prevent concurrent overwrites.
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
