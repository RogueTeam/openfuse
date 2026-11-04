//go:build openbsd

package memfs

import (
	"io"
	"io/fs"
	"slices"
	"strings"

	"github.com/RogueTeam/openfuse/fuse"
	"github.com/RogueTeam/openfuse/handlers/fshandler"
)

type Directory struct {
	Closed    bool
	Cursor    int
	Info      FileInfo
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

func (d *Directory) ReadDir(n int) (dents []fs.DirEntry, err error) {
	if d.Closed {
		return nil, fs.ErrClosed
	}

	if d.Cursor >= len(d.Reference.Children) {
		return nil, io.EOF
	}

	dents = make([]fs.DirEntry, 0, len(d.Reference.Children))
	for _, entry := range d.Reference.Children {
		dents = append(dents, entry.DirEntry())
	}
	slices.SortFunc(dents, func(a, b fs.DirEntry) int { return strings.Compare(a.Name(), b.Name()) })

	dents = dents[d.Cursor:min(d.Cursor+n, len(dents))]
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

	dents = make([]fs.DirEntry, 0, len(d.Reference.Children))
	for _, entry := range d.Reference.Children {
		dents = append(dents, entry.DirEntry())
	}
	slices.SortFunc(dents, func(a, b fs.DirEntry) int { return strings.Compare(a.Name(), b.Name()) })

	dents = dents[off:min(off+uint64(n), uint64(len(dents)))]
	return dents, nil
}

func (d *Directory) ResetReadDir() (err error) {
	d.Cursor = 0
	return nil
}

func (d *Directory) Setattr(attr *fshandler.Setattr) (info fs.FileInfo, err error) {
	if attr.Valid&fuse.FuseFattrMode == fuse.FuseFattrMode {
		d.Info.Attributes.Mode = attr.Mode

	}
	if attr.Valid&fuse.FuseFattrUID == fuse.FuseFattrUID {
		d.Info.Attributes.Uid = attr.Uid

	}
	if attr.Valid&fuse.FuseFattrGID == fuse.FuseFattrGID {
		d.Info.Attributes.Gid = attr.Gid
	}
	if attr.Valid&fuse.FuseFattrAtime == fuse.FuseFattrAtime {
		d.Info.Attributes.Atime = attr.Atime
		d.Info.Attributes.Atimesec = attr.Atimensec
	}
	if attr.Valid&fuse.FuseFattrMtime == fuse.FuseFattrMtime {
		d.Info.Attributes.Mtime = attr.Mtime
		d.Info.Attributes.Mtimesec = attr.Mtimensec
	}

	return &d.Info, nil
}

var _ fshandler.FullDir = (*Directory)(nil)
