//go:build openbsd

package fshandler

import (
	"io/fs"
	"syscall"

	"github.com/RogueTeam/openfuse/fuse"
	"github.com/RogueTeam/openfuse/handlers/fshandler/tables"
)

func FuseAttrFromFileInfo(ino uint64, info fs.FileInfo) (out fuse.FuseAttr) {
	if extInfo, ok := info.(ExtFileInfo); ok {
		return extInfo.Ext().toFuse(ino)
	}

	if sys, ok := info.Sys().(*syscall.Stat_t); ok {
		return fuse.FuseAttr{
			Ino:       ino,
			Mode:      sys.Mode,
			Rdev:      uint32(sys.Rdev),
			Nlink:     sys.Nlink,
			Uid:       sys.Uid,
			Gid:       sys.Gid,
			Atime:     uint64(sys.Atim.Sec),
			Mtime:     uint64(sys.Mtim.Sec),
			Ctime:     uint64(sys.Ctim.Sec),
			Atimesec:  uint32(sys.Atim.Nsec),
			Mtimesec:  uint32(sys.Mtim.Nsec),
			Ctimesec:  uint32(sys.Ctim.Nsec),
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
