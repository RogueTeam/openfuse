package samplefs

import "embed"

//go:embed root
var RootFS embed.FS

var Paths = []string{
	".",
	"root/",
	"root/README.md",
}
