//go:build openbsd

package testsuite

import (
	"io/fs"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/RogueTeam/openfuse/handlers/fshandler"
	"github.com/RogueTeam/openfuse/handlers/tables"
	"github.com/RogueTeam/openfuse/memfs"
	"golang.org/x/sys/unix"
)

type MemfsPrebuiltEntry struct {
	Path     string
	Mode     fs.FileMode
	Contents string
}

func MemfsGenerate(tb testing.TB, f *memfs.FS, in []MemfsPrebuiltEntry) {
	for _, entry := range in {
		switch entry.Mode.Type() {
		case 0:
			MemfsCreateFile(tb, f, entry.Path, entry.Mode, entry.Contents)
		case fs.ModeDir:
			MemfsMkdirAll(tb, f, entry.Path, entry.Mode)
		}
	}
}

func MemfsMkdirAll(tb testing.TB, f *memfs.FS, name string, mode fs.FileMode) {
	name = path.Clean(name)

	parts := strings.Split(name, "/")
	curr := &f.Root
	for index, part := range parts {
		if !curr.Info.IsDir() {
			tb.Fatalf("one of the part already exists and is not a directory: %s", path.Join(parts[:1+index]...))
			return
		}

		next, found := curr.Children[part]
		if !found {
			next = &memfs.Node{
				Info: memfs.FileInfo{
					SetName: part,
					Attributes: fshandler.Attributes{
						Mode:  tables.RawModeFromFileMode(mode) | unix.S_IFDIR,
						Mtime: uint64(time.Now().UnixMilli()),
					},
				},
				Children: make(map[string]*memfs.Node, 1),
			}
			curr.Children[part] = next
		}
		curr = next
	}
}

func MemfsCreateFile[T ~string | ~[]byte](tb testing.TB, f *memfs.FS, name string, mode fs.FileMode, contents T) {
	name = path.Clean(name)

	dir, filename := path.Split(name)

	MemfsMkdirAll(tb, f, dir, 0o700|fs.ModeDir)

	parentNode, err := f.Root.Find(dir)
	if err != nil {
		tb.Fatalf("failed to get parent node: %v", err)
		return
	}

	if !parentNode.Info.IsDir() {
		tb.Fatalf("parent is not a directory")
		return
	}

	parentNode.Children[filename] = &memfs.Node{
		Info: memfs.FileInfo{
			SetName: filename,
			Attributes: fshandler.Attributes{
				Size:  uint64(len(contents)),
				Mode:  tables.RawModeFromFileMode(mode) | unix.S_IFREG,
				Mtime: uint64(time.Now().UnixMilli()),
			},
		},
		Contents: []byte(contents),
	}
}
