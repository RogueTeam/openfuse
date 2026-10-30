//go:build openbsd

package memfs

import (
	"fmt"
	"io/fs"
	"path"
	"strings"
	"time"
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
			Name: part,
			Info: fileinfo{
				name:  part,
				size:  0,
				mode:  (mode & ^fs.ModeType) | fs.ModeDir,
				mtime: time.Now(),
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
		Name: filename,
		Info: fileinfo{
			name:  filename,
			size:  int64(len(contents)),
			mode:  mode & ^fs.ModeType,
			mtime: time.Now(),
		},
		Contents: []byte(contents),
	}
	return nil
}
