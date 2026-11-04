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

// A ReadDirAtFile is a directory file whose entries can be read with the ReadDirAt method.
// Every directory file should implement this interface. If user want fresh directory listing on every call.
// (It is permissible for any file to implement this interface,
// but if so ReadDirAt should return an error for non-directories.)
type ReadDirAtFile interface {
	fs.File

	// ReadDirAt reads the contents of the directory and returns
	// a slice of up to n DirEntry values in directory order.
	// Subsequent calls on the same file will yield further DirEntry values.
	//
	// If n > 0, ReadDirAt returns at most n DirEntry structures.
	// In this case, if ReadDirAt returns an empty slice, it will return
	// a non-nil error explaining why.
	// At the end of a directory, the error is io.EOF.
	// (ReadDirAt must return io.EOF itself, not an error wrapping io.EOF.)
	//
	// If n <= 0, ReadDirAt returns all remaining DirEntry values from the directory
	// in a single slice. In this case, if ReadDirAt succeeds (reads all the way
	// to the end of the directory), it returns the slice and a nil error.
	// If it encounters an error before the end of the directory,
	// ReadDirAt returns the DirEntry list read until that point and a non-nil error.
	ReadDirAt(off uint64, n int) (dirent []fs.DirEntry, err error)
}

// Any directory implementing this reset interface should
// reset the internal cursor use by ReadDir() back to zero in order to force calls
// from the beginning of the file
// TODO: Verify seek isn't a better approach
type ResetReadDirFile interface {
	fs.File
	ResetReadDir() (err error)
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
	ResetReadDirFile
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
