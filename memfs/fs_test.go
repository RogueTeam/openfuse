//go:build openbsd

package memfs_test

import (
	"io/fs"
	"log/slog"
	"os"
	"path"
	"slices"
	"testing"

	"github.com/RogueTeam/openfuse/fuse"
	"github.com/RogueTeam/openfuse/handlers/fshandler"
	"github.com/RogueTeam/openfuse/memfs"
	"github.com/RogueTeam/openfuse/testsuite"
)

func Test_Mount(t *testing.T) {
	t.Run("Single instance", func(t *testing.T) {
		i, err := fuse.Mount(t.TempDir(), fuse.Options{Name: "single"}, fshandler.New(memfs.New()))
		if err != nil {
			t.Fatal("failed to create fuse instance")
		}
		t.Cleanup(func() {
			t.Logf("Closing running instance")
			err := i.Close()
			if err != nil {
				t.Errorf("failed to close: %v", err)
			}
		})
	})
	t.Run("Multiple instances", func(t *testing.T) {
		i1, err := fuse.Mount(t.TempDir(), fuse.Options{Name: "first"}, fshandler.New(memfs.New()))
		if err != nil {
			t.Fatal("failed to create fuse instance 1")
		}
		t.Cleanup(func() {
			t.Logf("Closing running instance 1")
			err := i1.Close()
			if err != nil {
				t.Errorf("failed to close instance 1: %v", err)
			}
		})

		i2, err := fuse.Mount(t.TempDir(), fuse.Options{Name: "second"}, fshandler.New(memfs.New()))
		if err != nil {
			t.Fatal("failed to create fuse instance 2")
		}
		t.Cleanup(func() {
			t.Logf("Closing running instance 2")
			err := i2.Close()
			if err != nil {
				t.Errorf("failed to close instance 2: %v", err)
			}
		})
	})
}

func Test_Getattr(t *testing.T) {
	t.Run("Succeed", func(t *testing.T) {
		i, err := fuse.Mount(t.TempDir(), fuse.Options{Name: "simple"}, fshandler.New(memfs.New()))
		if err != nil {
			t.Fatal("failed to create fuse instance")
		}
		t.Cleanup(func() {
			t.Logf("Closing running instance")
			err := i.Close()
			if err != nil {
				t.Errorf("failed to close: %v", err)
			}
		})

		logger := testsuite.Logger(t, slog.LevelDebug)
		go i.Serve(t.Context(), logger)

		info, err := os.Stat(i.Mountpoint)
		if err != nil {
			t.Fatalf("failed to stat mountpoint: %v", err)
			return
		}

		t.Logf("Info: %v", info)
	})
}

func Test_Readdir(t *testing.T) {
	t.Run("Succeed", func(t *testing.T) {
		type Entry struct {
			Path     string
			Mode     fs.FileMode
			Contents string
		}
		paths := []Entry{
			{Path: "root", Mode: 0o700},
			{Path: "root/README.md", Mode: 0o700, Contents: "Hello world"},
		}

		filesystem := memfs.New()
		for _, entry := range paths {
			switch entry.Mode.Type() {
			case 0:
				filesystem.HelperCreateFile(entry.Path, entry.Mode, entry.Contents)
			case fs.ModeDir:
				filesystem.HelperMkdirAll(entry.Path, entry.Mode)
			}
		}

		i, err := fuse.Mount(t.TempDir(), fuse.Options{Name: "simple"}, fshandler.New(filesystem))
		if err != nil {
			t.Fatal("failed to create fuse instance")
		}
		t.Cleanup(func() {
			t.Logf("Closing running instance")
			err := i.Close()
			if err != nil {
				t.Errorf("failed to close: %v", err)
			}
		})

		logger := testsuite.Logger(t, slog.LevelDebug)
		go i.Serve(t.Context(), logger)

		t.Run("Root", func(t *testing.T) {
			entries, err := os.ReadDir(i.Mountpoint)
			if err != nil {
				t.Fatalf("failed to readdir mountpoint: %v", err)
				return
			}

			t.Logf("Entries: %v", entries)
		})

		t.Run("Walk", func(t *testing.T) {
			var visited []string
			err = fs.WalkDir(
				os.DirFS(i.Mountpoint), ".",
				func(path string, d fs.DirEntry, err error) error {
					if path == "." {
						return nil
					}
					t.Logf("File/Dir: %s %v", path, d)
					visited = append(visited, path)
					return nil
				},
			)
			if err != nil {
				t.Fatalf("failed to walk filesystem")
				return
			}

			if len(visited) != len(paths) {
				t.Fatalf("Visited less files than available: visited(%d) != expecting(%d)", len(visited), len(paths))
				return
			}

			if !slices.EqualFunc(paths, visited, func(a Entry, b string) bool { return a.Path == b }) {
				t.Fatalf("visited(%v) doesn't match expected(%v)", visited, paths)
				return
			}
		})
	})
}

