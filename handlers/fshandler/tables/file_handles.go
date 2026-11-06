//go:build openbsd

package tables

import (
	"io/fs"
	"sync"
	"sync/atomic"

	"github.com/RogueTeam/openfuse/v8/fuse"
)

type FileHandle struct {
	FileDescriptor uint64
	Handle         fs.File
	Inode          *Inode

	ReaddirCache []fs.DirEntry
	CacheBuffer  []byte
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
func (t *FileHandlesTable) New(inode *Inode, file fs.File) (fh *FileHandle) {
	t.init()

	fh = &FileHandle{
		FileDescriptor: t.next.Add(1),
		Handle:         file,
		Inode:          inode,
	}

	t.Handles[fh.FileDescriptor] = fh

	return fh
}

// Retrieves nil if the handle was not found, otherwise return the handle
func (t *FileHandlesTable) GetByFd(fd uint64) (h *FileHandle) {
	h, _ = t.Handles[fd]
	return h
}

func (t *FileHandlesTable) Del(fd uint64) (hErr *fuse.HError) {
	_, found := t.Handles[fd]
	if !found {
		return fuse.ErrorNotFound
	}

	// Delete references
	delete(t.Handles, fd)

	return nil
}
