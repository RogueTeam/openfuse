//go:build openbsd

package memfs

import (
	"bytes"
	"io"
	"io/fs"

	"github.com/RogueTeam/openfuse/fuse"
	"github.com/RogueTeam/openfuse/handlers/fshandler"
)

type File struct {
	AllowRead, AllowWrite bool
	Closed                bool
	Reference             *Node
	Cursor                int
	Contents              []byte
}

func (f *File) Stat() (info fs.FileInfo, err error) {
	if f.Closed {
		return nil, fs.ErrClosed
	}
	return &f.Reference.Info, nil
}

func (f *File) Read(b []byte) (n int, err error) {
	if f.Closed {
		return 0, fs.ErrClosed
	}
	if !f.AllowRead {
		return 0, fuse.ErrorBadFileDescriptor.GoError
	}

	n = copy(b[f.Cursor:], f.Contents)
	f.Cursor += n
	if n == 0 {
		return 0, io.EOF
	}
	return n, nil
}

func (f *File) ReadAt(b []byte, off int64) (n int, err error) {
	if f.Closed {
		return 0, fs.ErrClosed
	}
	if !f.AllowRead {
		return 0, fuse.ErrorBadFileDescriptor.GoError
	}

	if off >= int64(len(f.Contents)) {
		return 0, io.EOF
	}

	n = copy(b, f.Contents[off:min(off+int64(len(b)), int64(len(f.Contents)))])
	return n, nil
}

func (f *File) WriteAt(b []byte, off int64) (n int, err error) {
	if f.Closed {
		return 0, fs.ErrClosed
	}
	if !f.AllowWrite {
		return 0, fuse.ErrorBadFileDescriptor.GoError
	}

	return 0, fuse.ErrorNotSupported
}

func (f *File) Close() (err error) {
	if f.Closed {
		return fs.ErrClosed
	}
	f.Closed = true
	if bytes.Equal(f.Contents, f.Reference.Contents) {
		f.Reference.Contents = f.Contents
	}
	return nil
}

var _ fshandler.FullFile = (*File)(nil)
