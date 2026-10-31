//go:build openbsd

package fshandler

import "io/fs"

type FullFile interface {
	fs.File
	ReadAtFile
}

type FullDir interface {
	fs.File
	fs.ReadDirFile
}

type ReadAtFile interface {
	fs.File
	ReadAt(b []byte, off int64) (n int, err error)
}
