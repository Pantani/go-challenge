package filestore

import "errors"

var (
	ErrFileExists = errors.New("file already exists")
	ErrNotFound   = errors.New("file not found")
)
