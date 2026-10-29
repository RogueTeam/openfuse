//go:build openbsd

package fshandler

import (
	"sync"
	"sync/atomic"

	"github.com/RogueTeam/gopenfuse/fuse"
)

type Inode struct {
	NodeId uint64
	// Special condition that prevents an inode from being removed
	Protected bool
	// Path of the node
	Path     string
	Parent   *Inode
	Children map[uint64]*Inode
}

type InodeTable struct {
	once sync.Once
	next atomic.Uint64

	Paths map[string]*Inode
	Nodes map[uint64]*Inode
}

func (t *InodeTable) init() {
	t.once.Do(func() {
		t.Paths = make(map[string]*Inode)
		t.Nodes = make(map[uint64]*Inode)
	})
}

// Get a inode based on path or creates a new one
func (t *InodeTable) Get(path string, protected bool, parent *Inode) (inode *Inode) {
	t.init()

	inode, found := t.Paths[path]
	if found {
		return inode
	}

	inode = &Inode{
		NodeId:    t.next.Add(1),
		Protected: protected,
		Path:      path,
	}

	t.Paths[path] = inode
	t.Nodes[inode.NodeId] = inode

	return inode
}

// Get an inode by its node id, returns nil if nothing was found
func (t *InodeTable) GetByIno(ino uint64) (inode *Inode) {
	t.init()

	inode, _ = t.Nodes[ino]
	return inode
}

func (t *InodeTable) DelByPath(path string) {
	inode, found := t.Paths[path]
	if !found {
		return
	}
	t.Del(inode.NodeId)
}

func (t *InodeTable) Del(ino uint64) (hErr *fuse.HError) {
	inode, found := t.Nodes[ino]
	if !found {
		return fuse.ErrorNotFound
	}

	if inode.Protected {
		return fuse.ErrorPermissionDenied
	}

	// Delete references
	delete(t.Nodes, ino)
	delete(t.Paths, inode.Path)

	// TODO: Delete any reference from parent
	// TODO: Delete references from children
	return nil
}
