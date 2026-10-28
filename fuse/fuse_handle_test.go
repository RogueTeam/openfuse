package fuse_test

import (
	"io/fs"
	"log/slog"
	"os"
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

		mpFs := os.DirFS(i.Mountpoint)

		err = fs.WalkDir(mpFs, "/", func(path string, d fs.DirEntry, err error) error {
			t.Logf("File/Dir: %s", path)
			return nil
		})
		if err != nil {
			t.Fatalf("failed to walk filesystem")
			return
		}
	})
}
