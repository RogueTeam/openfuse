//go:build openbsd

package fsscript_test

import (
	"testing"

	"github.com/RogueTeam/openfuse/testsuite/fsscript"
)

func Test_GenerateScripts(t *testing.T) {
	tests := fsscript.GenericScripts(t)
	t.Logf("Parsed: %d scripts", len(tests))
}
