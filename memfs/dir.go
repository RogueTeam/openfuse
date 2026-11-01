//go:build openbsd

package memfs

import (
	"fmt"
	"io"
	"io/fs"
	"slices"
	"strings"

	"github.com/RogueTeam/openfuse/fuse"
	"github.com/RogueTeam/openfuse/handlers/fshandler"
	"github.com/RogueTeam/openfuse/handlers/tables"
)

type Directory struct {
	Closed    bool
	Cursor    int
	Info      fileinfo
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

func (d *Directory) Setattr(attr *fshandler.Setattr) (out fshandler.AttrOut, err error) {
	fmt.Println("------------------------------ Dir | Setattr ------------------------------")
	fmt.Println(attr)
	// TODO: Implement me!
	return out, fuse.ErrorNotSupported
}

func (d *Directory) BetterStat() (out fshandler.AttrOut, err error) {
	out = fshandler.AttrOut{
		Attr: fshandler.Attr{
			Size:      0,
			Blocks:    0,
			Atime:     0,
			Mtime:     uint64(d.Info.ModTime().Unix()),
			Ctime:     0,
			Atimesec:  0,
			Mtimesec:  0,
			Ctimesec:  0,
			Mode:      uint32(d.Info.mode&0o777) | tables.ModeTypeFromFileMode(d.Info.mode),
			Nlink:     2,
			Uid:       0,
			Gid:       0,
			Rdev:      0,
			Blocksize: 0,
			Padding:   0,
		},
	}
	return out, nil
}

var _ fshandler.FullDir = (*Directory)(nil)
