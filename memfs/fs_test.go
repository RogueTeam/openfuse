//go:build openbsd

package memfs_test

import (
	"bytes"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path"
	"slices"
	"strings"
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
				t.Logf("failed to close: %v", err)
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
				t.Logf("failed to close: %v", err)
			}
		})

		logger := testsuite.Logger(t, slog.LevelDebug)
		go func() {
			i.Serve(t.Context(), logger)
			i.Close()
		}()

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
				t.Logf("failed to close: %v", err)
			}
		})

		logger := testsuite.Logger(t, slog.LevelDebug)
		go func() {
			i.Serve(t.Context(), logger)
			i.Close()
		}()

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
				t.Logf("failed to close: %v", err)
			}
		})

		logger := testsuite.Logger(t, slog.LevelDebug)
		go func() {
			i.Serve(t.Context(), logger)
			i.Close()
		}()

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

				if string(contents) != entry.Contents {
					t.Fatalf("contents doesn't match")
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
				t.Logf("failed to close: %v", err)
			}
		})

		logger := testsuite.Logger(t, slog.LevelDebug)
		go func() {
			i.Serve(t.Context(), logger)
			i.Close()
		}()

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

func Test_Rmdir(t *testing.T) {
	t.Run("Succeed", func(t *testing.T) {
		type Entry struct {
			Path     string
			Mode     fs.FileMode
			Contents string
		}
		paths := []Entry{
			{Path: "remove", Mode: 0o700 | fs.ModeDir},
			{Path: "remove/README.md", Mode: 0o700, Contents: "Hello world"},
			{Path: "remove-all", Mode: 0o700 | fs.ModeDir},
			{Path: "remove-all/README.md", Mode: 0o700, Contents: "Hello world"},
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
				t.Logf("failed to close: %v", err)
			}
		})

		logger := testsuite.Logger(t, slog.LevelDebug)
		go func() {
			i.Serve(t.Context(), logger)
			i.Close()
		}()

		t.Run("RemoveAll", func(t *testing.T) {
			fullpath := path.Join(i.Mountpoint, "remove-all")

			info, err := os.Stat(fullpath)
			if err != nil {
				t.Fatalf("failed to get directory info: %v", err)
				return
			}

			if !info.IsDir() {
				t.Fatalf("expecting a directory")
				return
			}

			err = os.RemoveAll(fullpath)
			if err != nil {
				t.Fatalf("failed to remove all from root: %v", err)
				return
			}
		})
		t.Run("Remove", func(t *testing.T) {
			fullpath := path.Join(i.Mountpoint, "remove")

			info, err := os.Stat(fullpath)
			if err != nil {
				t.Fatalf("failed to get directory info: %v", err)
				return
			}

			if !info.IsDir() {
				t.Fatalf("expecting a directory")
				return
			}

			err = os.Remove(fullpath)
			if err == nil {
				t.Fatalf("should fail since directory is not empty")
				return
			}
		})

	})
}

func Test_Unlink(t *testing.T) {
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
				t.Logf("failed to close: %v", err)
			}
		})

		logger := testsuite.Logger(t, slog.LevelDebug)
		go func() {
			i.Serve(t.Context(), logger)
			i.Close()
		}()

		t.Run("Remove", func(t *testing.T) {
			for _, entry := range slices.Backward(paths) {
				t.Run(entry.Path, func(t *testing.T) {
					fullpath := path.Join(i.Mountpoint, entry.Path)

					err = os.Remove(fullpath)
					if err != nil {
						t.Fatalf("failed to remove file: %v", err)
						return
					}
				})
			}
		})
		t.Run("Walk", func(t *testing.T) {
			var visited int
			err := fs.WalkDir(os.DirFS(i.Mountpoint), ".", func(path string, d fs.DirEntry, err error) error {
				if path == "." {
					return nil
				}

				visited++
				return nil
			})
			if err != nil {
				t.Fatalf("failed to walk filesystem: %v", err)
				return
			}

			t.Logf("Visited: %v", visited)

			if visited > 0 {
				t.Fatalf("expecting no files: visited %d", visited)
				return
			}
		})
	})
}

