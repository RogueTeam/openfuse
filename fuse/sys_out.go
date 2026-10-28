//go:build openbsd

package fuse

import "unsafe"

type FuseOutHeader struct {
	Length uint32
	Error  uint32
	Unique uint64
}

const FuseOutHeaderSize = unsafe.Sizeof(FuseOutHeader{})

type FuseEntryOut struct {
	NodeId         uint64
	Generation     uint64 // Not implemented
	EntryValid     uint64 // Not implemented
	AttrValid      uint64 // Not implemented
	EntryValidNsec uint32 // Not implemented
	AttrValidNsec  uint32 // Not implemented
	Attr           FuseAttr
}

const FuseEntryOutSize = unsafe.Sizeof(FuseEntryOut{})

type FuseAttrOut struct {
	AttrValid     uint64 // Not implemented
	AttrValidNsec uint32 // Not implemented
	Dummy         uint32
	Attr          FuseAttr
}

const FuseAttrOutSize = unsafe.Sizeof(FuseAttrOut{})

type FuseOpenOut struct {
	FileHandle uint64
	OpenFlags  uint32
	Padding    uint32
}

const FuseOpenOutSize = unsafe.Sizeof(FuseOpenOut{})

type FuseWriteOut struct {
	Size    uint32
	Padding uint32
}

const FuseWriteOutSize = unsafe.Sizeof(FuseWriteOut{})

type FuseKstatfs struct {
	Blocks  uint64
	Bfree   uint64
	Bavail  uint64
	Files   uint64
	Ffree   uint64
	Bsize   uint32
	Namelen uint32
	Frsize  uint32
	Padding uint32
	Spare   [6]uint32
}

type FuseStatfsOut struct {
	Stat FuseKstatfs
}

const FuseStatfsOutSize = unsafe.Sizeof(FuseStatfsOut{})

type FuseInitOut struct {
	Major               uint32
	Minor               uint32
	MaxReadahead        uint32 // Not implemented
	Flags               uint32 // Not implemented
	MaxBackground       uint16 // Not implemented
	CongestionThreshold uint16 // Not implemented
	MaxWrite            uint32
}

const FuseInitOutSize = unsafe.Sizeof(FuseInitOut{})
