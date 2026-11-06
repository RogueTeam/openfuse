//go:build openbsd

package memfs

import (
	"io"
	"io/fs"
	"time"

	"github.com/RogueTeam/openfuse/v8/fuse"
	"github.com/RogueTeam/openfuse/v8/handlers/fshandler"
)

type File struct {
	AllowRead, AllowWrite bool
	Closed                bool
	Reference             *Node
	Cursor                int
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

	if f.Cursor >= len(f.Reference.Contents) {
		return 0, io.EOF
	}

	n = copy(b, f.Reference.Contents[f.Cursor:])
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

	if off >= int64(len(f.Reference.Contents)) {
		return 0, io.EOF
	}

	n = copy(b, f.Reference.Contents[off:min(off+int64(len(b)), int64(len(f.Reference.Contents)))])
	return n, nil
}

func (f *File) WriteAt(b []byte, off int64) (n int, err error) {
	if f.Closed {
		return 0, fs.ErrClosed
	}
	if !f.AllowWrite {
		return 0, fuse.ErrorBadFileDescriptor.GoError
	}
	if off < 0 {
		return 0, fuse.ErrorInvalidValue
	}
	if len(b) == 0 {
		return 0, nil
	}

	// this means if offset is 1 and we only have a buffer of size 1 or less
	// off cannot be used safely since it is out of bounds
	// Example inputs:
	// len(f.Contents) = 0
	// off = 1
	// out of bound index = f.Contents[1]
	//
	// Other scenario could be
	// len(f.Contents) = 1
	// off = 1
	// Out of bound index = f.Contents[1]
	if int64(len(f.Reference.Contents)) <= off {
		// Save enough space to store up to offset index + the size of the input buffer
		newBuffer := make([]byte, off+int64(len(b)))
		copy(newBuffer, f.Reference.Contents)
		f.Reference.Contents = newBuffer
	}

	sub := f.Reference.Contents[off:]
	if len(sub) < len(b) {
		newBuffer := make([]byte, len(f.Reference.Contents[:off])+len(b))
		copy(newBuffer, f.Reference.Contents[:off])
		f.Reference.Contents = newBuffer
		sub = f.Reference.Contents[off:]
	}

	n = copy(sub, b)

	f.Reference.Info.Attributes.Size = uint64(len(f.Reference.Contents))
	f.Reference.Info.Attributes.Mtime = uint64(time.Now().Unix())
	return n, nil
}

func (f *File) Sync() (err error) {
	if f.Closed {
		return fs.ErrClosed
	}
	return nil
}

func (f *File) Close() (err error) {
	err = f.Sync()
	if err != nil {
		return err
	}
	f.Closed = true
	return err
}

func (f *File) Setattr(attr *fshandler.Setattr) (info fs.FileInfo, err error) {
	if attr.Valid&fuse.FuseFattrSize == fuse.FuseFattrSize {
		switch {
		case uint64(len(f.Reference.Contents)) == attr.Size:
			break
		case uint64(len(f.Reference.Contents)) > attr.Size:
			f.Reference.Contents = f.Reference.Contents[:attr.Size]
		case uint64(len(f.Reference.Contents)) < attr.Size:
			newBuffer := make([]byte, attr.Size)
			copy(newBuffer, f.Reference.Contents)
			f.Reference.Contents = newBuffer
		}

		if f.Cursor >= len(f.Reference.Contents) {
			f.Cursor = len(f.Reference.Contents) - 1
		}
		f.Reference.Info.Attributes.Size = uint64(len(f.Reference.Contents))
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
