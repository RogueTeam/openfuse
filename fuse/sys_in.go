//go:build openbsd

package fuse

import "unsafe"

type FusebufType uint32

const (
	FusebufTypeLOOKUP     FusebufType = 1
	FusebufTypeGETATTR    FusebufType = 3
	FusebufTypeSETATTR    FusebufType = 4
	FusebufTypeREADLINK   FusebufType = 5
	FusebufTypeSYMLINK    FusebufType = 6
	FusebufTypeMKNOD      FusebufType = 8
	FusebufTypeMKDIR      FusebufType = 9
	FusebufTypeUNLINK     FusebufType = 10
	FusebufTypeRMDIR      FusebufType = 11
	FusebufTypeRENAME     FusebufType = 12
	FusebufTypeLINK       FusebufType = 13
	FusebufTypeOPEN       FusebufType = 14
	FusebufTypeREAD       FusebufType = 15
	FusebufTypeWRITE      FusebufType = 16
	FusebufTypeSTATFS     FusebufType = 17
	FusebufTypeRELEASE    FusebufType = 18
	FusebufTypeFSYNC      FusebufType = 20
	FusebufTypeFLUSH      FusebufType = 25
	FusebufTypeINIT       FusebufType = 26
	FusebufTypeOPENDIR    FusebufType = 27
	FusebufTypeREADDIR    FusebufType = 28
	FusebufTypeRELEASEDIR FusebufType = 29
	FusebufTypeDESTROY    FusebufType = 38
	FusebufTypeFORGET     FusebufType = 2
)

func (t FusebufType) String() (s string) {
	switch t {
	default:
		return "INVALID"
	case FusebufTypeLOOKUP:
		return "LOOKUP"
	case FusebufTypeGETATTR:
		return "GETATTR"
	case FusebufTypeSETATTR:
		return "SETATTR"
	case FusebufTypeREADLINK:
		return "READLINK"
	case FusebufTypeSYMLINK:
		return "SYMLINK"
	case FusebufTypeMKNOD:
		return "MKNOD"
	case FusebufTypeMKDIR:
		return "MKDIR"
	case FusebufTypeUNLINK:
		return "UNLINK"
	case FusebufTypeRMDIR:
		return "RMDIR"
	case FusebufTypeRENAME:
		return "RENAME"
	case FusebufTypeLINK:
		return "LINK"
	case FusebufTypeOPEN:
		return "OPEN"
	case FusebufTypeREAD:
		return "READ"
	case FusebufTypeWRITE:
		return "WRITE"
	case FusebufTypeSTATFS:
		return "STATFS"
	case FusebufTypeRELEASE:
		return "RELEASE"
	case FusebufTypeFSYNC:
		return "FSYNC"
	case FusebufTypeFLUSH:
		return "FLUSH"
	case FusebufTypeINIT:
		return "INIT"
	case FusebufTypeOPENDIR:
		return "OPENDIR"
	case FusebufTypeREADDIR:
		return "READDIR"
	case FusebufTypeRELEASEDIR:
		return "RELEASEDIR"
	case FusebufTypeDESTROY:
		return "DESTROY"
	case FusebufTypeFORGET:
		return "FORGET"
	}
}

type FuseInHeader struct {
	Length  uint32
	Opcode  FusebufType
	Unique  uint64
	NodeId  uint64
	Uid     uint32
	Gid     uint32
	Pid     uint32
	Padding uint32
}

const FuseInHeaderSize = unsafe.Sizeof(FuseInHeader{})

type FuseForgetIn struct {
	Nlookup uint64
}

type FuseGetattrIn struct {
	GetattrFlags uint32 // Not implemented
	Dummy        uint32
	FileHandle   uint64 // Not implemented
}

const FuseGetattrInSize = int(unsafe.Sizeof(FuseGetattrIn{}))

type FuseSetattrIn struct {
	Valid      uint32
	Padding    uint32
	FileHandle uint64
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

type FuseMknodIn struct {
	Mode    uint32
	Rdev    uint32
	Umask   uint32
	Padding uint32
}

type FuseMkdirIn struct {
	Mode uint32
	Mask uint32
}

type FuseRenameIn struct {
	NewDir uint64
}

type FuseLinkIn struct {
	OldNodeId uint64
}

type FuseOpenIn struct {
	Flags  uint32
	Unused uint32
}

const FuseOpenInSize = int(unsafe.Sizeof(FuseOpenIn{}))

type FuseReadIn struct {
	FileHandle uint64
	Offset     uint64
	Size       uint32
	ReadFlags  uint32
	LockOwner  uint64
	Flags      uint32
	Padding    uint32
}

type FuseWriteIn struct {
	FileHandle uint64
	Offset     uint64
	Size       uint32
	WriteFlags uint32
	LockOwner  uint64
	Flags      uint32
	Padding    uint32
}

type FuseReleaseIn struct {
	FileHandle   uint64
	Flags        uint32
	ReleaseFlags uint32
	LockOwner    uint64
}

type FuseFsyncIn struct {
	FileHandle uint64
	FsyncFlags uint32
	Padding    uint32
}

type FuseFlushIn struct {
	FileHandle uint64
	Unused     uint32
	Padding    uint32
	LockOwner  uint64
}

type FuseInitIn struct {
	Major        uint32
	Minor        uint32
	MaxReadahead uint32
	Flags        uint32
}

const FuseInitInSize = int(unsafe.Sizeof(FuseInitIn{}))

const FusebufInSize = max(
	unsafe.Sizeof(FuseForgetIn{}),
	unsafe.Sizeof(FuseGetattrIn{}),
	unsafe.Sizeof(FuseSetattrIn{}),
	unsafe.Sizeof(FuseMknodIn{}),
	unsafe.Sizeof(FuseMkdirIn{}),
	unsafe.Sizeof(FuseRenameIn{}),
	unsafe.Sizeof(FuseLinkIn{}),
	unsafe.Sizeof(FuseOpenIn{}),
	unsafe.Sizeof(FuseReadIn{}),
	unsafe.Sizeof(FuseWriteIn{}),
	unsafe.Sizeof(FuseReleaseIn{}),
	unsafe.Sizeof(FuseFsyncIn{}),
	unsafe.Sizeof(FuseFlushIn{}),
	unsafe.Sizeof(FuseInitIn{}),
)

type Fusebuf struct {
	Header FuseInHeader
	In     [FusebufInSize]byte
}

const FusebufSize = int(unsafe.Sizeof(Fusebuf{}))
