//go:build openbsd

package memfs

import (
	"fmt"
	"io/fs"
	"strings"
	"time"

	"github.com/RogueTeam/openfuse/fuse"
)

type Node struct {
	Name      string
	Protected bool
	Info      fileinfo
	Children  map[string]*Node
	Contents  []byte
}

func (n *Node) DirEntry() (e fs.DirEntry) {
	return &direntry{
		name: n.Name,
		info: &n.Info,
	}
}

func (n *Node) Open() (file fs.File) {
	switch n.Info.Mode().Type() {
	case 0:
		return &File{Reference: n, Contents: n.Contents}
	case fs.ModeDir:
		return &Directory{Reference: n}
	default:
		return nil
	}
}

func (n *Node) Find(name []string) (node *Node, err error) {
	if !n.Info.IsDir() {
		return nil, fuse.ErrorNotDirectory
	}

	fmt.Println(name)
	return nil, fuse.ErrorNotSupported
}

type FS struct {
	Root Node
}

func (f *FS) Open(name string) (file fs.File, err error) {
	node, err := f.Root.Find(strings.Split(name, "/"))
	if err != nil {
		return nil, err
	}

	file = node.Open()
	if file == nil {
		return nil, fuse.ErrorNotSupported
	}
	return file, nil
}

func (f *FS) Stat(name string) (fs.FileInfo, error) {
	node, err := f.Root.Find(strings.Split(name, "/"))
	if err != nil {
		return nil, err
	}

	return &node.Info, nil
}

var (
	_ fs.FS     = (*FS)(nil)
	_ fs.StatFS = (*FS)(nil)
	// _ fshandler.MkdirFS     = (*FS)(nil)
	// _ fshandler.OpenFlagsFS = (*FS)(nil)
	// _ fshandler.DestroyFS   = (*FS)(nil)
)

func New() (filesystem *FS) {
	return &FS{
		Root: Node{
			Protected: true,
			Info: fileinfo{
				name:  "",
				mode:  0o700 | fs.ModeDir,
				mtime: time.Now(),
			},
			Children: make(map[string]*Node),
		},
	}
}