func Test_OpenWriteRead(t *testing.T) {
	t.Run("Succeed", func(t *testing.T) {
		type Entry struct {
			Path     string
			Mode     fs.FileMode
			Contents string
		}
		paths := []Entry{
			{Path: "file1", Mode: 0o700, Contents: "file1"},
			{Path: "file2", Mode: 0o700, Contents: "file2"},
			{Path: "file3", Mode: 0o700, Contents: "file3"},
			{Path: "file4", Mode: 0o700, Contents: "file4"},
			{Path: "empty", Mode: 0o700, Contents: "file4"},
			{Path: "1kb", Mode: 0o700, Contents: strings.Repeat("A", 1024)},
			{Path: "32kb", Mode: 0o700, Contents: strings.Repeat("A", 32*1024)},
			{Path: "1mb", Mode: 0o700, Contents: strings.Repeat("A", 1024*1024)},
			{Path: "4mb", Mode: 0o700, Contents: strings.Repeat("A", 4*1024*1024)},
			{Path: "100mb", Mode: 0o700, Contents: strings.Repeat("A", 100*1024*1024)},
		}

		filesystem := memfs.New()
		for _, entry := range paths {
			switch entry.Mode.Type() {
			case 0:
				filesystem.HelperCreateFile(entry.Path, entry.Mode, "")
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
				t.Logf("failed to close: %v", err)
			}
		})

		logger := testsuite.Logger(t, slog.LevelDebug)
		go func() {
			i.Serve(t.Context(), logger)
			i.Close()
		}()

		t.Run("Write", func(t *testing.T) {
			for _, entry := range paths {
				if !entry.Mode.IsRegular() {
					t.Logf("Skipping: %s %v", entry.Path, entry.Mode)
					continue
				}

				t.Run(entry.Path, func(t *testing.T) {
					fullpath := path.Join(i.Mountpoint, entry.Path)

					file, err := os.OpenFile(fullpath, os.O_TRUNC|os.O_WRONLY, 0)
					if err != nil {
						t.Fatalf("failed to open file: %v", err)
						return
					}
					t.Cleanup(func() { file.Close() })

					_, err = file.Write([]byte(entry.Contents))
					if err != nil {
						t.Fatalf("failed to write contents: %v", err)
						return
					}

					err = file.Sync()
					if err != nil {
						t.Fatalf("failed to sync changes: %v", err)
						return
					}
				})
			}
		})

		t.Run("Verify", func(t *testing.T) {
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

					if string(contents) != entry.Contents {
						t.Fatalf("contents doesn't match: expect(%s) != received(%s)", entry.Contents, contents)
					}
				})
			}
		})

	})
}

func Test_Symlink(t *testing.T) {
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
				t.Logf("failed to close: %v", err)
			}
		})

		logger := testsuite.Logger(t, slog.LevelDebug)
		go func() {
			i.Serve(t.Context(), logger)
			i.Close()
		}()

		oldname := path.Join(i.Mountpoint, "root/README.md")
		newname := path.Join(i.Mountpoint, "README.md")
		t.Run("Create", func(t *testing.T) {
			err := os.Symlink(oldname, newname)
			if err != nil {
				t.Fatalf("failed to create symlink: %v", err)
				return
			}
		})
		t.Run("Verify", func(t *testing.T) {
			oldc, err := os.ReadFile(oldname)
			if err != nil {
				t.Fatalf("failed to read old file")
				return
			}

			newc, err := os.ReadFile(newname)
			if err != nil {
				t.Fatalf("failed to read new file")
				return
			}

			if !bytes.Equal(oldc, newc) {
				t.Fatalf("contents doesn't match expected(%s) != received(%s)", oldc, newc)
				return
			}
		})
	})
}

func Test_Rename(t *testing.T) {
	t.Run("Succeed", func(t *testing.T) {
		type Entry struct {
			Path     string
			Mode     fs.FileMode
			Contents string
		}
		paths := []Entry{
			{Path: "inside", Mode: 0o700 | fs.ModeDir},
			{Path: "inside/README.md", Mode: 0o700, Contents: "Hello world"},
			{Path: "inside-diff", Mode: 0o700 | fs.ModeDir},
			{Path: "inside-diff/README.md", Mode: 0o700, Contents: "Hello world"},
			{Path: "outside", Mode: 0o700 | fs.ModeDir},
			{Path: "outside/README.md", Mode: 0o700, Contents: "Hello world"},
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
				t.Logf("failed to close: %v", err)
			}
		})

		logger := testsuite.Logger(t, slog.LevelDebug)
		go func() {
			i.Serve(t.Context(), logger)
			i.Close()
		}()

		t.Run("Inside", func(t *testing.T) {
			t.Run("Same name", func(t *testing.T) {
				oldname := path.Join(i.Mountpoint, "inside/README.md")
				newname := path.Join(i.Mountpoint, "README.md")
				t.Run("Rename", func(t *testing.T) {
					err := os.Rename(oldname, newname)
					if err != nil {
						t.Fatalf("failed to rename: %v", err)
						return
					}
				})
				t.Run("Verify", func(t *testing.T) {
					oldc, err := os.ReadFile(oldname)
					if err == nil {
						t.Fatalf("read to old file should fail")
						return
					}

					newc, err := os.ReadFile(newname)
					if err != nil {
						t.Fatalf("failed to read new file")
						return
					}

					if bytes.Equal(oldc, newc) {
						t.Fatalf("contents should match received(%v)", newc)
						return
					}
				})
			})
			t.Run("Diff name", func(t *testing.T) {
				oldname := path.Join(i.Mountpoint, "inside-diff/README.md")
				newname := path.Join(i.Mountpoint, "diff-README.md")
				t.Run("Rename", func(t *testing.T) {
					err := os.Rename(oldname, newname)
					if err != nil {
						t.Fatalf("failed to rename: %v", err)
						return
					}
				})
				t.Run("Verify", func(t *testing.T) {
					oldc, err := os.ReadFile(oldname)
					if err == nil {
						t.Fatalf("read to old file should fail")
						return
					}

					newc, err := os.ReadFile(newname)
					if err != nil {
						t.Fatalf("failed to read new file")
						return
					}

					if bytes.Equal(oldc, newc) {
						t.Fatalf("contents should match received(%v)", newc)
						return
					}
				})
			})
		})
		t.Run("Ouside", func(t *testing.T) {
			oldname := path.Join(i.Mountpoint, "outside/README.md")
			newname := path.Join(t.TempDir(), "README.md")
			t.Run("Rename", func(t *testing.T) {
				err := os.Rename(oldname, newname)
				if err == nil {
					t.Fatalf("rename should fail: %v", err)
					return
				}
			})
			t.Run("Verify", func(t *testing.T) {
				oldc, err := os.ReadFile(oldname)
				if err != nil {
					t.Fatalf("failed to read old file")
					return
				}

				newc, err := os.ReadFile(newname)
				if err == nil {
					t.Fatalf("other path should not exists")
					return
				}

				if bytes.Equal(oldc, newc) {
					t.Fatalf("contents should not match received(%s)", newc)
					return
				}
			})
		})
	})
}

