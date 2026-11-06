//go:build openbsd

package testsuite

import (
	"io/fs"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/RogueTeam/openfuse/v8/handlers/fshandler"
	"github.com/RogueTeam/openfuse/v8/handlers/fshandler/tables"
	"github.com/RogueTeam/openfuse/v8/memfs"
	"golang.org/x/sys/unix"
)

type PrebuiltEntry struct {
	Path     string
	Mode     fs.FileMode
	Contents string
}

func Generate(tb testing.TB, f *memfs.FS, in []PrebuiltEntry) {
	for _, entry := range in {
		switch entry.Mode.Type() {
		case 0:
			CreateFile(tb, f, entry.Path, entry.Mode, entry.Contents)
		case fs.ModeDir:
			MkdirAll(tb, f, entry.Path, entry.Mode)
		}
	}
}

func MkdirAll(tb testing.TB, f *memfs.FS, name string, mode fs.FileMode) {
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
						Mtime: uint64(time.Now().Unix()),
					},
				},
				Children: make(map[string]*memfs.Node, 1),
			}
			curr.Children[part] = next
		}
		curr = next
	}
}

func CreateFile[T ~string | ~[]byte](tb testing.TB, f *memfs.FS, name string, mode fs.FileMode, contents T) {
	name = path.Clean(name)

	dir, filename := path.Split(name)

	MkdirAll(tb, f, dir, 0o700|fs.ModeDir)

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
				Mtime: uint64(time.Now().Unix()),
			},
		},
		Contents: []byte(contents),
	}
}
