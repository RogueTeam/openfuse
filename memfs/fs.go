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
	Protected      bool
	Info           FileInfo
	Children       map[string]*Node
	Contents       []byte
	SymlinkOldname string
}

func sumStats(a, b fuse.FuseStatfsOut) (o fuse.FuseStatfsOut) {
	return fuse.FuseStatfsOut{
		Stat: fuse.FuseKstatfs{
			// Blocks: a.Stat.Blocks + b.Stat.Blocks,
			// Bfree:  a.Stat.Bfree + b.Stat.Bfree,
			// Bavail: a.Stat.Bavail + b.Stat.Bavail,
			Files: a.Stat.Files + b.Stat.Files,
			// Ffree:  a.Stat.Ffree + b.Stat.Ffree,
			// Bsize:   a.Stat.Bsize + b.Stat.Bsize,
			// Namelen: a.Stat.Namelen + b.Stat.Namelen,
			// Frsize:  a.Stat.Frsize + b.Stat.Frsize,
			// Padding: a.Stat.Padding + b.Stat.Padding,
			// Spare:   a.Stat.Spare + b.Stat.Spare,
		},
	}
}

func (n *Node) Statfs() (stats fuse.FuseStatfsOut) {
	// TODO: implement me
	stats.Stat.Files += uint64(len(n.Children))

	for _, child := range n.Children {
		stats = sumStats(stats, child.Statfs())
	}
	return stats
}

func (n *Node) DirEntry() (e fs.DirEntry) {
	return &Dirent{
		Reference: n,
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
		return &File{AllowRead: true, AllowWrite: true, Reference: n, Cursor: cursor, Contents: bytes.Clone(contents)}
	case flags&unix.O_WRONLY == unix.O_WRONLY:
		return &File{AllowWrite: true, Reference: n, Cursor: cursor, Contents: bytes.Clone(contents)}
	case flags&unix.O_RDONLY == unix.O_RDONLY:
		return &File{AllowRead: true, Reference: n, Cursor: cursor, Contents: contents}
	}
}

func (n *Node) openSymlink() (symlink *Symlink) {
	switch {
	default:
		return &Symlink{
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

		Info: FileInfo{
			SetName: target,
			Attributes: fshandler.Attributes{
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

		Info: FileInfo{
			SetName: target,
			Attributes: fshandler.Attributes{
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

func (f *FS) Rename(oldpath, newpath string) (err error) {
	oldDir, oldName := path.Split(oldpath)
	newDir, newName := path.Split(newpath)

	if strings.HasPrefix(newDir+"/", oldpath) {
		return fuse.ErrorExists
	}

	oldParent, err := f.Root.Find(oldDir)
	if err != nil {
		return err
	}

	if !oldParent.Info.IsDir() {
		return fuse.ErrorNotDirectory
	}

	target, found := oldParent.Children[oldName]
	if !found {
		return fuse.ErrorNotFound
	}

	newParent, err := f.Root.Find(newDir)
	if err != nil {
		return err
	}
	if !newParent.Info.IsDir() {
		return fuse.ErrorNotDirectory
	}

	existing, found := newParent.Children[newName]
	if !found {

		target.Info.SetName = newName
		newParent.Children[newName] = target
		delete(oldParent.Children, oldName)
		return nil
	}

	if existing.Info.Mode().Type() == target.Info.Mode().Type() {
		if existing.Info.IsDir() && len(existing.Children) > 0 {
			return fuse.ErrorExists
		}

		target.Info.SetName = newName
		newParent.Children[newName] = target
		delete(oldParent.Children, oldName)
		return nil
	}

	if existing.Info.IsDir() && !target.Info.IsDir() {
		return fuse.ErrorIsDirectory
	} else if !existing.Info.IsDir() && target.Info.IsDir() {
		return fuse.ErrorNotDirectory
	}
	target.Info.SetName = newName
	newParent.Children[newName] = target
	delete(oldParent.Children, oldName)
	return nil
}

func (f *FS) Statfs() (stats fuse.FuseStatfsOut) {
	return f.Root.Statfs()
}

func (f *FS) mknodFile(name string, _, umask, mode uint32) (err error) {
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

	perm := (mode & 0o777) & ^umask
	parent.Children[target] = &Node{
		Info: FileInfo{
			SetName: target,
			Attributes: fshandler.Attributes{
				Mode:  perm | (mode & ^uint32(0o777)) | unix.S_IFREG,
				Mtime: uint64(time.Now().Unix()),
			},
		},
	}

	return nil
}

func (f *FS) Mknod(name string, rdev uint32, umask uint32, mode uint32) (err error) {
	switch mode & fuse.S_IFModeType {
	case unix.S_IFREG:
		return f.mknodFile(name, rdev, umask, mode)
	default:
		return fuse.ErrorNotSupported
	}
}

func (f *FS) Link(oldname, newname string) (err error) {
	return fuse.ErrorNotSupported
}

var _ fshandler.FullFS = (*FS)(nil)

func New() (filesystem *FS) {
	return &FS{
		Root: Node{
			Protected: true,
			Info: FileInfo{
				SetName: "",
				Attributes: fshandler.Attributes{
					Mode:  0o700 | unix.S_IFDIR,
					Mtime: uint64(time.Now().Unix()),
				},
			},
			Children: make(map[string]*Node),
		},
	}
}
