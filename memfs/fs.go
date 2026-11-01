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

func (n *Node) Open(flags uint32) (file fs.File) {
	switch n.Info.Mode().Type() {
	case 0:
		return n.openFile(flags)
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

func (f *FS) Destroy() (err error) {
	return nil
}

func (f *FS) Unlink(name string) (err error) {
	dir, filename := path.Split(name)
	parent, err := f.Root.Find(dir)
	if err != nil {
		return err
	}

	if !parent.Info.IsDir() {
		return fuse.ErrorNotDirectory
	}

	_, found := parent.Children[filename]
	if !found {
		return fuse.ErrorNotFound
	}

	delete(parent.Children, filename)

	return nil
}

func (f *FS) Setattr(name string, attr *fshandler.Setattr) (out fshandler.AttrOut, err error) {
	node, err := f.Root.Find(name)
	if err != nil {
		return fshandler.AttrOut{}, err
	}

	file := node.Open(unix.O_RDWR)

	setattrFile, ok := file.(fshandler.SetattrFile)
	if !ok {
		return out, fuse.ErrorNotSupported
	}

	out, err = setattrFile.Setattr(attr)
	if err != nil {
		return fshandler.AttrOut{}, err
	}

	err = file.Close()
	if err != nil {
		return fshandler.AttrOut{}, err
	}

	return out, nil
}

func (f *FS) BetterStat(name string) (out fshandler.AttrOut, err error) {
	node, err := f.Root.Find(name)
	if err != nil {
		return out, err
	}

	out = fshandler.AttrOut{
		Attr: fshandler.Attr{
			Size:      uint64(node.Info.size),
			Blocks:    1,
			Atime:     0,
			Mtime:     uint64(node.Info.ModTime().Unix()),
			Ctime:     0,
			Atimesec:  0,
			Mtimesec:  0,
			Ctimesec:  0,
			Mode:      uint32(node.Info.mode&0o777) | tables.ModeTypeFromFileMode(node.Info.mode),
			Nlink:     2,
			Uid:       0,
			Gid:       0,
			Rdev:      0,
			Blocksize: uint32(node.Info.size),
			Padding:   0,
		},
	}
	return out, nil
}

var _ fshandler.FullFS = (*FS)(nil)

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
