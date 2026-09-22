package mock

import (
	"bytes"
	"io/fs"
)

// DefaultContentType is reported for files stored without an explicit
// content type, including MemoryFile values seeded into a Bucket by hand.
const DefaultContentType = "application/octet-stream"

// MemoryFile is an object held in a mock Bucket and, separately, the reader
// handed out by Client.Get.
//
// The Client never hands out the stored object itself: every Get returns a
// fresh snapshot, so reading or closing one reader cannot disturb the stored
// bytes or any other reader. A MemoryFile that has been placed in a Bucket
// must therefore not be written to or closed afterwards.
type MemoryFile struct {
	bytes.Buffer

	// ContentType is what Client.Get reports for the file. Empty means
	// DefaultContentType.
	ContentType string

	closed bool
}

// NewMemoryFile builds a file holding a copy of data with the given content
// type, ready to be seeded into a Bucket.
func NewMemoryFile(data []byte, contentType string) *MemoryFile {
	f := &MemoryFile{ContentType: contentType}
	_, _ = f.Write(data)
	return f
}

// snapshot returns an independent copy of f positioned at its start.
func (f *MemoryFile) snapshot() *MemoryFile { return NewMemoryFile(f.Bytes(), f.ContentType) }

// contentType returns f.ContentType, defaulted when unset.
func (f *MemoryFile) contentType() string {
	if f.ContentType == "" {
		return DefaultContentType
	}
	return f.ContentType
}

// Close releases the file's contents. Reading afterwards fails with
// fs.ErrClosed, mirroring a real file handle.
func (f *MemoryFile) Close() error {
	f.closed = true
	f.Reset()
	return nil
}

// Read implements io.Reader.
func (f *MemoryFile) Read(p []byte) (int, error) {
	if f.closed {
		return 0, fs.ErrClosed
	}
	return f.Buffer.Read(p)
}
