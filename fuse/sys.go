package fuse

// Mapped directly from struct fusefs_args
type FuseFsArgs struct {
	Name       *byte
	Fd         int32
	MaxRead    int32
	AllowOther int32
}
