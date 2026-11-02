//go:build openbsd

package memfs

import (
	"io/fs"
	"time"

	"github.com/RogueTeam/openfuse/handlers/fshandler"
)

type Dirent struct {
	SetName string
	SetInfo fs.FileInfo
}

var _ fs.DirEntry = (*Dirent)(nil)

func (e *Dirent) Name() string {
	return e.SetName
}

func (e *Dirent) IsDir() bool {
	return e.SetInfo.IsDir()
}

func (e *Dirent) Type() fs.FileMode {
	return e.SetInfo.Mode()
}

func (e *Dirent) Info() (fs.FileInfo, error) {
	return e.SetInfo, nil
}

type FileInfo struct {
	SetName    string
	Attributes fshandler.Attributes
}

var (
	_ fs.FileInfo           = (*FileInfo)(nil)
	_ fshandler.ExtFileInfo = (*FileInfo)(nil)
)

func (i *FileInfo) Name() string                          { return i.SetName }
func (i *FileInfo) Size() int64                           { return int64(i.Attributes.Size) }
func (i *FileInfo) Mode() fs.FileMode                     { return i.Attributes.FileMode() }
func (i *FileInfo) ModTime() time.Time                    { return i.Attributes.ModTime() }
func (i *FileInfo) IsDir() bool                           { return i.Attributes.FileMode().IsDir() }
func (i *FileInfo) Sys() any                              { return nil }
func (i *FileInfo) Ext() (extended *fshandler.Attributes) { return &i.Attributes }
