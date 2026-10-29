//go:build openbsd

package fshandler

import (
	"io/fs"
	"sync"
	"sync/atomic"
)

type FileHandle struct {
	Handle fs.File
	Inode  *Inode

	ReaddirCache []fs.DirEntry
}

type FileHandlesTable struct {
	once sync.Once
	next atomic.Uint64

	Handles map[uint64]*FileHandle
}

func (t *FileHandlesTable) init() {
	t.once.Do(func() {
		t.Handles = make(map[uint64]*FileHandle)
	})
}

// Get a file handle based on path or creates a new one
// For creating a new file handle pass 0 as argument
func (t *FileHandlesTable) New(inode *Inode, file fs.File) (fd uint64, fh *FileHandle) {
	t.init()

	fd = t.next.Add(1)
	fh = &FileHandle{
		Handle: file,
		Inode:  inode,
	}

	t.Handles[fd] = fh

	return fd, fh
}

// Retrieves nil if the handle was not found, otherwise return the handle
func (t *FileHandlesTable) GetByFd(fd uint64) (h *FileHandle) {
	h, _ = t.Handles[fd]
	return h
}
