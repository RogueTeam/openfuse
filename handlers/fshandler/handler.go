//go:build openbsd

package fshandler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"strings"

	"github.com/RogueTeam/gopenfuse/fuse"
	"github.com/RogueTeam/gopenfuse/handlers/tables"
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

type Handler struct {
	fs          fs.FS
	fileHandles tables.FileHandlesTable
	nodes       tables.InodeTable
}

var _ fuse.Handler = (*Handler)(nil)

func (h *Handler) GetAttr(ctx context.Context, logger *slog.Logger, header *fuse.FuseInHeader, in *fuse.FuseGetattrIn) (out fuse.FuseAttrOut, hErr *fuse.HError) {
	inode := h.nodes.GetByIno(header.NodeId)
	if inode == nil {
		return out, fuse.ErrorNotFound
	}

	statFS, ok := h.fs.(fs.StatFS)

	var info fs.FileInfo
	if ok {
		var err error
		info, err = statFS.Stat(inode.Path)
		if err != nil {
			return out, HErrorFromGoError(err)
		}
	} else {
		file, err := h.fs.Open(inode.Path)
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
		Attr: tables.FuseAttrFromFileInfo(header, info),
	}
	return out, nil
}

// Is not our responsability to destroying anything
func (h *Handler) Destroy(ctx context.Context, logger *slog.Logger, header *fuse.FuseInHeader) (hErr *fuse.HError) {
	return nil
}

func (h *Handler) OpenDir(ctx context.Context, logger *slog.Logger, header *fuse.FuseInHeader, _ *fuse.FuseOpenIn) (out fuse.FuseOpenOut, hErr *fuse.HError) {
	inode := h.nodes.GetByIno(header.NodeId)
	if inode == nil {
		return out, fuse.ErrorNotFound
	}

	// TODO: This prototype should also handle the flags, but how?
	file, err := h.fs.Open(inode.Path)
	if err != nil {
		// TODO: We just failed, should we remove the entry from the inodes list?
		return out, HErrorFromGoError(err)
	}

	info, err := file.Stat()
	if err != nil {
		return out, HErrorFromGoError(err)
	}

	if !info.IsDir() {
		file.Close()
		return out, &fuse.HError{
			Errno:   unix.ENOTDIR,
			GoError: fmt.Errorf("is not a directory"),
		}
	}

	fh := h.fileHandles.New(inode, file)

	out = fuse.FuseOpenOut{
		FileHandle: fh.FileDescriptor,
		OpenFlags:  0,
		Padding:    0,
	}
	return out, nil
}

func (h *Handler) Readdir(ctx context.Context, logger *slog.Logger, header *fuse.FuseInHeader, in *fuse.FuseReadIn) (out fuse.FuseDirEntries, hErr *fuse.HError) {
	fh := h.fileHandles.GetByFd(in.FileHandle)

	rdFile, ok := fh.Handle.(fs.ReadDirFile)
	if !ok {
		return out, &fuse.HError{
			Errno:   unix.ENOTDIR,
			GoError: fmt.Errorf("handle is not a directory"),
		}
	}

	offset := in.Offset

insertCache:
	for len(fh.ReaddirCache) > 0 {
		entry := fh.ReaddirCache[0]
		fmt.Println(entry.Name())

		info, err := entry.Info()
		if err != nil {
			return out, HErrorFromGoError(err)
		}

		entryName := strings.Trim(entry.Name(), "/")
		entryPath := path.Join(fh.Inode.Path, entryName)
		dirent := fuse.FuseDirent{
			Header: fuse.FuseDirentHeader{
				Ino:     h.nodes.Get(entryPath, false, fh.Inode).NodeId,
				Off:     offset,
				Namelen: uint32(len(entryName)),
				Type:    tables.ModeTypeFromFileMode(info.Mode()),
			},
			Name: []byte(entryName),
		}

		if out.Len()+dirent.Size() > int(in.Size) {
			return out, nil
		}

		out.Append(&dirent)
		fh.ReaddirCache = fh.ReaddirCache[1:]
		offset++
	}

	entries, err := rdFile.ReadDir(64)
	switch {
	case err == nil:
		fh.ReaddirCache = entries
		goto insertCache
	case errors.Is(err, io.EOF):
		fmt.Println(fh.Inode.Path, out.Len())
		return out, nil
	default:
		return out, HErrorFromGoError(err)
	}
}

func (h *Handler) Lookup(ctx context.Context, logger *slog.Logger, header *fuse.FuseInHeader, name string) (out fuse.FuseEntryOut, hErr *fuse.HError) {
	inode := h.nodes.GetByIno(header.NodeId)
	if inode == nil {
		return out, fuse.ErrorNotFound
	}

	sub, err := fs.Sub(h.fs, inode.Path)
	if err != nil {
		return out, HErrorFromGoError(err)
	}

	file, err := sub.Open(name)
	if err != nil {
		return out, HErrorFromGoError(err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return out, HErrorFromGoError(err)
	}

	targetInode := h.nodes.Get(path.Join(inode.Path, name), false, inode)

	out = fuse.FuseEntryOut{
		NodeId: targetInode.NodeId,
		Attr:   tables.FuseAttrFromFileInfo(header, info),
	}

	return out, nil
}

func (h *Handler) ReleaseDir(ctx context.Context, logger *slog.Logger, header *fuse.FuseInHeader, in *fuse.FuseReleaseIn) (hErr *fuse.HError) {
	inode := h.nodes.GetByIno(header.NodeId)
	if inode == nil {
		return fuse.ErrorNotFound
	}

	fh := h.fileHandles.GetByFd(in.FileHandle)
	if fh == nil {
		return fuse.ErrorNotFound
	}

	if fh.Inode != inode {
		return fuse.ErrorPermissionDenied
	}

	err := fh.Handle.Close()
	if err != nil {
		logger.With("error-msg", err).WarnContext(ctx, "Failed to close directory handle")
	}
	return h.fileHandles.Del(fh.FileDescriptor)
}

func (h *Handler) Forget(ctx context.Context, logger *slog.Logger, header *fuse.FuseInHeader, in *fuse.FuseForgetIn) (hErr *fuse.HError) {
	inode := h.nodes.GetByIno(header.NodeId)
	if inode == nil {
		return fuse.ErrorNotFound
	}

	return h.nodes.Del(inode.NodeId)
}

// Provided filesystem may implement any of the following subtypes in order to support more operations
// When fs.StatFS is implemented any stat operation may be performed directly with the available path without first calling Open
// When fshandler.MkdirFS is implemented, directory creation is permitted
// When fshandler.MknodFS is implemented, custom node creation is permitted
func New(f fs.FS) (h *Handler) {
	h = &Handler{
		fs: f,
	}
	h.nodes.Get(".", true, nil)
	return h
}
