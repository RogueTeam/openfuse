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
	"github.com/RogueTeam/openfuse/handlers/fshandler/tables"
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
				Ino:     h.nodes.New(fh.Inode, entryPath, false, false).NodeId,
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
	parent := h.nodes.GetByIno(header.NodeId)
	if parent == nil {
		return out, fuse.ErrorNotFound
	}

	fullpath := path.Clean(path.Join(parent.Path, name))

	info, err := h.pathInfo(fullpath)
	if err != nil {
		return out, HErrorFromGoError(err)
	}

	targetInode := h.nodes.New(parent, path.Join(parent.Path, name), true, false)
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

	return h.nodes.ForgetByIno(inode.NodeId, in.Nlookup)
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

	parent := h.nodes.GetByIno(header.NodeId)
	if parent == nil {
		return out, fuse.ErrorNotFound
	}

	info, err := h.pathInfo(parent.Path)
	if err != nil {
		return out, HErrorFromGoError(err)
	}

	if !info.IsDir() {
		return out, fuse.ErrorNotDirectory
	}

	fullpath := path.Clean(path.Join(parent.Path, name))

	err = mkdirFs.Mkdir(fullpath, in.Mask, in.Mode)
	if err != nil {
		return out, HErrorFromGoError(err)
	}

	dirInfo, err := h.pathInfo(fullpath)
	if err != nil {
		return out, HErrorFromGoError(err)
	}

	newInode := h.nodes.New(parent, fullpath, true, false)
	out.NodeId = newInode.NodeId
	out.Attr = FuseAttrFromFileInfo(newInode.NodeId, dirInfo)

	return out, nil
}

func (h *Handler) Mknod(ctx context.Context, logger *slog.Logger, header *fuse.FuseInHeader, in *fuse.FuseMknodIn, name string) (out fuse.FuseEntryOut, hErr *fuse.HError) {
	mknodFs, ok := h.fs.(MknodFS)
	if !ok {
		return out, fuse.ErrorNotSupported
	}

	parent := h.nodes.GetByIno(header.NodeId)
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

	err = mknodFs.Mknod(fullpath, in.Rdev, in.Umask, in.Mode)
	if err != nil {
		return out, HErrorFromGoError(err)
	}

	nodInfo, err := h.pathInfo(fullpath)
	if err != nil {
		return out, HErrorFromGoError(err)
	}

	newInode := h.nodes.New(parent, fullpath, true, false)
	out.NodeId = newInode.NodeId
	out.Attr = FuseAttrFromFileInfo(newInode.NodeId, nodInfo)

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

	var setattr Setattr
	// There is no valid for filehandle field so use the fuse spec as reference.
	// valid handles start from 1
	if in.FileHandle != 0 {
		fh := h.fileHandles.GetByFd(in.FileHandle)
		if fh == nil {
			return out, fuse.ErrorNotFound
		}

		setattrFile, ok := fh.Handle.(SetattrFile)
		if !ok {
			goto useFS
		}

		setattr.fromFuse(fh.Handle, in)
		info, err := setattrFile.Setattr(&setattr)
		if err != nil {
			return out, HErrorFromGoError(err)
		}

		return fuse.FuseAttrOut{
			Attr: FuseAttrFromFileInfo(inode.NodeId, info),
		}, nil
	}

useFS:
	setattr.fromFuse(nil, in)
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

	if syncFile, ok := fh.Handle.(SyncFile); ok {
		return HErrorFromGoError(syncFile.Sync())
	}

	return nil
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

	err = symlinkFS.Symlink(oldname, fullpath)
	if err != nil {
		return out, HErrorFromGoError(err)
	}

	symlinkInfo, err := h.pathInfo(fullpath)
	if err != nil {
		return out, HErrorFromGoError(err)
	}

	symlinkInode := h.nodes.New(parent, fullpath, true, false)
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

	h.nodes.ChangeDirectory(newParent, target, newpath)
	return nil
}

func (h *Handler) Statfs(ctx context.Context, logger *slog.Logger, hdr *fuse.FuseInHeader) (out fuse.FuseStatfsOut, hErr *fuse.HError) {
	statfsFS, ok := h.fs.(StatfsFS)
	if !ok {
		return out, fuse.ErrorNotSupported
	}
	return statfsFS.Statfs(), nil
}

func (h *Handler) Link(ctx context.Context, logger *slog.Logger, hdr *fuse.FuseInHeader, inLink *fuse.FuseLinkIn, name string) (out fuse.FuseEntryOut, hErr *fuse.HError) {
	linkFS, ok := h.fs.(LinkFS)
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

	old := h.nodes.GetByIno(inLink.OldNodeId)
	if old == nil {
		return out, fuse.ErrorNotFound
	}

	fullpath := path.Clean(path.Join(parent.Path, name))

	err = linkFS.Link(old.Path, fullpath)
	if err != nil {
		return out, HErrorFromGoError(err)
	}

	linkInfo, err := h.pathInfo(fullpath)
	if err != nil {
		return out, HErrorFromGoError(err)
	}

	// TODO: This is incorrect - Hardlinks doesn't create new inodes
	linkInode := h.nodes.New(parent, fullpath, true, false)
	out.NodeId = linkInode.NodeId
	out.Attr = FuseAttrFromFileInfo(linkInode.NodeId, linkInfo)
	return out, nil
}

// Provided filesystem may implement FullFS or a subset of it in order to support non Read-Only filesystems.
// Files and directories may need to implement FullFile or FullDir accordingly in order to support all file and directory operations.
// ExtFileInfo can be used to return more detailed attributes to the fuse device
func New(f fs.FS) (h *Handler) {
	h = &Handler{
		fs: f,
	}
	h.nodes.New(nil, ".", true, true)
	return h
}
