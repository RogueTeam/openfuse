//go:build openbsd

package fshandler

import (
	"io/fs"
	"time"

	"github.com/RogueTeam/openfuse/fuse"
	"golang.org/x/sys/unix"
)

type Attributes struct {
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

func (a *Attributes) FileMode() (mode fs.FileMode) {
	mode = fs.FileMode(a.Mode & 0o777)
	switch a.Mode & fuse.S_IFModeType {
	case unix.S_IFLNK:
		mode |= fs.ModeSymlink
	case unix.S_IFDIR:
		mode |= fs.ModeDir
	case unix.S_IFIFO:
		mode |= fs.ModeNamedPipe
	case unix.S_IFSOCK:
		mode |= fs.ModeSocket
	case unix.S_IFCHR:
		mode |= fs.ModeDevice | fs.ModeCharDevice
	case unix.S_IFBLK:
		mode |= fs.ModeDevice
	case unix.S_IFREG:
		mode |= 0
	default:
		mode |= fs.ModeIrregular
	}
	return mode
}

func (a *Attributes) ModTime() (mtime time.Time) {
	if a.Mtime != 0 {
		return time.Unix(int64(a.Mtime), 0)
	} else if a.Mtimesec != 0 {
		return time.Unix(int64(a.Mtimesec), 0)
	}
	return mtime
}

// Method reserved by fshandler
func (a *Attributes) toFuse(ino uint64) (out fuse.FuseAttr) {
	return fuse.FuseAttr{
		Ino:       ino,
		Size:      a.Size,
		Blocks:    a.Blocks,
		Atime:     a.Atime,
		Mtime:     a.Mtime,
		Ctime:     a.Ctime,
		Atimesec:  a.Atimesec,
		Mtimesec:  a.Mtimesec,
		Ctimesec:  a.Ctimesec,
		Mode:      a.Mode,
		Nlink:     a.Nlink,
		Uid:       a.Uid,
		Gid:       a.Gid,
		Rdev:      a.Rdev,
		Blocksize: a.Blocksize,
		Padding:   a.Padding,
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

func (s *Setattr) fromFuse(fh fs.File, in *fuse.FuseSetattrIn) {
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
	Setattr(name string, set *Setattr) (info fs.FileInfo, err error)
}

type RmdirFS interface {
	fs.FS
	Rmdir(name string) (err error)
}

type SymlinkFS interface {
	fs.FS
	Symlink(oldname, newname string) (err error)
	Readlink(name string) (oldname string, err error)
}

type RenameFS interface {
	fs.FS
	Rename(oldname, newname string) (err error)
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
	RmdirFS
	SymlinkFS
	RenameFS
}
