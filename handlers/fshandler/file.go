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
	Setattr(attr *Setattr) (out AttrOut, err error)
}

// Better stat used as a way to return more detailed file stats
type BetterStatFile interface {
	fs.File
	BetterStat() (out AttrOut, err error)
}

type FullDir interface {
	fs.File
	fs.ReadDirFile
	SetattrFile
	BetterStatFile
}

type FullFile interface {
	fs.File
	ReadAtFile
	WriteAtFile
	SetattrFile
	BetterStatFile
}
