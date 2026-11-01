//go:build openbsd

package fshandler

import (
	"io/fs"

	"github.com/RogueTeam/openfuse/fuse"
)

type Attr struct {
	Size      uint64
	Blocks    uint64
	Atime     uint64
	Mtime     uint64
	Ctime     uint64
	Atimesec  uint32
	Mtimesec  uint32
	Ctimesec  uint32
	Mode      uint32
	Nlink     uint32
	Uid       uint32
	Gid       uint32
	Rdev      uint32
	Blocksize uint32
	Padding   uint32
}

type AttrOut struct {
	Attr Attr
}

// Method reserved by fshandler
func (a *AttrOut) ToFuse(ino uint64) (out fuse.FuseAttrOut) {
	return fuse.FuseAttrOut{
		Attr: fuse.FuseAttr{
			Ino:       ino,
			Size:      a.Attr.Size,
			Blocks:    a.Attr.Blocks,
			Atime:     a.Attr.Atime,
			Mtime:     a.Attr.Mtime,
			Ctime:     a.Attr.Ctime,
			Atimesec:  a.Attr.Atimesec,
			Mtimesec:  a.Attr.Mtimesec,
			Ctimesec:  a.Attr.Ctimesec,
			Mode:      a.Attr.Mode,
			Nlink:     a.Attr.Nlink,
			Uid:       a.Attr.Uid,
			Gid:       a.Attr.Gid,
			Rdev:      a.Attr.Rdev,
			Blocksize: a.Attr.Blocksize,
			Padding:   a.Attr.Padding,
		},
	}
}

type Setattr struct {
	// Valid is a specialized field intended to let the caller know
	// which other fields to consider
	Valid      fuse.SetattrValid
	Padding    uint32
	FileHandle fs.File
	Size       uint64
	LockOwner  uint64
	Atime      uint64
	Mtime      uint64
	Unused2    uint64
	Atimensec  uint32
	Mtimensec  uint32
	Unused3    uint32
	Mode       uint32
	Unused4    uint32
	Uid        uint32
	Gid        uint32
	Unused5    uint32
}

func (s *Setattr) FromFuse(fh fs.File, in *fuse.FuseSetattrIn) {
	s.Valid = in.Valid
	s.Padding = in.Padding
	s.FileHandle = fh
	s.Size = in.Size
	s.LockOwner = in.LockOwner
	s.Atime = in.Atime
	s.Mtime = in.Mtime
	s.Unused2 = in.Unused2
	s.Atimensec = in.Atimensec
	s.Mtimensec = in.Mtimensec
	s.Unused3 = in.Unused3
	s.Mode = in.Mode
	s.Unused4 = in.Unused4
	s.Uid = in.Uid
	s.Gid = in.Gid
	s.Unused5 = in.Unused5
}

// Permits the handler to creates directories based on a path
type MkdirFS interface {
	fs.FS
	Mkdir(name string, mask, mode uint32) (err error)
}

// Permits more complex opening of file by receiving the flags from kernel
// Check OpenBSD open flags for more detail
type OpenFlagsFS interface {
	fs.FS
	OpenFlags(name string, flags uint32) (file fs.File, err error)
}

// Permits the filesystem to run custom cleanup code when Destroy message is received from the kernel
type DestroyFS interface {
	fs.FS
	Destroy() (err error)
}

type UnlinkFS interface {
	fs.FS
	Unlink(name string) (err error)
}

type SetattrFS interface {
	fs.FS
	Setattr(name string, attr *Setattr) (out AttrOut, err error)
}

// Better stat used as a way to return more detailed file stats
type BetterStatFS interface {
	fs.FS
	BetterStat(name string) (out AttrOut, err error)
}

// implements a complete filesystem with all features enabled
type FullFS interface {
	fs.FS
	fs.StatFS
	MkdirFS
	OpenFlagsFS
	DestroyFS
	UnlinkFS
	SetattrFS
	BetterStatFS
}
