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

	"github.com/RogueTeam/openfuse/fuse"
	"github.com/RogueTeam/openfuse/handlers/tables"
	"golang.org/x/sys/unix"
)

func HErrorFromGoError(err error) (hErr *fuse.HError) {
	if err == nil {
		return nil
	}

	asHerr, ok := err.(*fuse.HError)
	if ok {
		return asHerr
	}

	switch {
	case errors.Is(err, fs.ErrClosed):
		return fuse.ErrorBadFileDescriptor
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

	var info fs.FileInfo
	if statFS, ok := h.fs.(fs.StatFS); ok {
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
	destroyFS, ok := h.fs.(DestroyFS)
	if ok {
		return HErrorFromGoError(destroyFS.Destroy())
	}

	for _, handle := range h.fileHandles.Handles {
		handle.Handle.Close()
	}

	return nil
}

func (h *Handler) OpenDir(ctx context.Context, logger *slog.Logger, header *fuse.FuseInHeader, in *fuse.FuseOpenIn) (out fuse.FuseOpenOut, hErr *fuse.HError) {
	inode := h.nodes.GetByIno(header.NodeId)
	if inode == nil {
		return out, fuse.ErrorNotFound
	}

	var err error
	var file fs.File
	switch f := h.fs.(type) {
	case OpenFlagsFS:
		file, err = f.OpenFlags(inode.Path, in.Flags)
	default:
		file, err = f.Open(inode.Path)
	}
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

		info, err := entry.Info()
		if err != nil {
			return out, HErrorFromGoError(err)
		}

		entryName := strings.Trim(entry.Name(), "/")
		entryPath := path.Join(fh.Inode.Path, entryName)
		dirent := fuse.FuseDirent{
			Header: fuse.FuseDirentHeader{
				Ino:     h.nodes.New(entryPath, false, false, fh.Inode).NodeId,
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

	targetInode := h.nodes.New(path.Join(inode.Path, name), true, false, inode)

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

	return h.nodes.ForgetByIno(inode.NodeId)
}

func (h *Handler) Open(ctx context.Context, logger *slog.Logger, header *fuse.FuseInHeader, in *fuse.FuseOpenIn) (out fuse.FuseOpenOut, hErr *fuse.HError) {
	inode := h.nodes.GetByIno(header.NodeId)
	if inode == nil {
		return out, fuse.ErrorNotFound
	}

	var err error
	var file fs.File
	switch f := h.fs.(type) {
	case OpenFlagsFS:
		file, err = f.OpenFlags(inode.Path, in.Flags)
	default:
		file, err = f.Open(inode.Path)
	}
	if err != nil {
		// TODO: We just failed, should we remove the entry from the inodes list?
		return out, HErrorFromGoError(err)
	}

	info, err := file.Stat()
	if err != nil {
		return out, HErrorFromGoError(err)
	}

	if info.IsDir() {
		file.Close()
		return out, &fuse.HError{
			Errno:   unix.EISDIR,
			GoError: fmt.Errorf("is a directory"),
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

func (h *Handler) Release(ctx context.Context, logger *slog.Logger, header *fuse.FuseInHeader, in *fuse.FuseReleaseIn) (hErr *fuse.HError) {
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

func (h *Handler) Read(ctx context.Context, logger *slog.Logger, header *fuse.FuseInHeader, in *fuse.FuseReadIn) (out []byte, hErr *fuse.HError) {
	fh := h.fileHandles.GetByFd(in.FileHandle)

useCache:
	// Read from buffer
	if fh.CacheBuffer != nil {
		if in.Offset >= uint64(len(fh.CacheBuffer)) {
			// EOF
			return nil, nil
		}

		out = fh.CacheBuffer[in.Offset:min(in.Offset+uint64(in.Size), uint64(len(fh.CacheBuffer)))]
		return out, nil
	}

	var err error
	switch file := fh.Handle.(type) {
	case ReadAtFile:
		out = make([]byte, in.Size)
		n, err := file.ReadAt(out, int64(in.Offset))
		switch {
		case err == nil || errors.Is(err, io.EOF):
			return out[:n], nil
		default:
			return nil, HErrorFromGoError(err)
		}
	default:
		fh.CacheBuffer, err = io.ReadAll(file)
		if err != nil {
			return nil, HErrorFromGoError(err)
		}

		if len(fh.CacheBuffer) == 0 {
			return nil, nil
		}
		goto useCache
	}
}

func (h *Handler) Mkdir(ctx context.Context, logger *slog.Logger, header *fuse.FuseInHeader, in *fuse.FuseMkdirIn, name string) (hErr *fuse.HError) {
	mkdirFs, ok := h.fs.(MkdirFS)
	if !ok {
		return fuse.ErrorNotSupported
	}

	inode := h.nodes.GetByIno(header.NodeId)
	if inode == nil {
		return fuse.ErrorNotFound
	}

	var info fs.FileInfo
	if statFS, ok := h.fs.(fs.StatFS); ok {
		var err error
		info, err = statFS.Stat(inode.Path)
		if err != nil {
			return HErrorFromGoError(err)
		}
	} else {
		file, err := h.fs.Open(inode.Path)
		if err != nil {
			return HErrorFromGoError(err)
		}

		info, err = file.Stat()
		file.Close()
		if err != nil {
			return HErrorFromGoError(err)
		}
	}

	if !info.IsDir() {
		return fuse.ErrorNotDirectory
	}

	fullname := path.Join(inode.Path, name)

	return HErrorFromGoError(mkdirFs.Mkdir(fullname, in.Mask, in.Mode))
}

// Provided filesystem may implement any of the following subtypes in order to support more operations
// When fs.StatFS is implemented any stat operation may be performed directly with the available path without first calling Open
// When fshandler.OpenFlagsFS is implemented, granular use of flags can be used when opening a file
// When fshandler.DestroyFS is implemented, custom cleanup code can be ran when the filesystem is unmounted
// TODO: When fshandler.MkdirFS is implemented, directory creation is permitted
// TODO: When fshandler.MknodFS is implemented, custom node creation is permitted
// Files could also extend their implementation with:
// When fshandler.ReadAtFile is implemented, no caching will be required for random reads.
// When fs.ReadDirFile is implemented, child files and directories can be obtained from a handle.
func New(f fs.FS) (h *Handler) {
	h = &Handler{
		fs: f,
	}
	h.nodes.New(".", false, true, nil)
	return h
}
