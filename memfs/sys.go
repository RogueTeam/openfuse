//go:build openbsd

package memfs

import (
	"io/fs"
	"time"

	"github.com/RogueTeam/openfuse/v8/handlers/fshandler"
)

type Dirent struct {
	Reference *Node
}

var _ fs.DirEntry = (*Dirent)(nil)

func (e *Dirent) Name() string {
	return e.Reference.Info.SetName
}

func (e *Dirent) IsDir() bool {
	return e.Reference.Info.IsDir()
}

func (e *Dirent) Type() fs.FileMode {
	return e.Reference.Info.Mode().Type()
}

func (e *Dirent) Info() (fs.FileInfo, error) {
	return &e.Reference.Info, nil
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
