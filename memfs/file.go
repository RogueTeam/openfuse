//go:build openbsd

package memfs

import (
	"bytes"
	"io"
	"io/fs"
	"time"

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

	n = copy(b, f.Contents[f.Cursor:])
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

func (f *File) Sync() (err error) {
	if f.Closed {
		return fs.ErrClosed
	}

	if !f.AllowWrite {
		return nil
	}

	if !bytes.Equal(f.Contents, f.Reference.Contents) {
		f.Reference.Contents = f.Contents

		f.Reference.Info.Attributes.Size = uint64(len(f.Contents))
		f.Reference.Info.Attributes.Mtime = uint64(time.Now().Unix())
	}
	return
}

func (f *File) Close() (err error) {
	err = f.Sync()
	f.Closed = true
	return err
}

func (f *File) Setattr(attr *fshandler.Setattr) (info fs.FileInfo, err error) {
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
		f.Reference.Info.Attributes.Size = uint64(len(f.Contents))
	}

	if attr.Valid&fuse.FuseFattrMode == fuse.FuseFattrMode {
		oldMode := f.Reference.Info.Attributes.Mode
		f.Reference.Info.Attributes.Mode = (oldMode & ^uint32(0o7777)) | attr.Mode
	}
	if attr.Valid&fuse.FuseFattrUID == fuse.FuseFattrUID {
		f.Reference.Info.Attributes.Uid = attr.Uid

	}
	if attr.Valid&fuse.FuseFattrGID == fuse.FuseFattrGID {
		f.Reference.Info.Attributes.Gid = attr.Gid
	}
	if attr.Valid&fuse.FuseFattrAtime == fuse.FuseFattrAtime {
		f.Reference.Info.Attributes.Atime = attr.Atime
		f.Reference.Info.Attributes.Atimesec = attr.Atimensec
	}
	if attr.Valid&fuse.FuseFattrMtime == fuse.FuseFattrMtime {
		f.Reference.Info.Attributes.Mtime = attr.Mtime
		f.Reference.Info.Attributes.Mtimesec = attr.Mtimensec
	}

	return &f.Reference.Info, nil
}

var _ fshandler.FullFile = (*File)(nil)
