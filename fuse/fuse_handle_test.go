//go:build openbsd

package fuse_test

import (
	"io/fs"
	"log/slog"
	"os"
	"slices"
	"testing"

	"github.com/RogueTeam/gopenfuse/fshandler"
	"github.com/RogueTeam/gopenfuse/fuse"
	"github.com/RogueTeam/gopenfuse/testsuite"
	"github.com/RogueTeam/gopenfuse/testsuite/samplefs"
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
