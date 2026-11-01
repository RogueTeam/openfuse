//go:build openbsd

package fshandler

import "io/fs"

type FullDir interface {
	fs.File
	fs.ReadDirFile
}
