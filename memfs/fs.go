//go:build openbsd

package memfs

import (
	"io/fs"
	"path"
	"strings"
	"time"

	"github.com/RogueTeam/openfuse/fuse"
	"github.com/RogueTeam/openfuse/handlers/fshandler"
	"github.com/RogueTeam/openfuse/handlers/tables"
	"golang.org/x/sys/unix"
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

func (n *Node) find(name []string) (node *Node, err error) {
	curr := n
	for _, part := range name {
		if part == "." {
			continue
		}
		if !curr.Info.IsDir() {
			return nil, fuse.ErrorNotDirectory
		}
		next, found := curr.Children[part]
		if !found {
			return nil, fuse.ErrorNotFound
		}
		curr = next
	}

	return curr, nil
}

func (n *Node) Find(name string) (node *Node, err error) {
	return n.find(strings.Split(path.Clean(name), "/"))
}

type FS struct {
	Root Node
}

func (f *FS) Open(name string) (file fs.File, err error) {
	node, err := f.Root.Find(name)
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
	node, err := f.Root.Find(name)
	if err != nil {
		return nil, err
	}

	return &node.Info, nil
}

func (f *FS) Mkdir(name string, mask, mode uint32) (err error) {
	if mode&unix.S_IFDIR != unix.S_IFDIR {
		return fuse.ErrorInvalidValue
	}

	dir, childDir := path.Split(name)
	inode, err := f.Root.Find(dir)
	if err != nil {
		return err
	}

	if !inode.Info.Mode().IsDir() {
		return fuse.ErrorNotDirectory
	}

	inode.Children[childDir] = &Node{
		Name: childDir,
		Info: fileinfo{
			name:  childDir,
			size:  0,
			mode:  fs.FileMode(mode&0o777) | tables.FileModeFromModeType(mode),
			mtime: time.Now(),
		},
		Children: make(map[string]*Node),
	}

	return nil
}

var (
	_ fs.FS             = (*FS)(nil)
	_ fs.StatFS         = (*FS)(nil)
	_ fshandler.MkdirFS = (*FS)(nil)
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
