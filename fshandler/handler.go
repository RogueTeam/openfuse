//go:build openbsd

package fshandler

import (
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"os"
	"sync/atomic"

	"github.com/RogueTeam/gopenfuse/fuse"
	"golang.org/x/sys/unix"
)

func HErrorFromGoError(err error) (hErr *fuse.HError) {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, os.ErrNotExist) || os.IsNotExist(err):
		return fuse.ErrorNotFound
	default:
		return &fuse.HError{
			Errno:   unix.EPERM,
			GoError: err,
		}
	}
}

type filehandle struct {
	handle any
}

type inode struct {
	// Special flag to prevent nodes from being removed
	protected bool
	// path to pass to fs.FS
	path string
}

type Handler struct {
	fs                fs.FS
	nextHandle        atomic.Uint64
	nextInode         atomic.Uint64
	fileHandlesTables map[uint64]*filehandle
	inodesTable       map[uint64]*inode
}

var _ fuse.Handler = (*Handler)(nil)

func FuseAttrFromFileInfo(header *fuse.FuseInHeader, info fs.FileInfo) (out fuse.FuseAttr) {
	mode := uint32(info.Mode().Perm())
	switch info.Mode().Type() {
	case fs.ModeDir:
		mode |= unix.S_IFDIR
	case fs.ModeSymlink:
		mode |= unix.S_IFLNK
	case fs.ModeNamedPipe:
		mode |= unix.S_IFIFO
	case fs.ModeSocket:
		mode |= unix.S_IFSOCK
	case fs.ModeDevice:
		mode |= unix.S_IFBLK
	case fs.ModeCharDevice:
		mode |= unix.S_IFCHR
	case fs.ModeIrregular:
		mode |= unix.S_IFREG
	}
	return fuse.FuseAttr{
		Ino:   header.NodeId,
		Size:  uint64(info.Size()),
		Mode:  mode,
		Mtime: uint64(info.ModTime().UnixMilli()),
	}
}

func (h *Handler) GetAttr(ctx context.Context, logger *slog.Logger, header *fuse.FuseInHeader, in *fuse.FuseGetattrIn) (out fuse.FuseAttrOut, hErr *fuse.HError) {
	inode, found := h.inodesTable[header.NodeId]
	if !found {
		return out, fuse.ErrorNotFound
	}

	statFS, ok := h.fs.(fs.StatFS)

	var info fs.FileInfo
	if ok {
		var err error
		info, err = statFS.Stat(inode.path)
		if err != nil {
			return out, HErrorFromGoError(err)
		}
	} else {
		file, err := h.fs.Open(inode.path)
		if err != nil {
			return out, HErrorFromGoError(err)
		}
		defer file.Close()

		info, err = file.Stat()
		if err != nil {
			return out, HErrorFromGoError(err)
		}
	}

	out = fuse.FuseAttrOut{
		Attr: FuseAttrFromFileInfo(header, info),
	}
	return out, nil
}

// Is not our responsability to destroying anything
func (h *Handler) Destroy(ctx context.Context, logger *slog.Logger, header *fuse.FuseInHeader) (hErr *fuse.HError) {
	return nil
}

func (h *Handler) OpenDir(ctx context.Context, logger *slog.Logger, header *fuse.FuseInHeader, _ *fuse.FuseOpenIn) (out fuse.FuseOpenOut, hErr *fuse.HError) {
	inode, found := h.inodesTable[header.NodeId]
	if !found {
		return out, fuse.ErrorNotFound
	}

	// TODO: This prototype should also handle the flags, but how?
	fhObj, err := h.fs.Open(inode.path)
	if err != nil {
		// TODO: We just failed, should we remove the entry from the inodes list?
		return out, HErrorFromGoError(err)
	}

	fh := h.nextHandle.Add(1)

	h.fileHandlesTables[fh] = &filehandle{
		handle: fhObj,
	}

	out = fuse.FuseOpenOut{
		FileHandle: fh,
		OpenFlags:  0,
		Padding:    0,
	}
	return out, nil
}

// Provided filesystem may implement any of the following subtypes in order to support more operations
// fs.StatFS
func New(f fs.FS) (h *Handler) {
	h = &Handler{
		fs:                f,
		fileHandlesTables: make(map[uint64]*filehandle),
		inodesTable:       map[uint64]*inode{},
	}
	h.inodesTable[h.nextInode.Add(1)] = &inode{protected: true, path: "."}
	return h
}
