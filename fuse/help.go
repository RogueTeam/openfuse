//go:build openbsd

package fuse

import (
	"bytes"
	"fmt"
	"unsafe"

	"golang.org/x/sys/unix"
)

func (f *Fuse[T]) ReplyErrno(hdr *FuseInHeader, errno unix.Errno) (err error) {
	outHeader := FuseOutHeader{
		Length: uint32(FuseOutHeaderSize),
		Error:  uint32(-errno),
		Unique: hdr.Unique,
	}

	_, err = unix.Write(f.Fd, unsafe.Slice((*byte)(unsafe.Pointer(&outHeader)), FuseOutHeaderSize))
	if err != nil {
		return fmt.Errorf("failed to write reply: %w", err)
	}
	return nil
}

func (f *Fuse[T]) ReplyValue[O FuseAttrOut | FuseInitOut |
	FuseOpenOut | FuseEntryOut |
	FuseWriteOut | FuseStatfsOut](hdr *FuseInHeader, out *O) (err error) {
	outHeader := FuseOutHeader{
		Length: uint32(FuseOutHeaderSize) + uint32(unsafe.Sizeof(*out)),
		Error:  0,
		Unique: hdr.Unique,
	}

	parts := [][]byte{
		unsafe.Slice((*byte)(unsafe.Pointer(&outHeader)), FuseOutHeaderSize),
		unsafe.Slice((*byte)(unsafe.Pointer(out)), unsafe.Sizeof(*out)),
	}
	_, err = unix.Writev(f.Fd, parts)
	if err != nil {
		return fmt.Errorf("failed to write reply: %w", err)
	}
	return nil
}

func (f *Fuse[T]) ReplyNoValue(hdr *FuseInHeader) (err error) {
	outHeader := FuseOutHeader{
		Length: uint32(FuseOutHeaderSize),
		Error:  0,
		Unique: hdr.Unique,
	}

	_, err = unix.Write(f.Fd, unsafe.Slice((*byte)(unsafe.Pointer(&outHeader)), FuseOutHeaderSize))
	if err != nil {
		return fmt.Errorf("failed to write reply: %w", err)
	}
	return nil
}

func (f *Fuse[T]) ReplyBuf(hdr *FuseInHeader, b []byte) (err error) {
	outHeader := FuseOutHeader{
		Length: uint32(FuseOutHeaderSize) + uint32(len(b)),
		Error:  0,
		Unique: hdr.Unique,
	}

	parts := [][]byte{
		unsafe.Slice((*byte)(unsafe.Pointer(&outHeader)), FuseOutHeaderSize),
		b,
	}
	_, err = unix.Writev(f.Fd, parts)
	if err != nil {
		return fmt.Errorf("failed to write reply: %w", err)
	}
	return nil
}

// Converts a byte slice to precisely match the type
func ValueFromBytes[T any](b []byte) (v *T, hErr *HError) {
	if len(b) != int(unsafe.Sizeof(*new(T))) {
		return nil, ErrorInvalidValue
	}

	return (*T)(unsafe.Pointer(&b[0])), nil
}

// Retrieves the value and the remaining data after it
func ValueAndDataFromBytes[T any](b []byte) (v *T, data []byte, hErr *HError) {
	sizeT := int(unsafe.Sizeof(*new(T)))
	if len(b) < sizeT {
		return nil, nil, ErrorInvalidValue
	}

	return (*T)(unsafe.Pointer(&b[0])), b[sizeT:], nil
}

// Retrieves the value and the remaining string after it
func ValueAndStringFromBytes[T any](b []byte) (v *T, s string, hErr *HError) {
	sizeT := int(unsafe.Sizeof(*new(T)))
	if len(b) < sizeT {
		return nil, "", ErrorInvalidValue
	}

	s, hErr = StringFromBytes(b[sizeT:])
	if hErr != nil {
		return nil, "", hErr
	}

	return (*T)(unsafe.Pointer(&b[0])), s, nil
}

// Returns a string from the bytes by forcing the string be null terminated and use the entire buffer
func StringFromBytes(b []byte) (s string, hErr *HError) {
	if len(b) < 2 {
		return "", ErrorInvalidValue
	}
	idx := bytes.IndexByte(b, 0x00)
	if idx != len(b)-1 {
		return "", ErrorInvalidValue
	}

	return string(b[:idx]), nil
}

// Same as StringFromBytes but doesn't error when the string is not the full buffer
func LazyStringFromBytes(b []byte) (s string, remaining []byte, hErr *HError) {
	if len(b) < 2 {
		return "", nil, ErrorInvalidValue
	}
	idx := bytes.IndexByte(b, 0x00)
	if idx < 1 {
		return "", nil, ErrorInvalidValue
	}

	return string(b[:idx]), b[1+idx:], nil
}