func Test_OpenRead(t *testing.T) {
	t.Run("Succeed", func(t *testing.T) {
		type Entry struct {
			Path     string
			Mode     fs.FileMode
			Contents string
		}
		paths := []Entry{
			{Path: "root", Mode: 0o700 | fs.ModeDir},
			{Path: "root/README.md", Mode: 0o700, Contents: "Hello world"},
		}

		filesystem := memfs.New()
		for _, entry := range paths {
			switch entry.Mode.Type() {
			case 0:
				filesystem.HelperCreateFile(entry.Path, entry.Mode, entry.Contents)
			case fs.ModeDir:
				filesystem.HelperMkdirAll(entry.Path, entry.Mode)
			}
		}

		i, err := fuse.Mount(t.TempDir(), fuse.Options{Name: "simple"}, fshandler.New(filesystem))
		if err != nil {
			t.Fatal("failed to create fuse instance")
		}
		t.Cleanup(func() {
			t.Logf("Closing running instance")
			err := i.Close()
			if err != nil {
				t.Errorf("failed to close: %v", err)
			}
		})

		logger := testsuite.Logger(t, slog.LevelDebug)
		go i.Serve(t.Context(), logger)

		for _, entry := range paths {
			if !entry.Mode.IsRegular() {
				t.Logf("Skipping: %s %v", entry.Path, entry.Mode)
				continue
			}

			t.Run(entry.Path, func(t *testing.T) {
				fullpath := path.Join(i.Mountpoint, entry.Path)
				info, err := os.Stat(fullpath)
				if err != nil {
					t.Fatalf("failed to get file stat: %v", err)
					return
				}

				contents, err := os.ReadFile(fullpath)
				if err != nil {
					t.Fatalf("failed to read file: %v", err)
					return
				}

				if info.Size() != int64(len(contents)) {
					t.Fatalf("size doesn't match: expected(%d) != received(%d)", info.Size(), len(contents))
				}
			})
		}
	})
}

func Test_Mkdir(t *testing.T) {
	t.Run("Succeed", func(t *testing.T) {
		i, err := fuse.Mount(
			t.TempDir(),
			fuse.Options{Name: "simple"},
			fshandler.New(memfs.New()),
		)
		if err != nil {
			t.Fatal("failed to create fuse instance")
		}
		t.Cleanup(func() {
			t.Logf("Closing running instance")
			err := i.Close()
			if err != nil {
				t.Errorf("failed to close: %v", err)
			}
		})

		logger := testsuite.Logger(t, slog.LevelDebug)
		go i.Serve(t.Context(), logger)

		type Entry struct {
			Path string
			Perm fs.FileMode
		}
		paths := []Entry{
			{Path: "example", Perm: 0o700 | os.ModeDir},
			{Path: "example/1", Perm: 0o755 | os.ModeDir},
			{Path: "example/2", Perm: 0o755 | os.ModeDir},
			{Path: "example/3", Perm: 0o744 | os.ModeDir},
		}

		t.Run("Create", func(t *testing.T) {
			for _, entry := range paths {
				t.Run(entry.Path, func(t *testing.T) {
					fullname := path.Join(i.Mountpoint, entry.Path)
					err := os.MkdirAll(fullname, entry.Perm)
					if err != nil {
						t.Fatalf("failed to create directory: %v", err)
						return
					}
				})
			}
		})
		t.Run("Verify", func(t *testing.T) {
			for _, entry := range paths {
				t.Run(entry.Path, func(t *testing.T) {
					fullname := path.Join(i.Mountpoint, entry.Path)
					info, err := os.Stat(fullname)
					if err != nil {
						t.Fatalf("failed to stat directory: %v", err)
						return
					}

					if info.Mode() != entry.Perm {
						t.Fatalf("expecting the same perms: expected(%v) != received(%v)", entry.Perm, info.Mode())
						return
					}
				})
			}
		})
		t.Run("Walk", func(t *testing.T) {
			var visited []string
			err := fs.WalkDir(os.DirFS(i.Mountpoint), ".", func(path string, d fs.DirEntry, err error) error {
				if path == "." {
					return nil
				}

				visited = append(visited, path)
				return nil
			})
			if err != nil {
				t.Fatalf("failed to walk filesystem: %v", err)
				return
			}

			t.Logf("Visited: %v", visited)

			equal := slices.EqualFunc(paths, visited, func(a Entry, b string) bool {
				return a.Path == b
			})
			if !equal {
				t.Fatalf("visited(%v) != expected(%v)", visited, paths)
				return
			}
		})
	})
}
