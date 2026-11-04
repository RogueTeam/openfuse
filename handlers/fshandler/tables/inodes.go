//go:build openbsd

package tables

import (
	"github.com/RogueTeam/openfuse/fuse"
)

type Inode struct {
	NodeId uint64
	// Counter for every lookup/forget
	Viewers uint64
	// Special condition that prevents an inode from being removed
	Protected bool
	// Path of the node
	Path string

	Parent   *Inode
	Children map[uint64]*Inode
}

type InodeTable struct {
	next uint64

	Paths map[string]*Inode
	Nodes map[uint64]*Inode
}

func (t *InodeTable) init() {
	if t.Paths == nil {
		t.Paths = make(map[string]*Inode)
	}
	if t.Nodes == nil {
		t.Nodes = make(map[uint64]*Inode)
	}
}

func (t *InodeTable) changeChildrenDirectory(n *Inode, olddir, newdir string) {
	for _, child := range n.Children {
		oldchildname := child.Path
		child.Path = newdir + child.Path[len(olddir):]
		delete(t.Paths, oldchildname)
		t.Paths[child.Path] = child

		t.changeChildrenDirectory(child, olddir, newdir)
	}
}

// Changes the directory of this node and all its children
func (t *InodeTable) ChangeDirectory(newParent, n *Inode, newname string) {
	delete(t.Paths, n.Path)
	if n.Parent != nil {
		delete(n.Parent.Children, n.NodeId)
	}

	oldname := n.Path
	n.Path = newname
	n.Parent = newParent
	t.Paths[n.Path] = n
	newParent.Children[n.NodeId] = n

	t.changeChildrenDirectory(n, oldname, newname)
}

// Creates a new inode by its path or retrieves a existing
func (t *InodeTable) New(parent *Inode, path string, lookup, protected bool) (inode *Inode) {
	t.init()

	defer func() {
		if inode != nil && lookup {
			inode.Viewers++
		}
	}()

	inode, found := t.Paths[path]
	if found {
		return inode
	}

	t.next++
	inode = &Inode{
		NodeId:    t.next,
		Protected: protected,
		Path:      path,
		Parent:    parent,
		Children:  make(map[uint64]*Inode),
	}

	t.Paths[path] = inode
	t.Nodes[inode.NodeId] = inode
	if parent != nil {
		parent.Children[t.next] = inode
	}
	return inode
}

func (t *InodeTable) GetByPath(path string) (inode *Inode) {
	t.init()

	inode, _ = t.Paths[path]
	return inode
}

func (t *InodeTable) GetByIno(ino uint64) (inode *Inode) {
	t.init()

	inode, _ = t.Nodes[ino]
	return inode
}

func (t *InodeTable) ForgetByIno(ino uint64, nlookup uint64) (hErr *fuse.HError) {
	t.init()

	inode, found := t.Nodes[ino]
	if !found {
		return fuse.ErrorNotFound
	}

	if inode.Viewers < nlookup {
		inode.Viewers = 0
	} else {
		inode.Viewers -= nlookup
	}
	if inode.Protected {
		return fuse.ErrorPermissionDenied
	}

	// There are still viewers for this inode
	if inode.Viewers > 0 {
		return nil
	}

	// Only completely wipe those nodes with no child assigned
	if len(inode.Children) > 0 {
		return
	}

	// Delete references
	delete(t.Nodes, ino)
	delete(t.Paths, inode.Path)
	if inode.Parent != nil {
		delete(inode.Parent.Children, ino)
	}
	return nil
}
