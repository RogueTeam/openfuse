//go:build openbsd

package fshandler

import (
	"io/fs"
	"syscall"

	"github.com/RogueTeam/openfuse/fuse"
	"github.com/RogueTeam/openfuse/handlers/fshandler/tables"
	"golang.org/x/sys/unix"
)

func FuseAttrFromFileInfo(ino uint64, info fs.FileInfo) (out fuse.FuseAttr) {
	if extInfo, ok := info.(ExtFileInfo); ok {
		return extInfo.Ext().toFuse(ino)
	}

	if sys, ok := info.Sys().(*unix.Stat_t); ok {
		return fuse.FuseAttr{
			Ino:       ino,
			Mode:      sys.Mode,
			Rdev:      uint32(sys.Dev),
			Nlink:     sys.Nlink,
			Uid:       sys.Uid,
			Gid:       sys.Gid,
			Atime:     uint64(sys.Atim.Sec),
			Mtime:     uint64(sys.Mtim.Sec),
			Ctime:     uint64(sys.Ctim.Sec),
			Size:      uint64(sys.Size),
			Blocks:    uint64(sys.Blocks),
			Blocksize: uint32(sys.Blksize),
		}
	}
	if sys, ok := info.Sys().(*syscall.Stat_t); ok {
		return fuse.FuseAttr{
			Ino:       ino,
			Mode:      sys.Mode,
			Rdev:      uint32(sys.Dev),
			Nlink:     sys.Nlink,
			Uid:       sys.Uid,
			Gid:       sys.Gid,
			Atime:     uint64(sys.Atim.Sec),
			Mtime:     uint64(sys.Mtim.Sec),
			Ctime:     uint64(sys.Ctim.Sec),
			Size:      uint64(sys.Size),
			Blocks:    uint64(sys.Blocks),
			Blocksize: uint32(sys.Blksize),
		}
	}
	mode := uint32(info.Mode().Perm()) | tables.RawModeFromFileMode(info.Mode())
	return fuse.FuseAttr{
		Ino:   ino,
		Size:  uint64(info.Size()),
		Mode:  mode,
		Mtime: uint64(info.ModTime().Unix()),
	}
}
