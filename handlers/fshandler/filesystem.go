package fshandler

import (
	"io/fs"
)

// implements a complete filesystem with all features enabled
type FullFS interface {
	fs.FS
	fs.StatFS
	MkdirFS
	OpenFlagsFS
	DestroyFS
}

// Permits the handler to creates directories based on a path
type MkdirFS interface {
	fs.FS
	Mkdir(name string) (err error)
}

// Permits more complex opening of file by receiving the flags from kernel
// Check OpenBSD open flags for more detail
type OpenFlagsFS interface {
	fs.FS
	OpenFlags(name string, flags uint32) (file fs.File, err error)
}

// Permits the filesystem to run custom cleanup code when Destroy message is received from the kernel
type DestroyFS interface {
	fs.FS
	Destroy() (err error)
}
