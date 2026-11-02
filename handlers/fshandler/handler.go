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

func (h *Handler) pathInfo(name string) (info fs.FileInfo, err error) {
	if statFS, ok := h.fs.(fs.StatFS); ok {
		return statFS.Stat(name)
	}

	file, err := h.fs.Open(name)
	if err != nil {
		return nil, HErrorFromGoError(err)
	}
	defer file.Close()

	return file.Stat()
}

func (h *Handler) GetAttr(ctx context.Context, logger *slog.Logger, header *fuse.FuseInHeader, in *fuse.FuseGetattrIn) (out fuse.FuseAttrOut, hErr *fuse.HError) {
	inode := h.nodes.GetByIno(header.NodeId)
	if inode == nil {
		return out, fuse.ErrorNotFound
	}

	switch statFS := h.fs.(type) {
	case fs.StatFS:
		var err error
		info, err := statFS.Stat(inode.Path)
		if err != nil {
			return out, HErrorFromGoError(err)
		}

		return fuse.FuseAttrOut{
			Attr: FuseAttrFromFileInfo(header.NodeId, info),
		}, nil
	default:
		file, err := h.fs.Open(inode.Path)
		if err != nil {
			return out, HErrorFromGoError(err)
		}
		defer file.Close()

		info, err := file.Stat()
		if err != nil {
			return out, HErrorFromGoError(err)
		}

		return fuse.FuseAttrOut{Attr: FuseAttrFromFileInfo(header.NodeId, info)}, nil
	}
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

	if fh == nil {
		return out, fuse.ErrorNotFound
	}

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
				Type:    tables.RawModeFromFileMode(info.Mode()),
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

	fullpath := path.Clean(path.Join(inode.Path, name))

	file, err := h.fs.Open(fullpath)
	if err != nil {
		return out, HErrorFromGoError(err)
	}
	defer file.Close()

	targetInode := h.nodes.New(path.Join(inode.Path, name), true, false, inode)

	info, err := file.Stat()
	if err != nil {
		return out, HErrorFromGoError(err)
	}

	return fuse.FuseEntryOut{
		NodeId: targetInode.NodeId,
		Attr:   FuseAttrFromFileInfo(targetInode.NodeId, info),
	}, nil
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
		logger.WarnContext(ctx, "Handle not found")
		return nil
	}

	if fh.Inode != inode {
		return fuse.ErrorPermissionDenied
	}

	err := fh.Handle.Close()
	if err != nil {
		logger.With("error-msg", err).WarnContext(ctx, "Failed to close directory handle")
	}

	h.fileHandles.Del(fh.FileDescriptor)

	return nil
}

