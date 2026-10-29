package fshandler

import "io/fs"

// Permits the handler to creates directories based on a path
type MkdirFS interface {
	fs.FS
	Mkdir(name string) (err error)
}
