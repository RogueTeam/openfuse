//go:build openbsd

package tables

import (
	"io/fs"

	"golang.org/x/sys/unix"
)

func RawModeFromFileMode(mode fs.FileMode) (o uint32) {
	switch mode.Type() {
	case 0:
		return uint32(mode.Perm()) | unix.S_IFREG
	case fs.ModeDir:
		return uint32(mode.Perm()) | unix.S_IFDIR
	case fs.ModeSymlink:
		return uint32(mode.Perm()) | unix.S_IFLNK
	case fs.ModeNamedPipe:
		return uint32(mode.Perm()) | unix.S_IFIFO
	case fs.ModeSocket:
		return uint32(mode.Perm()) | unix.S_IFSOCK
	case fs.ModeDevice | fs.ModeCharDevice:
		return uint32(mode.Perm()) | unix.S_IFCHR
	case fs.ModeDevice:
		return uint32(mode.Perm()) | unix.S_IFBLK
	default: // case fs.ModeIrregular:
		return uint32(mode.Perm()) // | 0
	}
}

func FileModeFromModeType(mode uint32) (o fs.FileMode) {
	switch {
	case mode&unix.S_IFREG == unix.S_IFREG:
		return 0
	case mode&unix.S_IFDIR == unix.S_IFDIR:
		return fs.ModeDir
	case mode&unix.S_IFLNK == unix.S_IFLNK:
		return fs.ModeSymlink
	case mode&unix.S_IFIFO == unix.S_IFIFO:
		return fs.ModeNamedPipe
	case mode&unix.S_IFSOCK == unix.S_IFSOCK:
		return fs.ModeSocket
	case mode&unix.S_IFCHR == unix.S_IFCHR:
		return fs.ModeDevice | fs.ModeCharDevice
	case mode&unix.S_IFBLK == unix.S_IFBLK:
		return fs.ModeDevice
	default:
		return fs.ModeIrregular
	}
}