func (h *Handler) Read(ctx context.Context, logger *slog.Logger, header *fuse.FuseInHeader, in *fuse.FuseReadIn) (out []byte, hErr *fuse.HError) {
	fh := h.fileHandles.GetByFd(in.FileHandle)

	if fh == nil {
		return out, fuse.ErrorNotFound
	}

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

func (h *Handler) Mkdir(ctx context.Context, logger *slog.Logger, header *fuse.FuseInHeader, in *fuse.FuseMkdirIn, name string) (out fuse.FuseEntryOut, hErr *fuse.HError) {
	mkdirFs, ok := h.fs.(MkdirFS)
	if !ok {
		return out, fuse.ErrorNotSupported
	}

	inode := h.nodes.GetByIno(header.NodeId)
	if inode == nil {
		return out, fuse.ErrorNotFound
	}

	info, err := h.pathInfo(inode.Path)
	if err != nil {
		return out, HErrorFromGoError(err)
	}

	if !info.IsDir() {
		return out, fuse.ErrorNotDirectory
	}

	fullpath := path.Clean(path.Join(inode.Path, name))

	newInode := h.nodes.New(fullpath, false, false, inode)

	err = mkdirFs.Mkdir(fullpath, in.Mask, in.Mode)
	if err != nil {
		return out, HErrorFromGoError(err)
	}

	dirInfo, err := h.pathInfo(fullpath)
	if err != nil {
		return out, HErrorFromGoError(err)
	}

	out.NodeId = newInode.NodeId
	out.Attr = FuseAttrFromFileInfo(newInode.NodeId, dirInfo)

	return out, nil
}

func (h *Handler) Unlink(ctx context.Context, logger *slog.Logger, hdr *fuse.FuseInHeader, name string) (hErr *fuse.HError) {
	unlinkFS, ok := h.fs.(UnlinkFS)
	if !ok {
		return fuse.ErrorNotSupported
	}

	inode := h.nodes.GetByIno(hdr.NodeId)
	if inode == nil {
		return fuse.ErrorNotFound
	}

	fullpath := path.Clean(path.Join(inode.Path, name))

	info, err := h.pathInfo(fullpath)
	if err != nil {
		return HErrorFromGoError(err)
	}

	if info.IsDir() {
		return fuse.ErrorIsDirectory
	}

	return HErrorFromGoError(unlinkFS.Unlink(fullpath))
}

func (h *Handler) Write(ctx context.Context, logger *slog.Logger, header *fuse.FuseInHeader, in *fuse.FuseWriteIn, buffer []byte) (out fuse.FuseWriteOut, hErr *fuse.HError) {
	fh := h.fileHandles.GetByFd(in.FileHandle)
	if fh == nil {
		return out, fuse.ErrorNotFound
	}

	switch file := fh.Handle.(type) {
	case WriteAtFile:
		n, err := file.WriteAt(buffer, int64(in.Offset))
		if err != nil {
			return out, HErrorFromGoError(err)
		}

		return fuse.FuseWriteOut{
			Size: uint32(n),
		}, nil
	default:
		return out, fuse.ErrorNotSupported
	}
}

func (h *Handler) Setattr(ctx context.Context, logger *slog.Logger, header *fuse.FuseInHeader, in *fuse.FuseSetattrIn) (out fuse.FuseAttrOut, hErr *fuse.HError) {
	inode := h.nodes.GetByIno(header.NodeId)
	if inode == nil {
		return out, fuse.ErrorNotFound
	}

	fh := h.fileHandles.GetByFd(in.FileHandle)

	var setattr Setattr
	if fh == nil {
		setattr.fromFuse(nil, in)
		// Attempt to use the file system interface
		setattrFS, ok := h.fs.(SetattrFS)
		if !ok {
			return out, fuse.ErrorNotSupported
		}

		info, err := setattrFS.Setattr(inode.Path, &setattr)
		if err != nil {
			return out, HErrorFromGoError(err)
		}

		return fuse.FuseAttrOut{
			Attr: FuseAttrFromFileInfo(inode.NodeId, info),
		}, nil
	}

	setattr.fromFuse(fh.Handle, in)
	setattrFile, ok := fh.Handle.(SetattrFile)
	if ok {
		info, err := setattrFile.Setattr(&setattr)
		if err != nil {
			return out, HErrorFromGoError(err)
		}

		return fuse.FuseAttrOut{
			Attr: FuseAttrFromFileInfo(inode.NodeId, info),
		}, nil
	}

	setattrFS, ok := h.fs.(SetattrFS)
	if !ok {
		return out, fuse.ErrorNotSupported
	}
	info, err := setattrFS.Setattr(inode.Path, &setattr)
	if err != nil {
		return out, HErrorFromGoError(err)
	}

	return fuse.FuseAttrOut{
		Attr: FuseAttrFromFileInfo(inode.NodeId, info),
	}, nil
}

func (h *Handler) Flush(ctx context.Context, logger *slog.Logger, header *fuse.FuseInHeader, in *fuse.FuseFlushIn) (hErr *fuse.HError) {
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

func (h *Handler) Fsync(ctx context.Context, logger *slog.Logger, header *fuse.FuseInHeader, in *fuse.FuseFsyncIn) (hErr *fuse.HError) {
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

	if syncFile, ok := fh.Handle.(SyncFile); ok {
		return HErrorFromGoError(syncFile.Sync())
	}
	return fuse.ErrorNotSupported
}

func (h *Handler) Rmdir(ctx context.Context, logger *slog.Logger, hdr *fuse.FuseInHeader, name string) (hErr *fuse.HError) {
	rmdirFS, ok := h.fs.(RmdirFS)
	if !ok {
		return fuse.ErrorNotSupported
	}

	inode := h.nodes.GetByIno(hdr.NodeId)
	if inode == nil {
		return fuse.ErrorNotFound
	}

	fullpath := path.Clean(path.Join(inode.Path, name))
	info, err := h.pathInfo(fullpath)
	if err != nil {
		return HErrorFromGoError(err)
	}

	if !info.IsDir() {
		return fuse.ErrorNotDirectory
	}

	return HErrorFromGoError(rmdirFS.Rmdir(fullpath))
}

func (h *Handler) Symlink(ctx context.Context, logger *slog.Logger, hdr *fuse.FuseInHeader, name, oldname string) (out fuse.FuseEntryOut, hErr *fuse.HError) {
	symlinkFS, ok := h.fs.(SymlinkFS)
	if !ok {
		return out, fuse.ErrorNotSupported
	}

	parent := h.nodes.GetByIno(hdr.NodeId)
	if parent == nil {
		return out, fuse.ErrorNotFound
	}

	parentInfo, err := h.pathInfo(parent.Path)
	if err != nil {
		return out, HErrorFromGoError(err)
	}

	if !parentInfo.IsDir() {
		return out, fuse.ErrorNotDirectory
	}

	fullpath := path.Clean(path.Join(parent.Path, name))

	symlinkInode := h.nodes.New(fullpath, false, false, parent)

	err = symlinkFS.Symlink(oldname, fullpath)
	if err != nil {
		return out, HErrorFromGoError(err)
	}

	symlinkInfo, err := h.pathInfo(fullpath)
	if err != nil {
		return out, HErrorFromGoError(err)
	}

	out.NodeId = symlinkInode.NodeId
	out.Attr = FuseAttrFromFileInfo(symlinkInode.NodeId, symlinkInfo)
	return out, nil
}

func (h *Handler) Readlink(ctx context.Context, logger *slog.Logger, hdr *fuse.FuseInHeader) (oldname []byte, hErr *fuse.HError) {
	symlinkFS, ok := h.fs.(SymlinkFS)
	if !ok {
		return nil, fuse.ErrorNotSupported
	}

	inode := h.nodes.GetByIno(hdr.NodeId)
	if inode == nil {
		return nil, fuse.ErrorNotFound
	}

	var info fs.FileInfo
	if statFS, ok := h.fs.(fs.StatFS); ok {
		var err error
		info, err = statFS.Stat(inode.Path)
		if err != nil {
			return nil, HErrorFromGoError(err)
		}
	} else {
		file, err := h.fs.Open(inode.Path)
		if err != nil {
			return nil, HErrorFromGoError(err)
		}

		info, err = file.Stat()
		file.Close()
		if err != nil {
			return nil, HErrorFromGoError(err)
		}
	}

	if info.Mode().Type() != fs.ModeSymlink {
		return nil, fuse.ErrorInvalidValue
	}

	fullpath := path.Clean(inode.Path)

	oldnameS, err := symlinkFS.Readlink(fullpath)
	if err != nil {
		return nil, HErrorFromGoError(err)
	}

	return []byte(oldnameS), nil
}

func (h *Handler) Rename(ctx context.Context, logger *slog.Logger, hdr *fuse.FuseInHeader, in *fuse.FuseRenameIn, oldname, newname string) (hErr *fuse.HError) {
	renameFS, ok := h.fs.(RenameFS)
	if !ok {
		return fuse.ErrorNotSupported
	}

	oldParent := h.nodes.GetByIno(hdr.NodeId)
	if oldParent == nil {
		return fuse.ErrorNotFound
	}

	oldParentInfo, err := h.pathInfo(oldParent.Path)
	if err != nil {
		return HErrorFromGoError(err)
	}

	if !oldParentInfo.IsDir() {
		return fuse.ErrorNotDirectory
	}

	oldpath := path.Clean(path.Join(oldParent.Path, oldname))
	target := h.nodes.GetByPath(oldpath)
	if target == nil {
		return fuse.ErrorNotFound
	}

	newParent := h.nodes.GetByIno(in.NewDir)
	if newParent == nil {
		return fuse.ErrorNotFound
	}

	newParentInfo, err := h.pathInfo(newParent.Path)
	if err != nil {
		return HErrorFromGoError(err)
	}

	if !newParentInfo.IsDir() {
		return fuse.ErrorNotDirectory
	}

	newpath := path.Clean(path.Join(newParent.Path, newname))

	err = renameFS.Rename(oldpath, newpath)
	if err != nil {
		return HErrorFromGoError(err)
	}

	target.Path = newpath
	delete(h.nodes.Paths, oldpath)
	h.nodes.Paths[newpath] = target

	return nil
}

// TODO: Update with all implementations
// Provided filesystem may implement any of the following subtypes in order to support more operations
// When fs.StatFS is implemented any stat operation may be performed directly with the available path without first calling Open
// When fshandler.OpenFlagsFS is implemented, granular use of flags can be used when opening a file
// When fshandler.DestroyFS is implemented, custom cleanup code can be ran when the filesystem is unmounted
// When fshandler.MkdirFS is implemented, directory creation is permitted
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
