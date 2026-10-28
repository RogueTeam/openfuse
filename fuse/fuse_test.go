//go:build openbsd

package fuse_test

import (
	"testing"

	"github.com/RogueTeam/gopenfuse/fshandler"
	"github.com/RogueTeam/gopenfuse/fuse"
	"github.com/RogueTeam/gopenfuse/testsuite/samplefs"
)

func Test_Mount(t *testing.T) {
	t.Run("Single instance", func(t *testing.T) {
		i, err := fuse.Mount(t.TempDir(), fuse.Options{Name: "single"}, fshandler.New(samplefs.RootFS))
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
		i1, err := fuse.Mount(t.TempDir(), fuse.Options{Name: "first"}, fshandler.New(samplefs.RootFS))
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

		i2, err := fuse.Mount(t.TempDir(), fuse.Options{Name: "second"}, fshandler.New(samplefs.RootFS))
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
