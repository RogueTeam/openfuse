//go:build openbsd

package fshandler

import "io/fs"

type ReadAtFile interface {
	fs.File
	ReadAt(b []byte, off int64) (n int, err error)
}
