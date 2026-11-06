//go:build openbsd

package memfs

import (
	"io/fs"

	"github.com/RogueTeam/openfuse/v8/fuse"
)

type Symlink struct {
	Reference *Node
	Oldname   string
}

var _ fs.File = (*Symlink)(nil)

func (s *Symlink) Stat() (info fs.FileInfo, err error) {
	return &s.Reference.Info, nil
}

func (s *Symlink) Read([]byte) (int, error) { return 0, fuse.ErrorNotSupported }

func (s *Symlink) Close() error { return fuse.ErrorNotSupported }
