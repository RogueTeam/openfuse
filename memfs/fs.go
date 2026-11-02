//go:build openbsd

package memfs

import (
	"bytes"
	"io/fs"
	"path"
	"strings"
	"time"

	"github.com/RogueTeam/openfuse/fuse"
	"github.com/RogueTeam/openfuse/handlers/fshandler"
	"golang.org/x/sys/unix"
)

type Node struct {
	Name           string
	Protected      bool
	Info           fileinfo
	Children       map[string]*Node
	Contents       []byte
	SymlinkOldname string
}

func (n *Node) DirEntry() (e fs.DirEntry) {
	return &direntry{
		name: n.Name,
		info: &n.Info,
	}
}

func (n *Node) openFile(flags uint32) (file *File) {
	defer func() {
		if file == nil {
			return
		}
	}()
	var contents []byte = n.Contents
	var cursor int
	if flags&unix.O_TRUNC == unix.O_TRUNC {
		contents = nil
	}
	if flags&unix.O_APPEND == unix.O_APPEND {
		cursor = len(contents)
	}
	switch {
	default:
		return nil
	case flags&unix.O_RDWR == unix.O_RDWR:
		return &File{AllowRead: true, AllowWrite: true, Reference: n, Info: n.Info, Cursor: cursor, Contents: bytes.Clone(contents)}
	case flags&unix.O_WRONLY == unix.O_WRONLY:
		return &File{AllowWrite: true, Reference: n, Info: n.Info, Cursor: cursor, Contents: bytes.Clone(contents)}
	case flags&unix.O_RDONLY == unix.O_RDONLY:
		return &File{AllowRead: true, Reference: n, Info: n.Info, Cursor: cursor, Contents: contents}
	}
}

func (n *Node) openSymlink() (symlink *Symlink) {
	switch {
	default:
		return &Symlink{
			Info:      n.Info,
			Reference: n,
			Oldname:   n.SymlinkOldname,
		}
	}
}

func (n *Node) Open(flags uint32) (file fs.File) {
	switch n.Info.Mode().Type() {
	case 0:
		return n.openFile(flags)
	case fs.ModeSymlink:
		return n.openSymlink()
	case fs.ModeDir:
		return &Directory{Reference: n, Info: n.Info}
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

	file = node.Open(unix.O_RDONLY)
	if file == nil {
		return nil, fuse.ErrorNotSupported
	}
	return file, nil
}

func (f *FS) OpenFlags(name string, flags uint32) (file fs.File, err error) {
	node, err := f.Root.Find(name)
	if err != nil {
		return nil, err
	}

	file = node.Open(flags)
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

	dir, target := path.Split(name)
	parent, err := f.Root.Find(dir)
	if err != nil {
		return err
	}

	if !parent.Info.Mode().IsDir() {
		return fuse.ErrorNotDirectory
	}

	_, found := parent.Children[target]
	if found {
		return fuse.ErrorExists
	}

	parent.Children[target] = &Node{
		Name: target,
		Info: fileinfo{
			name: target,
			attributes: fshandler.Attributes{
				Mode:  mode,
				Mtime: uint64(time.Now().Unix()),
			},
		},
		Children: make(map[string]*Node),
	}

	return nil
}

func (f *FS) Destroy() (err error) {
	return nil
}

func (f *FS) Unlink(name string) (err error) {
	dir, target := path.Split(name)
	parent, err := f.Root.Find(dir)
	if err != nil {
		return err
	}

	if !parent.Info.IsDir() {
		return fuse.ErrorNotDirectory
	}

	child, found := parent.Children[target]
	if !found {
		return fuse.ErrorNotFound
	}

	if child.Info.IsDir() {
		return fuse.ErrorIsDirectory
	}

	delete(parent.Children, target)

	return nil
}

func (f *FS) Setattr(name string, attr *fshandler.Setattr) (info fs.FileInfo, err error) {
	node, err := f.Root.Find(name)
	if err != nil {
		return nil, err
	}

	file := node.Open(unix.O_RDWR)

	setattrFile, ok := file.(fshandler.SetattrFile)
	if !ok {
		return nil, fuse.ErrorNotSupported
	}

	info, err = setattrFile.Setattr(attr)
	if err != nil {
		return nil, err
	}
	file.Close()

	return info, nil
}

func (f *FS) Rmdir(name string) (err error) {
	dir, target := path.Split(name)
	parent, err := f.Root.Find(dir)
	if err != nil {
		return err
	}

	if !parent.Info.IsDir() {
		return fuse.ErrorNotDirectory
	}

	child, found := parent.Children[target]
	if !found {
		return fuse.ErrorNotFound
	}

	if !child.Info.IsDir() {
		return fuse.ErrorNotDirectory
	}

	if len(child.Children) > 0 {
		return fuse.ErrorNotEmpty
	}

	delete(parent.Children, target)

	return nil
}

func (f *FS) Symlink(oldname, newname string) (err error) {
	dir, target := path.Split(newname)
	parent, err := f.Root.Find(dir)
	if err != nil {
		return err
	}

	if !parent.Info.IsDir() {
		return fuse.ErrorNotDirectory
	}

	_, found := parent.Children[target]
	if found {
		return fuse.ErrorExists
	}

	parent.Children[target] = &Node{
		Name: target,
		Info: fileinfo{
			name: target,
			attributes: fshandler.Attributes{
				Mode:  0o777 | unix.S_IFLNK,
				Mtime: uint64(time.Now().Unix()),
			},
		},
		SymlinkOldname: oldname,
	}

	return nil
}

func (f *FS) Readlink(name string) (oldname string, err error) {
	node, err := f.Root.Find(name)
	if err != nil {
		return "", err
	}

	if node.Info.Mode().Type() != fs.ModeSymlink {
		return "", fuse.ErrorInvalidValue
	}

	return node.SymlinkOldname, nil
}

var _ fshandler.FullFS = (*FS)(nil)

func New() (filesystem *FS) {
	return &FS{
		Root: Node{
			Protected: true,
			Info: fileinfo{
				name: "",
				attributes: fshandler.Attributes{
					Mode:  0o700 | unix.S_IFDIR,
					Mtime: uint64(time.Now().Unix()),
				},
			},
			Children: make(map[string]*Node),
		},
	}
}
