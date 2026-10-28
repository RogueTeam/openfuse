//go:build openbsd

package fuse

import (
	"context"
	"fmt"
	"log/slog"
	"unsafe"

	"golang.org/x/sys/unix"
)

type Handler interface{}

func (f *Fuse[T]) ReplyError(hdr *FuseInHeader, errno unix.Errno) (err error) {
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

// Handle function responsible of handling requests to then route them to the upper
func (f *Fuse[T]) handle(ctx context.Context, logger *slog.Logger, b []byte) (err error) {
	logger = logger.WithGroup("Handle buffer").With("blength", len(b))

	// Critical error - No way to receive other packets from the kernel
	if len(b) < int(FuseInHeaderSize) {
		logger.ErrorContext(ctx, "Expecting at least the size of the FuseInHeader")
		return fmt.Errorf("expecting at least the size of the header")
	}

	header := (*FuseInHeader)(unsafe.Pointer(&b[0]))
	if int(header.Length) != len(b) {
		return fmt.Errorf("expecting the header size match the size of the received buffer: got(%d) != available(%d)", header.Length, len(b))
	}

	inB := b[FuseInHeaderSize:]
	switch header.Opcode {
	default:
		return fmt.Errorf("unknown opcode: %d", header.Opcode)
	case FusebufTypeLOOKUP:
	case FusebufTypeGETATTR:
	case FusebufTypeSETATTR:
	case FusebufTypeREADLINK:
	case FusebufTypeSYMLINK:
	case FusebufTypeMKNOD:
	case FusebufTypeMKDIR:
	case FusebufTypeUNLINK:
	case FusebufTypeRMDIR:
	case FusebufTypeRENAME:
	case FusebufTypeLINK:
	case FusebufTypeOPEN:
	case FusebufTypeREAD:
	case FusebufTypeWRITE:
	case FusebufTypeSTATFS:
	case FusebufTypeRELEASE:
	case FusebufTypeFSYNC:
	case FusebufTypeFLUSH:
	case FusebufTypeINIT:
		logger = logger.WithGroup("Init")
		if len(inB) != FuseInitInSize {
			logger.
				With("expecting", FuseInitInSize).
				With("received", len(inB)).
				ErrorContext(ctx, "invalid payload size")
			return f.ReplyError(header, unix.EINVAL)
		}

		inInit := (*FuseInitIn)(unsafe.Pointer(&inB[0]))
		if inInit.Major != FuseKernelVersion {
			logger.
				With("expecting", FuseKernelVersion).
				With("received", inInit.Major).
				ErrorContext(ctx, "fuse kernel major version mismatch")
			return f.ReplyError(header, unix.EINVAL)
		}

		err = f.ReplyValue(header, &FuseInitOut{
			Major:    FuseKernelVersion,
			Minor:    FuseKernelMinorVersion,
			MaxWrite: uint32(f.MaxWrite),
		})
		if err != nil {
			return fmt.Errorf("failed to send reply: %w", err)
		}
		return nil
	case FusebufTypeOPENDIR:
	case FusebufTypeREADDIR:
	case FusebufTypeRELEASEDIR:
	case FusebufTypeDESTROY:
	case FusebufTypeFORGET:
	}

	return fmt.Errorf("implement me!")
}
