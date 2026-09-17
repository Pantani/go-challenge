package mock

import (
	"bytes"
	"errors"
)

type MemoryFile struct {
	bytes.Buffer
	closed bool
}

func (f *MemoryFile) Close() error {
	f.closed = true
	f.Reset()
	return nil
}

func (f *MemoryFile) Read(p []byte) (int, error) {
	if f.closed {
		return 0, errors.New("file is closed")
	}
	return f.Buffer.Read(p)
}
