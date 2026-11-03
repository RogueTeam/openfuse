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
	Path     string
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

func (t *InodeTable) ChangeChildDirectory(n *Inode, olddir, newdir string) {
	delete(t.Paths, n.Path)

	n.Path = newdir + n.Path[len(olddir):]
	t.Paths[n.Path] = n

	for _, child := range n.Children {
		t.ChangeChildDirectory(child, olddir, newdir)
	}
}

// Changes the directory of this node and all its children
func (t *InodeTable) ChangeDirectory(n *Inode, newname string) {
	delete(t.Paths, n.Path)

	oldname := n.Path
	n.Path = newname
	t.Paths[n.Path] = n

	for _, child := range n.Children {
		t.ChangeChildDirectory(child, oldname, newname)
	}
}

// Creates a new inode by its path or retrieves a existing
func (t *InodeTable) New(path string, lookup, protected bool, parent *Inode) (inode *Inode) {
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
	}

	t.Paths[path] = inode
	t.Nodes[inode.NodeId] = inode

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

func (t *InodeTable) ForgetByPath(path string, nlookup uint64) {
	t.init()

	inode, found := t.Paths[path]
	if !found {
		return
	}
	t.ForgetByIno(inode.NodeId, nlookup)
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

	// Delete references
	delete(t.Nodes, ino)
	delete(t.Paths, inode.Path)
	return nil
}
