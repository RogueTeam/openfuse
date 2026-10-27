//go:build openbsd

package fuse_test

import (
	"testing"

	"github.com/RogueTeam/gopenfuse/fuse"
)

func Test_Mount(t *testing.T) {
	t.Run("Single instance", func(t *testing.T) {
		i, err := fuse.Mount(t.TempDir(), fuse.Options{Name: "single"})
		if err != nil {
			t.Fatal("failed to create fuse instance")
		}
		t.Cleanup(func() {
			err := i.Close()
			if err != nil {
				t.Errorf("failed to close: %v", err)
			}
		})
	})
	t.Run("Multiple instances", func(t *testing.T) {
		i1, err := fuse.Mount(t.TempDir(), fuse.Options{Name: "first"})
		if err != nil {
			t.Fatal("failed to create fuse instance 1")
		}
		t.Cleanup(func() {
			err := i1.Close()
			if err != nil {
				t.Errorf("failed to close instance 1: %v", err)
			}
		})

		i2, err := fuse.Mount(t.TempDir(), fuse.Options{Name: "second"})
		if err != nil {
			t.Fatal("failed to create fuse instance 2")
		}
		t.Cleanup(func() {
			err := i2.Close()
			if err != nil {
				t.Errorf("failed to close instance 2: %v", err)
			}
		})
	})
}
