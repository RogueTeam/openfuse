package memfs

import (
	"io/fs"
	"time"
)

type direntry struct {
	name string
	info fs.FileInfo
}

var _ fs.DirEntry = (*direntry)(nil)

func (e *direntry) Name() string {
	return e.name
}

func (e *direntry) IsDir() bool {
	return e.info.IsDir()
}

func (e *direntry) Type() fs.FileMode {
	return e.info.Mode()
}

func (e *direntry) Info() (fs.FileInfo, error) {
	return e.info, nil
}

type fileinfo struct {
	name  string
	size  int64
	mode  fs.FileMode
	mtime time.Time
}

var _ fs.FileInfo = (*fileinfo)(nil)

func (i *fileinfo) Name() string       { return i.name }
func (i *fileinfo) Size() int64        { return i.size }
func (i *fileinfo) Mode() fs.FileMode  { return i.mode }
func (i *fileinfo) ModTime() time.Time { return i.mtime }
func (i *fileinfo) IsDir() bool        { return i.mode.IsDir() }
func (i *fileinfo) Sys() any           { return nil }
