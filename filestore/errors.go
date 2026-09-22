// Package filestore defines the file storage abstraction shared by every
// storage provider, together with the sentinel errors that make up its
// error contract.
package filestore

import "errors"

var (
	// ErrFileExists is returned by Copy and Move when the destination
	// filename is already taken. Providers never overwrite an existing
	// destination implicitly; callers must Purge it first.
	ErrFileExists = errors.New("file already exists")

	// ErrNotFound is returned by Get, Move, Copy and GetPresignedURL when
	// the (source) filename does not exist. Purge is idempotent and does
	// not report a missing file.
	ErrNotFound = errors.New("file not found")
)
