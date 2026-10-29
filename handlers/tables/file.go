//go:build openbsd

package tables

import (
	"io/fs"

	"github.com/RogueTeam/openfuse/fuse"
	"golang.org/x/sys/unix"
)

func ModeTypeFromFileMode(mode fs.FileMode) (o uint32) {
	switch mode.Type() {
	case 0:
		return unix.S_IFREG
	case fs.ModeDir:
		return unix.S_IFDIR
	case fs.ModeSymlink:
		return unix.S_IFLNK
	case fs.ModeNamedPipe:
		return unix.S_IFIFO
	case fs.ModeSocket:
		return unix.S_IFSOCK
	case fs.ModeDevice | fs.ModeCharDevice:
		return unix.S_IFCHR
	case fs.ModeDevice:
		return unix.S_IFBLK
	default: // case fs.ModeIrregular:
		return 0
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
