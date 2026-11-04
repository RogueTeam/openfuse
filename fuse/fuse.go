//go:build openbsd

package fuse

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"sync/atomic"
	"unsafe"

	"golang.org/x/sys/unix"
)

// Last time checked 8.0
const FusebufMaxSize = 4096 * 1024 // 4MB

const DevicePath = "/dev/fuse0"

// Fuse, never create by hand. Always use Mount
type Fuse[T Handler] struct {
	// Read-Only Handler instance
	Handler T
	// Read-Only used to monitor if the fuse is running or not
	Finished atomic.Bool
	// Read-Only
	MaxWrite int
	// READ-Only Mountpoint, used during close for unmount
	Mountpoint string
	// READ-Only  File descriptor to the underlying
	Fd int
}

// Serve incoming fuse requests until the provided context is terminated
// Serves each request sequentially so no parallelism is involved at all
func (f *Fuse[T]) Serve(ctx context.Context, logger *slog.Logger) (err error) {
	var buffer []byte
	if f.MaxWrite > FusebufMaxSize {
		buffer = make([]byte, FusebufSize+FusebufMaxSize)
	} else if f.MaxWrite < unix.PathMax+unix.NAME_MAX {
		buffer = make([]byte, FusebufSize+unix.PathMax+unix.NAME_MAX)
	} else {
		buffer = make([]byte, FusebufSize+f.MaxWrite)
	}

	for !f.Finished.Load() {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context error: %w", ctx.Err())
		default:
			n, err := unix.Read(f.Fd, buffer)
			if err != nil {
				return fmt.Errorf("failed to read from fuse device: %w", err)
			}

			err = f.handle(ctx, logger, buffer[:n])
			if err != nil {
				return fmt.Errorf("failed to handle: %w", err)
			}
		}
	}
	return nil
}

// Unmounts and Closes the file descriptor related to the instance
func (f *Fuse[T]) Close() (err error) {
	if !f.Finished.CompareAndSwap(false, true) {
		return fmt.Errorf("already closed")
	}

	err0 := unix.Close(f.Fd)
	err1 := unix.Unmount(f.Mountpoint, unix.MNT_FORCE)
	switch {
	case err0 == nil && err1 == nil:
		return nil
	case err0 != nil && err1 == nil:
		return fmt.Errorf("failed to close file descriptor: %w", err0)
	case err0 == nil && err1 != nil:
		return fmt.Errorf("failed to unmount: %w", err1)
	default:
		return fmt.Errorf("multiple errors: %w", errors.Join(err0, err1))
	}
}

// Options for the mount
type Options struct {
	// Name of the filesystem to mount
	// This is never used in the libfuse sample (last time checked 8.0)
	Name string
	// Allow other users to access this FS
	AllowOther bool
	// Default permission to apply to every file
	// TODO: Implement me!
	DefaultPermissions int
	// Negotiated max read size
	MaxRead int
	// Negotiated max write size
	MaxWrite int
	// Do not permit the kernel to enforce atime
	NoAtime bool
	// Permit mount on non empty directory
	// This is never used in the libfuse sample (last time checked 8.0)
	NonEmpty bool
	// The Filesystem is Read only
	ReadOnly bool
}

const MaxPendingEvents = 1024

// Apply all the defaults
func (o *Options) Defaults() {
	if o.MaxRead <= 0 {
		o.MaxRead = FusebufMaxSize
	}
	if o.MaxWrite <= 0 {
		o.MaxWrite = FusebufMaxSize
	}
}

// Validate all the inputs
func (o *Options) Validate() (err error) {
	if o.Name == "" {
		return fmt.Errorf("no name provided")
	}
	if o.MaxRead > FusebufMaxSize {
		return fmt.Errorf("o.MaxRead can't be bigger than %d; got %d", FusebufMaxSize, o.MaxRead)
	}
	if o.MaxWrite > FusebufMaxSize {
		return fmt.Errorf("o.MaxWrite can't be bigger than %d; got %d", FusebufMaxSize, o.MaxWrite)
	}
	return nil
}

// Creates a new FUSE instance
func Mount[T Handler](dir string, o Options, h T) (i *Fuse[T], err error) {
	if dir == "" {
		return nil, fmt.Errorf("no directory provided")
	}

	err = o.Validate()
	if err != nil {
		return nil, fmt.Errorf("failed to validate options: %w", err)
	}

	o.Defaults()

	dir, err = filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path for directory: %w", err)
	}

	fd, err := unix.Open(DevicePath, unix.O_RDWR|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to open fuse device: %w", err)
	}
	defer func() {
		if err == nil {
			return
		}

		unix.Close(fd)
	}()

	var mountFlags int
	if o.ReadOnly {
		mountFlags |= unix.MNT_RDONLY
	}
	if o.NoAtime {
		mountFlags |= unix.MNT_NOATIME
	}

	i = &Fuse[T]{
		Handler:    h,
		MaxWrite:   o.MaxWrite,
		Mountpoint: dir,
		Fd:         fd,
	}

	var fuseArgs FuseFsArgs
	fuseArgs.Name, err = unix.BytePtrFromString(o.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to set fuse argument for name: %w", err)
	}
	fuseArgs.Fd = int32(fd)
	fuseArgs.MaxRead = int32(o.MaxRead)
	if o.AllowOther {
		fuseArgs.AllowOther = 1
	}

	err = unix.Mount(
		unix.MOUNT_FUSEFS,
		dir,
		mountFlags,
		unsafe.Pointer(&fuseArgs),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to mount fuse: %w", err)
	}

	return i, nil
}
