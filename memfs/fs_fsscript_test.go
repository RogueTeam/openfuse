//go:build openbsd

package memfs_test

import (
	"log/slog"
	"testing"

	"github.com/RogueTeam/openfuse/v8/fuse"
	"github.com/RogueTeam/openfuse/v8/handlers/fshandler"
	"github.com/RogueTeam/openfuse/v8/memfs"
	"github.com/RogueTeam/openfuse/v8/testsuite"
	"github.com/RogueTeam/openfuse/v8/testsuite/fsscript"
)

func Test_Fsscript(t *testing.T) {
	groups := fsscript.GenericScripts(t)

	for _, group := range groups {
		t.Run(group.File, func(t *testing.T) {
			for _, test := range group.Tests {
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

					ctx := fsscript.NewContext()
					t.Cleanup(func() { ctx.Close() })
					err = test.Run(ctx, i.Mountpoint)
					if err != nil {
						t.Fatalf("failed to run test: %v", err)
						return
					}
				})
			}
		})
	}
}
