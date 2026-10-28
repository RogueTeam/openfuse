//go:build openbsd

package fuse

import (
	"errors"
	"fmt"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/unix"
)

// Last time checked 8.0
const FuseBufMaxSize = 4096 * 1024 // 4MB

const DevicePath = "/dev/fuse0"

// Instance, never create by hand. Always use Mount
type Instance struct {
	// Mountpoint, used during close for unmount
	Mountpoint string
	// File descriptor to the underlying
	Fd int
}

// Closes the file descriptor related to the instance
func (i *Instance) Close() (err error) {
	err1 := unix.Unmount(i.Mountpoint, unix.MNT_FORCE)
	err2 := unix.Close(i.Fd)
	switch {
	case err1 == nil && err2 == nil:
		return nil
	case err1 != nil && err2 == nil:
		return fmt.Errorf("failed to unmount: %w", err1)
	case err1 == nil && err2 != nil:
		return fmt.Errorf("failed to close file descriptor: %w", err2)
	default:
		return fmt.Errorf("multiple errors: %w", errors.Join(err1, err2))
	}
}

// Options for the mount
type Options struct {
	// Name of the filesystem to mount
	// This is never used in the libfuse sample (last time checked 8.0)
	Name string
	// Allow other users to access this FS
	AllowOther bool
	// TODO: What is this for?
	// This is never used in the libfuse sample (last time checked 8.0)
	DefPerms int
	// Negotiated max read size
	MaxRead int
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
		o.MaxRead = 32 * 1024
	}
}

// Validate all the inputs
func (o *Options) Validate() (err error) {
	if o.Name == "" {
		return fmt.Errorf("no options provided")
	}
	if o.MaxRead > FuseBufMaxSize {
		return fmt.Errorf("o.MaxRead can't be bigger than %d; got %d", FuseBufMaxSize, o.MaxRead)
	}
	return nil
}

// Creates a new FUSE instance
func Mount(dir string, o Options) (i *Instance, err error) {
	if dir == "" {
		return nil, fmt.Errorf("no directory provided")
	}

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

	i = &Instance{
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
