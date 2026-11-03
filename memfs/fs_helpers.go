//go:build openbsd

package memfs

import (
	"fmt"
	"io/fs"
	"path"
	"strings"
	"time"

	"github.com/RogueTeam/openfuse/handlers/fshandler"
	"github.com/RogueTeam/openfuse/handlers/tables"
	"golang.org/x/sys/unix"
)

func (f *FS) HelperMkdirAll(name string, mode fs.FileMode) (err error) {
	name = path.Clean(name)

	parts := strings.Split(name, "/")
	curr := &f.Root
	for index, part := range parts {
		if !curr.Info.IsDir() {
			return fmt.Errorf("one of the part already exists and is not a directory: %s", path.Join(parts[:1+index]...))
		}

		next := &Node{

			Info: FileInfo{
				SetName: part,
				Attributes: fshandler.Attributes{
					Mode:  tables.RawModeFromFileMode(mode) | unix.S_IFDIR,
					Mtime: uint64(time.Now().UnixMilli()),
				},
			},
			Children: make(map[string]*Node, 1),
		}

		curr.Children[part] = next
		curr = next
	}
	return nil
}

func (f *FS) HelperCreateFile[T ~string | ~[]byte](name string, mode fs.FileMode, contents T) (err error) {
	name = path.Clean(name)

	dir, filename := path.Split(name)

	err = f.HelperMkdirAll(dir, 0o700|fs.ModeDir)
	if err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	dirNode, err := f.Root.Find(dir)
	if err != nil {
		return fmt.Errorf("failed to find directory part: %s: %w", dir, err)
	}

	dirNode.Children[filename] = &Node{

		Info: FileInfo{
			SetName: filename,
			Attributes: fshandler.Attributes{
				Size:  uint64(len(contents)),
				Mode:  tables.RawModeFromFileMode(mode) | unix.S_IFREG,
				Mtime: uint64(time.Now().UnixMilli()),
			},
		},
		Contents: []byte(contents),
	}
	return nil
}
