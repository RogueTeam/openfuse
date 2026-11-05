//go:build openbsd

package fsscript

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"slices"
	"strings"
)

type Mkdir struct {
	Line       int
	Directory  string
	Permission fs.FileMode
}

func (m *Mkdir) GetName() (name string) { return "mkdir" }
func (m *Mkdir) GetLine() (line int)    { return m.Line }
func (m *Mkdir) Run(base string) (err error) {
	dir := path.Clean(path.Join(base, m.Directory))

	return os.Mkdir(dir, m.Permission)
}

type Readdir struct {
	Line      int
	Directory string
	Expect    []string
}

func (r *Readdir) GetName() (name string) { return "ls" }
func (r *Readdir) GetLine() (line int)    { return r.Line }
func (r *Readdir) Run(base string) (err error) {
	dir := path.Clean(path.Join(base, r.Directory))

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	if len(r.Expect) == 0 {
		if len(entries) > 0 {
			return fmt.Errorf("expecting no values")
		}
		return nil
	}

	if len(r.Expect) != len(entries) {
		return fmt.Errorf("expecting %d elements; got %d", len(r.Expect), len(entries))
	}

	slices.SortFunc(entries, func(a, b fs.DirEntry) int { return strings.Compare(a.Name(), b.Name()) })
	slices.Sort(r.Expect)

	for index, entry := range entries {
		expect := r.Expect[index]
		if expect != entry.Name() {
			return fmt.Errorf("element %d doesn't match. expect(%s) != received(%s)", index, expect, entry.Name())
		}
	}

	return nil
}

type CreateFile struct {
	Line       int
	File       string
	Permission fs.FileMode
	Contents   []string
}

func (c *CreateFile) GetName() (name string) { return "file" }
func (c *CreateFile) GetLine() (line int)    { return c.Line }
func (c *CreateFile) Run(base string) (err error) {
	filepath := path.Clean(path.Join(base, c.File))
	return os.WriteFile(filepath, []byte(strings.Join(c.Contents, " ")), c.Permission)
}

type ReadFile struct {
	Line     int
	File     string
	Contents []string
}

func (r *ReadFile) GetName() (name string) { return "cat" }
func (r *ReadFile) GetLine() (line int)    { return r.Line }
func (r *ReadFile) Run(base string) (err error) {
	filepath := path.Clean(path.Join(base, r.File))
	contents, err := os.ReadFile(filepath)
	if err != nil {
		return fmt.Errorf("failed to read file contents: %w", err)
	}

	expect := strings.Join(r.Contents, " ")
	contentsS := string(contents)
	if expect != contentsS {
		return fmt.Errorf("contents doesn't match: expected(%s) != received(%s)", expect, contentsS)
	}
	return nil
}

type Permission struct {
	Line int
	Path string
	Mode fs.FileMode
}

func (p *Permission) GetName() (name string) { return "perm" }
func (p *Permission) GetLine() (line int)    { return p.Line }
func (p *Permission) Run(base string) (err error) {
	filepath := path.Clean(path.Join(base, p.Path))

	info, err := os.Stat(filepath)
	if err != nil {
		return fmt.Errorf("failed to get file stat: %w", err)
	}

	if p.Mode&0o7777 != info.Mode()&0o7777 {
		return fmt.Errorf("file permissions doesn't match: expect(%o) != received(%o)", p.Mode&0o7777, info.Mode()&0o7777)
	}
	return nil
}

type Type struct {
	Line int
	Path string
	Mode fs.FileMode
}

func (t *Type) GetName() (name string) { return "type" }
func (t *Type) GetLine() (line int)    { return t.Line }
func (t *Type) Run(base string) (err error) {
	filepath := path.Clean(path.Join(base, t.Path))

	info, err := os.Stat(filepath)
	if err != nil {
		return fmt.Errorf("failed to get file stat: %w", err)
	}

	if t.Mode.Type() != info.Mode().Type() {
		return fmt.Errorf("file type doesn't match: expect(%s) != received(%s)", t.Mode.Type(), info.Mode().Type())
	}
	return nil
}

type Remove struct {
	Line int
	Path string
}

func (r *Remove) GetName() (name string) { return "rm" }
func (r *Remove) GetLine() (line int)    { return r.Line }
func (r *Remove) Run(base string) (err error) {
	fullpath := path.Clean(path.Join(base, r.Path))

	err = os.Remove(fullpath)
	if err != nil {
		return fmt.Errorf("failed to remove path: %w", err)
	}
	return nil
}

type Rename struct {
	Line             int
	Oldname, Newname string
}

func (r *Rename) GetName() (name string) { return "mv" }
func (r *Rename) GetLine() (line int)    { return r.Line }
func (r *Rename) Run(base string) (err error) {
	oldfullpath := path.Clean(path.Join(base, r.Oldname))
	newfullpath := path.Clean(path.Join(base, r.Newname))

	err = os.Rename(oldfullpath, newfullpath)
	if err != nil {
		return fmt.Errorf("failed to rename oldname to newname: %w", err)
	}
	return nil
}

type Chmod struct {
	Line int
	Path string
	Mode fs.FileMode
}

func (c *Chmod) GetName() (name string) { return "chmod" }
func (c *Chmod) GetLine() (line int)    { return c.Line }
func (c *Chmod) Run(base string) (err error) {
	filepath := path.Clean(path.Join(base, c.Path))

	err = os.Chmod(filepath, c.Mode)
	if err != nil {
		return fmt.Errorf("failed to chmod path: %w", err)
	}
	return nil
}
