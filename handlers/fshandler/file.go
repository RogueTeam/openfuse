//go:build openbsd

package fshandler

import "io/fs"

type FullFile interface {
	fs.File
	ReadAtFile
	WriteAtFile
}

type ReadAtFile interface {
	fs.File
	ReadAt(b []byte, off int64) (n int, err error)
}

type WriteAtFile interface {
	fs.File
	WriteAt(b []byte, off int64) (n int, err error)
}
