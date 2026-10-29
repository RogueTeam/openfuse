//go:build openbsd

package fuse_test

import (
	"io/fs"
	"log/slog"
	"os"
	"path"
	"slices"
	"testing"

	"github.com/RogueTeam/openfuse/fuse"
	"github.com/RogueTeam/openfuse/handlers/fshandler"
	"github.com/RogueTeam/openfuse/testsuite"
	"github.com/RogueTeam/openfuse/testsuite/samplefs"
)

func Test_Getattr(t *testing.T) {
	t.Run("Succeed", func(t *testing.T) {
		i, err := fuse.Mount(t.TempDir(), fuse.Options{Name: "simple"}, fshandler.New(samplefs.RootFS))
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
		i, err := fuse.Mount(t.TempDir(), fuse.Options{Name: "simple"}, fshandler.New(samplefs.RootFS))
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
					t.Logf("File/Dir: %s %v", path, d)
					visited = append(visited, path)
					return nil
				},
			)
			if err != nil {
				t.Fatalf("failed to walk filesystem")
				return
			}

			if len(visited) != len(samplefs.Paths) {
				t.Fatalf("Visited less files than available: visited(%d) != expecting(%d)", len(visited), len(samplefs.Paths))
				return
			}

			if !slices.EqualFunc(samplefs.Paths, visited, func(a samplefs.Entry, b string) bool { return a.Path == b }) {
				t.Fatalf("visited(%v) doesn't match expected(%v)", visited, samplefs.Paths)
				return
			}
		})
	})
}

func Test_OpenRead(t *testing.T) {
	t.Run("Succeed", func(t *testing.T) {
		i, err := fuse.Mount(t.TempDir(), fuse.Options{Name: "simple"}, fshandler.New(samplefs.RootFS))
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

		for _, entry := range samplefs.Paths {
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
