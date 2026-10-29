package fshandler

import (
	"io/fs"
)

// Permits the handler to creates directories based on a path
type MkdirFS interface {
	fs.FS
	Mkdir(name string) (err error)
}

// Permits more complex opening of file by receiving the flags from kernel
type OpenFlagsFS interface {
	fs.FS
	OpenFlags(name string, flags uint32) (file fs.File, err error)
}

type ReadAtFile interface {
	fs.File
	ReadAt(b []byte, off int64) (n int, err error)
}
