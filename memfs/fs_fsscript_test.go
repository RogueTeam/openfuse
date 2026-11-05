//go:build openbsd

package memfs_test

import (
	"log/slog"
	"testing"

	"github.com/RogueTeam/openfuse/fuse"
	"github.com/RogueTeam/openfuse/handlers/fshandler"
	"github.com/RogueTeam/openfuse/memfs"
	"github.com/RogueTeam/openfuse/testsuite"
	"github.com/RogueTeam/openfuse/testsuite/fsscript"
)

func Test_Fsscript(t *testing.T) {
	tests := fsscript.GenericScripts(t)

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			filesystem := memfs.New()

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

			logger := testsuite.Logger(t, slog.LevelError)
			go func() {
				i.Serve(t.Context(), logger)
				i.Close()
			}()

			err = test.Run(i.Mountpoint)
			if err != nil {
				t.Fatalf("failed to run test: %v", err)
				return
			}
		})
	}
}
