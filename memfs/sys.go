//go:build openbsd

package memfs

import (
	"io/fs"
	"time"

	"github.com/RogueTeam/openfuse/handlers/fshandler"
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
	name       string
	attributes fshandler.Attributes
}

var (
	_ fs.FileInfo           = (*fileinfo)(nil)
	_ fshandler.ExtFileInfo = (*fileinfo)(nil)
)

func (i *fileinfo) Name() string                          { return i.name }
func (i *fileinfo) Size() int64                           { return int64(i.attributes.Size) }
func (i *fileinfo) Mode() fs.FileMode                     { return i.attributes.FileMode() }
func (i *fileinfo) ModTime() time.Time                    { return i.attributes.ModTime() }
func (i *fileinfo) IsDir() bool                           { return i.attributes.FileMode().IsDir() }
func (i *fileinfo) Sys() any                              { return nil }
func (i *fileinfo) Ext() (extended *fshandler.Attributes) { return &i.attributes }
