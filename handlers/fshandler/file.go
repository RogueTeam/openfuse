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

type ReadDirAtFile interface {
	fs.File
	ReadDirAt(off uint64, n int) (dirent []fs.DirEntry, err error)
}

type SyncFile interface {
	fs.File
	Sync() (err error)
}

type FullDir interface {
	fs.File
	fs.ReadDirFile
	SetattrFile
	ReadDirAtFile
}

type FullFile interface {
	fs.File
	ReadAtFile
	WriteAtFile
	SetattrFile
	SyncFile
}

type ExtFileInfo interface {
	fs.FileInfo
	Ext() (extended *Attributes)
}
