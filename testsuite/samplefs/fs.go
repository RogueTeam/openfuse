package samplefs

import (
	"embed"
	"io/fs"
)

//go:embed root
var RootFS embed.FS

type Entry struct {
	Path string
	Mode fs.FileMode
}

var Paths = []Entry{
	{Path: ".", Mode: fs.ModeDir},
	{Path: "root", Mode: fs.ModeDir},
	{Path: "root/README.md", Mode: fs.ModeIrregular},
}
