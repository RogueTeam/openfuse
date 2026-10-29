//go:build openbsd

package tables

import (
	"io/fs"

	"github.com/RogueTeam/gopenfuse/fuse"
	"golang.org/x/sys/unix"
)

func ModeTypeFromFileMode(mode fs.FileMode) (o uint32) {
	switch mode.Type() {
	case fs.ModeDir:
		return unix.S_IFDIR
	case fs.ModeSymlink:
		return unix.S_IFLNK
	case fs.ModeNamedPipe:
		return unix.S_IFIFO
	case fs.ModeSocket:
		return unix.S_IFSOCK
	case fs.ModeDevice:
		return unix.S_IFBLK
	case fs.ModeCharDevice:
		return unix.S_IFCHR
	default: // case fs.ModeIrregular:
		return unix.S_IFREG
	}
}

func FuseAttrFromFileInfo(header *fuse.FuseInHeader, info fs.FileInfo) (out fuse.FuseAttr) {
	mode := uint32(info.Mode().Perm()) | ModeTypeFromFileMode(info.Mode())
	return fuse.FuseAttr{
		Ino:   header.NodeId,
		Size:  uint64(info.Size()),
		Mode:  mode,
		Mtime: uint64(info.ModTime().UnixMilli()),
	}
}
