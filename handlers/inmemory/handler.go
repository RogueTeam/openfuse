//go:build openbsd

package inmemory

import (
	"fmt"
	"io/fs"

	"github.com/RogueTeam/openfuse/fuse"
	"golang.org/x/sys/unix"
)

type Node struct {
	Protected bool
	Name      string
	Parent    *Node
	Mode      fs.FileMode
	Children  map[string]*Node
	Buffer    []byte
}

func (n *Node) AddChild(c *Node) (hErr *fuse.HError) {
	if n.Mode&fs.ModeDir != fs.ModeDir {
		return &fuse.HError{
			Errno:   unix.ENOTDIR,
			GoError: fmt.Errorf("is not a directory"),
		}
	}

	return nil
}

type Handler struct {
	Root Node
}

var _ fuse.Handler = (*Handler)(nil)

func New() (h *Handler) {
	return &Handler{
		Root: Node{
			Protected: true,
			Name:      "",
			Mode:      0o700 | fs.ModeDir,
		},
	}
}
