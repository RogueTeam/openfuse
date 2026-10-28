//go:build openbsd

package testsuite

import "github.com/RogueTeam/gopenfuse/fuse"

type NopHandler struct{}

var _ fuse.Handler = (*NopHandler)(nil)
