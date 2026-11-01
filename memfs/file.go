//go:build openbsd

package memfs

import (
	"bytes"
	"io"
	"io/fs"

	"github.com/RogueTeam/openfuse/fuse"
	"github.com/RogueTeam/openfuse/handlers/fshandler"
	"github.com/RogueTeam/openfuse/handlers/tables"
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

	// TODO: Implement me!
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

func (f *File) Setattr(attr *fshandler.Setattr) (out fshandler.AttrOut, err error) {
	// TODO: Implement me!
	return out, fuse.ErrorNotSupported
}

func (f *File) BetterStat() (out fshandler.AttrOut, err error) {
	out = fshandler.AttrOut{
		Attr: fshandler.Attr{
			Size:      uint64(f.Reference.Info.size),
			Blocks:    1,
			Atime:     0,
			Mtime:     uint64(f.Reference.Info.ModTime().Unix()),
			Ctime:     0,
			Atimesec:  0,
			Mtimesec:  0,
			Ctimesec:  0,
			Mode:      uint32(f.Reference.Info.mode&0o777) | tables.ModeTypeFromFileMode(f.Reference.Info.mode),
			Nlink:     2,
			Uid:       0,
			Gid:       0,
			Rdev:      0,
			Blocksize: uint32(f.Reference.Info.size),
			Padding:   0,
		},
	}
	return out, nil
}

var _ fshandler.FullFile = (*File)(nil)
