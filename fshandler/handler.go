//go:build openbsd

package fshandler

import (
	"context"
	"io/fs"
	"log/slog"

	"github.com/RogueTeam/gopenfuse/fuse"
	"golang.org/x/sys/unix"
)

type Handler struct {
}

var _ fuse.Handler = (*Handler)(nil)

func (h *Handler) GetAttr(ctx context.Context, logger *slog.Logger, header *fuse.FuseInHeader, in *fuse.FuseGetattrIn) (out fuse.FuseAttrOut, err *fuse.HError) {
	if header.NodeId == 1 {
		return fuse.FuseAttrOut{
			Attr: fuse.FuseAttr{
				Ino:  1,
				Mode: 0o700 | unix.S_IFDIR,
			},
		}, nil
	}
	return out, fuse.ErrorNotFound
}

func New(f fs.FS) (h *Handler) {
	return &Handler{}
}
