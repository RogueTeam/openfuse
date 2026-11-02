//go:build openbsd

package fuse

import (
	"unsafe"

	"golang.org/x/sys/unix"
)

const S_IFModeType = unix.S_IFBLK |
	unix.S_IFCHR |
	unix.S_IFDIR |
	unix.S_IFIFO |
	unix.S_IFLNK |
	unix.S_IFMT |
	unix.S_IFREG |
	unix.S_IFSOCK

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

type FuseDirentHeader struct {
	Ino     uint64
	Off     uint64
	Namelen uint32
	Type    uint32
}

const FuseDirentHeaderSize = unsafe.Sizeof(FuseDirentHeader{})

type FuseDirent struct {
	Header FuseDirentHeader
	Name   []byte
}

func (e *FuseDirent) Size() (size int) {
	return int(FuseDirentHeaderSize) + len(e.Name)
}

// Helper type for constructing the FuseDirent list
type FuseDirEntries struct {
	Buffer []byte
}

func (e *FuseDirEntries) Len() (length int) {
	return len(e.Buffer)
}

func (e *FuseDirEntries) Append(entry *FuseDirent) {
	rawSize := int(FuseDirentHeaderSize) + len(entry.Name)
	length := (rawSize + 7 /* Sizeof(uint64) - 1 */) & ^7 /* Sizeof(uint64) - 1 */
	padding := length - rawSize

	if cap(e.Buffer) < length {
		newBuffer := make([]byte, len(e.Buffer), len(e.Buffer)+length)
		copy(newBuffer, e.Buffer)
		e.Buffer = newBuffer
	}

	headerBuf := unsafe.Slice((*byte)(unsafe.Pointer(&entry.Header)), FuseDirentHeaderSize)
	e.Buffer = append(e.Buffer, headerBuf...)
	e.Buffer = append(e.Buffer, entry.Name...)
	if padding > 0 {
		e.Buffer = append(e.Buffer, make([]byte, padding)...)
	}
}

const FuseDirentSize = unsafe.Sizeof(FuseDirent{})