func Test_Statfs(t *testing.T) {
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
				t.Logf("failed to close: %v", err)
			}
		})

		logger := testsuite.Logger(t, slog.LevelDebug)
		go func() {
			i.Serve(t.Context(), logger)
			i.Close()
		}()

		t.Run("Statfs", func(t *testing.T) {
			cmd := exec.CommandContext(t.Context(), "df", "-h", i.Mountpoint)
			cmd.Stdout, cmd.Stderr = t.Output(), t.Output()
			err := cmd.Run()
			if err != nil {
				t.Fatalf("failed to run command: %v", err)
				return
			}
		})
	})
}

func Test_Mknod(t *testing.T) {
	t.Run("Succeed", func(t *testing.T) {
		type Entry struct {
			Path     string
			Mode     fs.FileMode
			Contents string
		}
		paths := []Entry{
			{Path: "root", Mode: 0o700 | fs.ModeDir},
			{Path: "root/README.md", Mode: 0o700, Contents: "Hello world"},
			{Path: "README.md", Mode: 0o700, Contents: "Hello world"},
		}

		filesystem := memfs.New()
		for _, entry := range paths {
			switch entry.Mode.Type() {
			case 0:
				t.Logf("Skipping file: %s", entry.Path)
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
				t.Logf("failed to close: %v", err)
			}
		})

		logger := testsuite.Logger(t, slog.LevelDebug)
		go func() {
			i.Serve(t.Context(), logger)
			i.Close()
		}()

		t.Run("Mknod", func(t *testing.T) {
			for _, entry := range paths {
				if entry.Mode.IsDir() {
					continue
				}
				t.Run(entry.Path, func(t *testing.T) {
					fullpath := path.Clean(path.Join(i.Mountpoint, entry.Path))

					err := os.WriteFile(fullpath, []byte(entry.Contents), entry.Mode)
					if err != nil {
						t.Fatalf("failed to write file: %v", err)
					}

					t.Run("Twice", func(t *testing.T) {
						err := os.WriteFile(fullpath, []byte(entry.Contents), entry.Mode)
						if err != nil {
							t.Fatalf("failed to write file: %v", err)
						}
					})
				})
			}
		})
		t.Run("Verify", func(t *testing.T) {
			for _, entry := range paths {
				if entry.Mode.IsDir() {
					continue
				}
				t.Run(entry.Path, func(t *testing.T) {
					fullpath := path.Clean(path.Join(i.Mountpoint, entry.Path))

					contents, err := os.ReadFile(fullpath)
					if err != nil {
						t.Fatalf("failed to write file: %v", err)
					}

					if !bytes.Equal(contents, []byte(entry.Contents)) {
						t.Fatalf("contents doesn't match expected(%s) != received(%s)", entry.Contents, contents)
						return
					}
				})
			}
		})
	})
}

func Test_Hardlinks(t *testing.T) {
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
				t.Logf("failed to close: %v", err)
			}
		})

		logger := testsuite.Logger(t, slog.LevelDebug)
		go func() {
			i.Serve(t.Context(), logger)
			i.Close()
		}()

		oldname := path.Clean(path.Join(i.Mountpoint, "root/README.md"))
		newname := path.Clean(path.Join(i.Mountpoint, "README.md"))
		t.Run("Create", func(t *testing.T) {
			err := os.Link(oldname, newname)
			if err != nil {
				t.Fatalf("failed to create hardlink: %v", err)
				return
			}
		})
	})
}
