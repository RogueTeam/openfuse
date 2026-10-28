//go:build openbsd

package fuse

const (
	FuseKernelVersion      = 7
	FuseKernelMinorVersion = 19
)

// Mapped directly from struct fusefs_args
type FuseFsArgs struct {
	Name       *byte
	Fd         int32
	MaxRead    int32
	AllowOther int32
}

type FuseAttr struct {
	Ino       uint64
	Size      uint64
	Blocks    uint64
	Atime     uint64
	Mtime     uint64
	Ctime     uint64
	Atimesec  uint64
	Mtimesec  uint64
	Ctimesec  uint64
	Mode      uint32
	Nlink     uint32
	Uid, Gid  uint32
	Rdev      uint32
	Blocksize uint32
	Padding   uint32
}
