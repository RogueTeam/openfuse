//go:build openbsd

package fuse

import (
	"fmt"

	"golang.org/x/sys/unix"
)

// HError wrapper to easily respond with errors from the handler
// Check for already constructed fuse.HError* values for predefined values
type HError struct {
	// when value is 0 errno fallbacks to unix.EINVAL
	Errno unix.Errno
	// Traditional go error used by the logger
	// When the value errors.Is(e.Error, ErrCritical) == true
	// the handle returns a wrapper error that fill force the serve function to stop
	GoError error
}

var _ error = (*HError)(nil)

func (e *HError) getErrno() (errno unix.Errno) {
	if e.Errno == 0 {
		return unix.EINVAL
	}
	return e.Errno
}

func (e *HError) Error() (s string) {
	errno := e.getErrno()
	if e.GoError == nil {
		return fmt.Sprintf("errno: %d: %s", errno, errno.Error())
	}
	return fmt.Sprintf("%s: errno %d: %s", e.GoError, errno, errno.Error())
}

func (e *HError) String() (s string) { return e.Error() }

var (
	ErrorNotSupported      = &HError{Errno: unix.ENOTSUP, GoError: fmt.Errorf("operation not supported")}
	ErrorNotFound          = &HError{Errno: unix.ENOENT, GoError: fmt.Errorf("not found")}
	ErrorInvalidValue      = &HError{Errno: unix.EINVAL, GoError: fmt.Errorf("invalid value")}
	ErrorPermissionDenied  = &HError{Errno: unix.EPERM, GoError: fmt.Errorf("permission denied")}
	ErrorIsDirectory       = &HError{Errno: unix.EISDIR, GoError: fmt.Errorf("is a directory")}
	ErrorNotDirectory      = &HError{Errno: unix.ENOTDIR, GoError: fmt.Errorf("is not a directory")}
	ErrorBadFileDescriptor = &HError{Errno: unix.EBADF, GoError: fmt.Errorf("bad file descriptor")}
)
