//go:build openbsd

package memfs

import (
	"io"
	"io/fs"
	"slices"
	"strings"

	"github.com/RogueTeam/openfuse/v8/fuse"
	"github.com/RogueTeam/openfuse/v8/handlers/fshandler"
)

type Directory struct {
	Closed    bool
	Cursor    int
	Reference *Node
}

func (d *Directory) Stat() (info fs.FileInfo, err error) {
	if d.Closed {
		return nil, fs.ErrClosed
	}

	return &d.Reference.Info, nil
}

func (d *Directory) Read([]byte) (n int, err error) {
	if d.Closed {
		return 0, fs.ErrClosed
	}
	return 0, fuse.ErrorNotSupported
}

func (d *Directory) Close() (err error) {
	if d.Closed {
		return fs.ErrClosed
	}
	d.Closed = true
	return nil
}

func (d *Directory) prepareDirents() (dents []fs.DirEntry) {
	dents = make([]fs.DirEntry, 0, len(d.Reference.Children))
	for _, entry := range d.Reference.Children {
		dents = append(dents, entry.DirEntry())
	}
	slices.SortFunc(dents, func(a, b fs.DirEntry) int { return strings.Compare(a.Name(), b.Name()) })

	return dents
}

func (d *Directory) ReadDir(n int) (dents []fs.DirEntry, err error) {
	if d.Closed {
		return nil, fs.ErrClosed
	}

	if d.Cursor >= len(d.Reference.Children) {
		if n > 0 {
			return nil, io.EOF
		}
		return nil, nil
	}

	dents = d.prepareDirents()

	dents = dents[d.Cursor:]
	if n > 0 {
		upperLimit := min(n, len(dents))
		dents = dents[:upperLimit]
	}
	d.Cursor += len(dents)

	return dents, nil
}

func (d *Directory) ReadDirAt(off uint64, n int) (dents []fs.DirEntry, err error) {
	if d.Closed {
		return nil, fs.ErrClosed
	}

	if off >= uint64(len(d.Reference.Children)) {
		return nil, io.EOF
	}

	dents = d.prepareDirents()
	dents = dents[off:]
	if n > 0 {
		upperLimit := min(n, len(dents))
		dents = dents[:upperLimit]
	}
	return dents, nil
}

func (d *Directory) ResetReadDir() (err error) {
	d.Cursor = 0
	return nil
}

func (d *Directory) Setattr(attr *fshandler.Setattr) (info fs.FileInfo, err error) {
	if attr.Valid&fuse.FuseFattrMode == fuse.FuseFattrMode {
		oldMode := d.Reference.Info.Attributes.Mode
		d.Reference.Info.Attributes.Mode = (oldMode & ^uint32(0o7777)) | attr.Mode
	}
	if attr.Valid&fuse.FuseFattrUID == fuse.FuseFattrUID {
		d.Reference.Info.Attributes.Uid = attr.Uid
	}
	if attr.Valid&fuse.FuseFattrGID == fuse.FuseFattrGID {
		d.Reference.Info.Attributes.Gid = attr.Gid
	}
	if attr.Valid&fuse.FuseFattrAtime == fuse.FuseFattrAtime {
		d.Reference.Info.Attributes.Atime = attr.Atime
		d.Reference.Info.Attributes.Atimesec = attr.Atimensec
	}
	if attr.Valid&fuse.FuseFattrMtime == fuse.FuseFattrMtime {
		d.Reference.Info.Attributes.Mtime = attr.Mtime
		d.Reference.Info.Attributes.Mtimesec = attr.Mtimensec
	}

	return &d.Reference.Info, nil
}

var _ fshandler.FullDir = (*Directory)(nil)
