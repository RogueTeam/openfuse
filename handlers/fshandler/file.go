//go:build openbsd

package fshandler

import (
	"io/fs"
)

type ReadAtFile interface {
	fs.File
	ReadAt(b []byte, off int64) (n int, err error)
}

type WriteAtFile interface {
	fs.File
	WriteAt(b []byte, off int64) (n int, err error)
}

type SetattrFile interface {
	fs.File
	Setattr(set *Setattr) (info fs.FileInfo, err error)
}

type FullDir interface {
	fs.File
	fs.ReadDirFile
	SetattrFile
}

type FullFile interface {
	fs.File
	ReadAtFile
	WriteAtFile
	SetattrFile
}

type ExtFileInfo interface {
	fs.FileInfo
	Ext() (extended *Attributes)
}
