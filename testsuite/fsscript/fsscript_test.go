//go:build openbsd

package fsscript_test

import (
	"testing"

	"github.com/RogueTeam/openfuse/v8/testsuite/fsscript"
)

func Test_GenerateScripts(t *testing.T) {
	groups := fsscript.GenericScripts(t)
	for _, group := range groups {
		t.Run(group.File, func(t *testing.T) {
			for _, test := range group.Tests {
				t.Run(test.Name, func(t *testing.T) {
					ctx := fsscript.NewContext()
					t.Cleanup(func() { ctx.Close() })
					err := test.Run(ctx, t.TempDir())
					if err != nil {
						t.Fatalf("failed to run test: %v", err)
					}
				})
			}
		})
	}
}
