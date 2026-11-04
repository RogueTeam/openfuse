//go:build openbsd

package fshandler

import (
	"io/fs"

	"github.com/RogueTeam/openfuse/fuse"
	"github.com/RogueTeam/openfuse/handlers/fshandler/tables"
)

func FuseAttrFromFileInfo(ino uint64, info fs.FileInfo) (out fuse.FuseAttr) {
	if extInfo, ok := info.(ExtFileInfo); ok {
		return extInfo.Ext().toFuse(ino)
	}
	mode := uint32(info.Mode().Perm()) | tables.RawModeFromFileMode(info.Mode())
	return fuse.FuseAttr{
		Ino:   ino,
		Size:  uint64(info.Size()),
		Mode:  mode,
		Mtime: uint64(info.ModTime().Unix()),
	}
}
