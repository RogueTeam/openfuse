//go:build openbsd

package fuse

import (
	"context"
	"fmt"
	"log/slog"
	"unsafe"

	"golang.org/x/sys/unix"
)

var (
	ErrCritical  = fmt.Errorf("critical error")
	ErrDestroyed = fmt.Errorf("instance destroyed by the kernel")
)

type Handler interface {
	// Retrieves the attributes for a specific file identified by the inode in the header
	GetAttr(ctx context.Context, logger *slog.Logger, header *FuseInHeader, in *FuseGetattrIn) (out FuseAttrOut, hErr *HError)
	// Permits the handle to perform any cleanup prior releasing it entirely
	Destroy(ctx context.Context, logger *slog.Logger, header *FuseInHeader) (hErr *HError)
	// Opens a directory identified by the inode in the header
	OpenDir(ctx context.Context, logger *slog.Logger, header *FuseInHeader, in *FuseOpenIn) (out FuseOpenOut, hErr *HError)
	// Handle readdir
	Readdir(ctx context.Context, logger *slog.Logger, header *FuseInHeader, in *FuseReadIn) (out FuseDirEntries, hErr *HError)
	// Handle lookup
	Lookup(ctx context.Context, logger *slog.Logger, header *FuseInHeader, name string) (out FuseEntryOut, hErr *HError)
	// Releases a directory handle
	ReleaseDir(ctx context.Context, logger *slog.Logger, header *FuseInHeader, in *FuseReleaseIn) (hErr *HError)
	// Releases a inode entry
	Forget(ctx context.Context, logger *slog.Logger, header *FuseInHeader, in *FuseForgetIn) (hErr *HError)
	// Open files
	Open(ctx context.Context, logger *slog.Logger, header *FuseInHeader, in *FuseOpenIn) (out FuseOpenOut, hErr *HError)
	// Releases a file handle
	Release(ctx context.Context, logger *slog.Logger, header *FuseInHeader, in *FuseReleaseIn) (hErr *HError)
	// Reads from a file handle
	Read(ctx context.Context, logger *slog.Logger, header *FuseInHeader, in *FuseReadIn) (out []byte, hErr *HError)
	// Creates a new directory
	Mkdir(ctx context.Context, logger *slog.Logger, header *FuseInHeader, in *FuseMkdirIn, name string) (hErr *HError)
	// Unlink is used to remove a file from the filesystem
	// It is different from hard removals since open handles can still exists
	Unlink(ctx context.Context, logger *slog.Logger, header *FuseInHeader, name string) (hErr *HError)
	// Permits writing to files
	Write(ctx context.Context, logger *slog.Logger, header *FuseInHeader, in *FuseWriteIn, buffer []byte) (out FuseWriteOut, hErr *HError)
	// Supports setattr
	Setattr(ctx context.Context, logger *slog.Logger, header *FuseInHeader, in *FuseSetattrIn) (out FuseAttrOut, hErr *HError)
	// Handle file close
	Flush(ctx context.Context, logger *slog.Logger, header *FuseInHeader, in *FuseFlushIn) (hErr *HError)
	// Syncs a file with the underlying storage
	Fsync(ctx context.Context, logger *slog.Logger, header *FuseInHeader, in *FuseFsyncIn) (hErr *HError)
	// Removes a directory from the filesystem
	Rmdir(ctx context.Context, logger *slog.Logger, header *FuseInHeader, name string) (hErr *HError)
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
		logger.
			With("expected-size", header.Length).
			With("received-size", len(b)).
			ErrorContext(ctx, "Expecting the buffer size being the same as the reported one in the header")
		return fmt.Errorf("expecting the header size match the size of the received buffer: got(%d) != available(%d)", header.Length, len(b))
	}

	// Raw input buffer
	inBuffer := b[FuseInHeaderSize:]

	logger = logger.
		WithGroup(header.Opcode.String()).
		With("remaining-size", len(inBuffer))

	logger.DebugContext(ctx, "Handling")

	switch header.Opcode {
	default:
		return fmt.Errorf("unknown opcode: %d", header.Opcode)
	case FusebufTypeLOOKUP:
		name, hErr := StringFromBytes(inBuffer)
		if hErr != nil {
			logger.
				With("error", hErr.Error()).
				ErrorContext(ctx, "Invalid name")
			return f.ReplyErrno(header, hErr.Errno)
		}

		out, hErr := f.Handler.Lookup(ctx, logger, header, name)
		if hErr != nil {
			logger.
				With("error", hErr.Error()).
				ErrorContext(ctx, "Handler Error")
			return f.ReplyErrno(header, hErr.getErrno())
		}

		err := f.ReplyValue(header, &out)
		if err != nil {
			logger.With("error-msg", err).ErrorContext(ctx, "Failed to reply")
			return fmt.Errorf("failed to send reply: %w", err)
		}
		return nil
	case FusebufTypeGETATTR:
		inGetattr, hErr := ValueFromBytes[FuseGetattrIn](inBuffer)
		if hErr != nil {
			logger.With("error-msg", hErr.Error()).ErrorContext(ctx, "failed to get value from buffer")
			return f.ReplyErrno(header, hErr.getErrno())
		}

		out, hErr := f.Handler.GetAttr(ctx, logger, header, inGetattr)
		if hErr != nil {
			logger.
				With("error", hErr.Error()).
				ErrorContext(ctx, "Handler Error")
			return f.ReplyErrno(header, hErr.getErrno())
		}

		err := f.ReplyValue(header, &out)
		if err != nil {
			logger.With("error-msg", err).ErrorContext(ctx, "Failed to reply")
			return fmt.Errorf("failed to send reply: %w", err)
		}
		return nil
	case FusebufTypeSETATTR:
		inSetattr, hErr := ValueFromBytes[FuseSetattrIn](inBuffer)
		if hErr != nil {
			logger.With("error-msg", hErr.Error()).ErrorContext(ctx, "failed to get value from buffer")
			return f.ReplyErrno(header, hErr.getErrno())
		}

		out, hErr := f.Handler.Setattr(ctx, logger, header, inSetattr)
		if hErr != nil {
			logger.
				With("error", hErr.Error()).
				ErrorContext(ctx, "Handler Error")
			return f.ReplyErrno(header, hErr.getErrno())
		}

		err := f.ReplyValue(header, &out)
		if err != nil {
			logger.With("error-msg", err).ErrorContext(ctx, "Failed to reply")
			return fmt.Errorf("failed to send reply: %w", err)
		}
		return nil
	case /* TODO: Implement me */ FusebufTypeREADLINK:
	case /* TODO: Implement me */ FusebufTypeSYMLINK:
	case /* TODO: Implement me */ FusebufTypeMKNOD:
	case FusebufTypeMKDIR:
		inMkdir, name, hErr := ValueAndStringFromBytes[FuseMkdirIn](inBuffer)
		if hErr != nil {
			logger.With("error-msg", hErr.Error()).ErrorContext(ctx, "failed to get value from buffer")
			return f.ReplyErrno(header, hErr.getErrno())
		}

		hErr = f.Handler.Mkdir(ctx, logger, header, inMkdir, name)
		if hErr != nil {
			logger.
				With("error", hErr.Error()).
				ErrorContext(ctx, "Handler Error")
			return f.ReplyErrno(header, hErr.getErrno())
		}

		err := f.ReplyNoValue(header)
		if err != nil {
			logger.With("error-msg", err).ErrorContext(ctx, "Failed to reply")
			return fmt.Errorf("failed to send reply: %w", err)
		}
		return nil
	case FusebufTypeUNLINK:
		name, hErr := StringFromBytes(inBuffer)
		if hErr != nil {
			logger.
				With("error", hErr.Error()).
				ErrorContext(ctx, "Invalid name")
			return f.ReplyErrno(header, hErr.Errno)
		}

		hErr = f.Handler.Unlink(ctx, logger, header, name)
		if hErr != nil {
			logger.
				With("error", hErr.Error()).
				ErrorContext(ctx, "Handler Error")
			return f.ReplyErrno(header, hErr.getErrno())
		}

		err := f.ReplyNoValue(header)
		if err != nil {
			logger.With("error-msg", err).ErrorContext(ctx, "Failed to reply")
			return fmt.Errorf("failed to send reply: %w", err)
		}
		return nil
	case FusebufTypeRMDIR:
		name, hErr := StringFromBytes(inBuffer)
		if hErr != nil {
			logger.
				With("error", hErr.Error()).
				ErrorContext(ctx, "Invalid name")
			return f.ReplyErrno(header, hErr.Errno)
		}

		hErr = f.Handler.Rmdir(ctx, logger, header, name)
		if hErr != nil {
			logger.
				With("error", hErr.Error()).
				ErrorContext(ctx, "Handler Error")
			return f.ReplyErrno(header, hErr.getErrno())
		}

		err := f.ReplyNoValue(header)
		if err != nil {
			logger.With("error-msg", err).ErrorContext(ctx, "Failed to reply")
			return fmt.Errorf("failed to send reply: %w", err)
		}
		return nil
	case /* TODO: Implement me */ FusebufTypeRENAME:
	case /* TODO: Implement me */ FusebufTypeLINK:
	case FusebufTypeOPEN:
		inOpen, hErr := ValueFromBytes[FuseOpenIn](inBuffer)
		if hErr != nil {
			logger.With("error-msg", hErr.Error()).ErrorContext(ctx, "failed to get value from buffer")
			return f.ReplyErrno(header, hErr.getErrno())
		}

		out, hErr := f.Handler.Open(ctx, logger, header, inOpen)
		if hErr != nil {
			logger.
				With("error", hErr.Error()).
				ErrorContext(ctx, "Handler Error")
			return f.ReplyErrno(header, hErr.getErrno())
		}

		err := f.ReplyValue(header, &out)
		if err != nil {
			logger.With("error-msg", err).ErrorContext(ctx, "Failed to reply")
			return fmt.Errorf("failed to send reply: %w", err)
		}
		return nil
	case FusebufTypeREAD:
		inRead, hErr := ValueFromBytes[FuseReadIn](inBuffer)
		if hErr != nil {
			logger.With("error-msg", hErr.Error()).ErrorContext(ctx, "failed to get value from buffer")
			return f.ReplyErrno(header, hErr.getErrno())
		}

		out, hErr := f.Handler.Read(ctx, logger, header, inRead)
		if hErr != nil {
			logger.
				With("error", hErr.Error()).
				ErrorContext(ctx, "Handler Error")
			return f.ReplyErrno(header, hErr.getErrno())
		}

		err := f.ReplyBuf(header, out)
		if err != nil {
			logger.With("error-msg", err).ErrorContext(ctx, "Failed to reply")
			return fmt.Errorf("failed to send reply: %w", err)
		}
		return nil
	case FusebufTypeWRITE:
		inWrite, data, hErr := ValueAndDataFromBytes[FuseWriteIn](inBuffer)
		if hErr != nil {
			logger.With("error-msg", hErr.Error()).ErrorContext(ctx, "failed to get value from buffer")
			return f.ReplyErrno(header, hErr.getErrno())
		}

		out, hErr := f.Handler.Write(ctx, logger, header, inWrite, data)
		if hErr != nil {
			logger.
				With("error", hErr.Error()).
				ErrorContext(ctx, "Handler Error")
			return f.ReplyErrno(header, hErr.getErrno())
		}

		err := f.ReplyValue(header, &out)
		if err != nil {
			logger.With("error-msg", err).ErrorContext(ctx, "Failed to reply")
			return fmt.Errorf("failed to send reply: %w", err)
		}
		return nil
	case /* TODO: Implement me */ FusebufTypeSTATFS:
	case FusebufTypeRELEASE:
		inRelease, hErr := ValueFromBytes[FuseReleaseIn](inBuffer)
		if hErr != nil {
			logger.With("error-msg", hErr.Error()).ErrorContext(ctx, "failed to get value from buffer")
			return f.ReplyErrno(header, hErr.getErrno())
		}

		hErr = f.Handler.Release(ctx, logger, header, inRelease)
		if hErr != nil {
			logger.
				With("error", hErr.Error()).
				ErrorContext(ctx, "Handler Error")
			return f.ReplyErrno(header, hErr.getErrno())
		}

		err := f.ReplyNoValue(header)
		if err != nil {
			logger.With("error-msg", err).ErrorContext(ctx, "Failed to reply")
			return fmt.Errorf("failed to send reply: %w", err)
		}
		return nil
	case FusebufTypeFSYNC:
		inFsync, hErr := ValueFromBytes[FuseFsyncIn](inBuffer)
		if hErr != nil {
			logger.With("error-msg", hErr.Error()).ErrorContext(ctx, "failed to get value from buffer")
			return f.ReplyErrno(header, hErr.getErrno())
		}

		hErr = f.Handler.Fsync(ctx, logger, header, inFsync)
		if hErr != nil {
			logger.
				With("error", hErr.Error()).
				ErrorContext(ctx, "Handler Error")
			return f.ReplyErrno(header, hErr.getErrno())
		}

		err := f.ReplyNoValue(header)
		if err != nil {
			logger.With("error-msg", err).ErrorContext(ctx, "Failed to reply")
			return fmt.Errorf("failed to send reply: %w", err)
		}
		return nil
	case FusebufTypeFLUSH:
		inFuse, hErr := ValueFromBytes[FuseFlushIn](inBuffer)
		if hErr != nil {
			logger.With("error-msg", hErr.Error()).ErrorContext(ctx, "failed to get value from buffer")
			return f.ReplyErrno(header, hErr.getErrno())
		}

		hErr = f.Handler.Flush(ctx, logger, header, inFuse)
		if hErr != nil {
			logger.
				With("error", hErr.Error()).
				ErrorContext(ctx, "Handler Error")
			return f.ReplyErrno(header, hErr.getErrno())
		}

		err := f.ReplyNoValue(header)
		if err != nil {
			logger.With("error-msg", err).ErrorContext(ctx, "Failed to reply")
			return fmt.Errorf("failed to send reply: %w", err)
		}
		return nil
	case FusebufTypeINIT:
		inInit, hErr := ValueFromBytes[FuseInitIn](inBuffer)
		if hErr != nil {
			logger.With("error-msg", hErr.Error()).ErrorContext(ctx, "failed to get value from buffer")
			return f.ReplyErrno(header, hErr.getErrno())
		}

		if inInit.Major != FuseKernelVersion {
			logger.
				With("expecting", FuseKernelVersion).
				With("received", inInit.Major).
				ErrorContext(ctx, "fuse kernel major version mismatch")
			return f.ReplyErrno(header, unix.EINVAL)
		}

		err = f.ReplyValue(header, &FuseInitOut{
			Major:    FuseKernelVersion,
			Minor:    FuseKernelMinorVersion,
			MaxWrite: uint32(f.MaxWrite),
		})
		if err != nil {
			logger.With("error-msg", err).ErrorContext(ctx, "Failed to reply")
			return fmt.Errorf("failed to send reply: %w", err)
		}
		return nil
	case FusebufTypeOPENDIR:
		inOpen, hErr := ValueFromBytes[FuseOpenIn](inBuffer)
		if hErr != nil {
			logger.With("error-msg", hErr.Error()).ErrorContext(ctx, "failed to get value from buffer")
			return f.ReplyErrno(header, hErr.getErrno())
		}

		out, hErr := f.Handler.OpenDir(ctx, logger, header, inOpen)
		if hErr != nil {
			logger.
				With("error", hErr.Error()).
				ErrorContext(ctx, "Handler Error")
			return f.ReplyErrno(header, hErr.getErrno())
		}

		err := f.ReplyValue(header, &out)
		if err != nil {
			logger.With("error-msg", err).ErrorContext(ctx, "Failed to reply")
			return fmt.Errorf("failed to send reply: %w", err)
		}
		return nil
	case FusebufTypeREADDIR:
		inRead, hErr := ValueFromBytes[FuseReadIn](inBuffer)
		if hErr != nil {
			logger.With("error-msg", hErr.Error()).ErrorContext(ctx, "failed to get value from buffer")
			return f.ReplyErrno(header, hErr.getErrno())
		}

		out, hErr := f.Handler.Readdir(ctx, logger, header, inRead)
		if hErr != nil {
			logger.
				With("error", hErr.Error()).
				ErrorContext(ctx, "Handler Error")
			return f.ReplyErrno(header, hErr.getErrno())
		}

		err := f.ReplyBuf(header, out.Buffer)
		if err != nil {
			logger.With("error-msg", err).ErrorContext(ctx, "Failed to reply")
			return fmt.Errorf("failed to send reply: %w", err)
		}
		return nil
	case FusebufTypeRELEASEDIR:
		inRelease, hErr := ValueFromBytes[FuseReleaseIn](inBuffer)
		if hErr != nil {
			logger.With("error-msg", hErr.Error()).ErrorContext(ctx, "failed to get value from buffer")
			return f.ReplyErrno(header, hErr.getErrno())
		}

		hErr = f.Handler.ReleaseDir(ctx, logger, header, inRelease)
		if hErr != nil {
			logger.
				With("error", hErr.Error()).
				ErrorContext(ctx, "Handler Error")
			return f.ReplyErrno(header, hErr.getErrno())
		}

		err := f.ReplyNoValue(header)
		if err != nil {
			logger.With("error-msg", err).ErrorContext(ctx, "Failed to reply")
			return fmt.Errorf("failed to send reply: %w", err)
		}
		return nil
	case FusebufTypeDESTROY:
		if len(inBuffer) != 0 {
			logger.
				With("expecting", 0).
				With("received", len(inBuffer)).
				ErrorContext(ctx, "expecting no payload")
			return f.ReplyErrno(header, unix.EINVAL)
		}

		hErr := f.Handler.Destroy(ctx, logger, header)
		if hErr != nil {
			logger.
				With("error", hErr.Error()).
				ErrorContext(ctx, "Handler Error")
			return f.ReplyErrno(header, hErr.getErrno())
		}

		err := f.ReplyNoValue(header)
		if err != nil {
			logger.With("error-msg", err).ErrorContext(ctx, "Failed to reply")
			return fmt.Errorf("failed to send reply: %w", err)
		}
		return ErrDestroyed
	case FusebufTypeFORGET:
		inForget, hErr := ValueFromBytes[FuseForgetIn](inBuffer)
		if hErr != nil {
			logger.With("error-msg", hErr.Error()).ErrorContext(ctx, "failed to get value from buffer")
			return f.ReplyErrno(header, hErr.getErrno())
		}

		hErr = f.Handler.Forget(ctx, logger, header, inForget)
		if hErr != nil {
			logger.
				With("error", hErr.Error()).
				ErrorContext(ctx, "Handler Error")
			return f.ReplyErrno(header, hErr.getErrno())
		}

		err := f.ReplyNoValue(header)
		if err != nil {
			logger.With("error-msg", err).ErrorContext(ctx, "Failed to reply")
			return fmt.Errorf("failed to send reply: %w", err)
		}
		return nil
	}

	logger.ErrorContext(ctx, "Not implemented")
	err = f.ReplyErrno(header, unix.ENOTSUP)
	if err != nil {
		logger.With("error-msg", err).ErrorContext(ctx, "Failed to reply")
		return fmt.Errorf("failed to reply with error: %w", err)
	}
	return nil
}
