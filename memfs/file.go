//go:build openbsd

package memfs

import (
	"bytes"
	"io"
	"io/fs"
	"time"

	"github.com/RogueTeam/openfuse/fuse"
	"github.com/RogueTeam/openfuse/handlers/fshandler"
	"github.com/RogueTeam/openfuse/handlers/tables"
)

type File struct {
	AllowRead, AllowWrite bool
	Closed                bool
	Info                  fileinfo
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

	if off > int64(len(f.Contents)) {
		return 0, io.EOF
	}

	sub := f.Contents[off:]
	if len(sub) < len(b) {
		newBuffer := make([]byte, len(f.Contents[:off])+len(b))
		copy(newBuffer, f.Contents[:off])
		f.Contents = newBuffer
		sub = f.Contents[off:]
	}

	n = copy(sub, b)
	return n, nil
}

func (f *File) Close() (err error) {
	if f.Closed {
		return fs.ErrClosed
	}
	f.Closed = true

	if !f.AllowWrite {
		return nil
	}

	if !bytes.Equal(f.Contents, f.Reference.Contents) {
		f.Reference.Contents = f.Contents

		f.Info.size = int64(len(f.Contents))
		f.Info.mtime = time.Now()
	}

	if f.Info != f.Reference.Info {
		f.Reference.Info = f.Info
	}
	return nil
}

func (f *File) Setattr(attr *fshandler.Setattr) (out fshandler.AttrOut, err error) {
	if attr.Valid&fuse.FuseFattrSize == fuse.FuseFattrSize {
		switch {
		case uint64(len(f.Contents)) == attr.Size:
			break
		case uint64(len(f.Contents)) > attr.Size:
			f.Contents = f.Contents[:attr.Size]
		case uint64(len(f.Contents)) < attr.Size:
			newBuffer := make([]byte, attr.Size)
			copy(newBuffer, f.Contents)
			f.Contents = newBuffer
		}

		if f.Cursor >= len(f.Contents) {
			f.Cursor = len(f.Contents) - 1
		}
		f.Info.size = int64(len(f.Contents))
	}

	if attr.Valid&fuse.FuseFattrMode == fuse.FuseFattrMode {
		// TODO: Implement me
	}
	if attr.Valid&fuse.FuseFattrUID == fuse.FuseFattrUID {
		// TODO: Implement me
	}
	if attr.Valid&fuse.FuseFattrGID == fuse.FuseFattrGID {
		// TODO: Implement me
	}
	if attr.Valid&fuse.FuseFattrAtime == fuse.FuseFattrAtime {
		// TODO: Implement me
	}
	if attr.Valid&fuse.FuseFattrMtime == fuse.FuseFattrMtime {
		// TODO: Implement me
	}

	return f.BetterStat()
}

func (f *File) BetterStat() (out fshandler.AttrOut, err error) {
	out = fshandler.AttrOut{
		Attr: fshandler.Attr{
			Size:      uint64(f.Info.size),
			Blocks:    1,
			Atime:     0,
			Mtime:     uint64(f.Info.ModTime().Unix()),
			Ctime:     0,
			Atimesec:  0,
			Mtimesec:  0,
			Ctimesec:  0,
			Mode:      uint32(f.Info.mode&0o777) | tables.ModeTypeFromFileMode(f.Info.mode),
			Nlink:     2,
			Uid:       0,
			Gid:       0,
			Rdev:      0,
			Blocksize: uint32(f.Info.size),
			Padding:   0,
		},
	}
	return out, nil
}

var _ fshandler.FullFile = (*File)(nil)
