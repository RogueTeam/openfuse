//go:build openbsd

package fsscript

import (
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"slices"
	"strconv"
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

type ReaddirCount struct {
	Line      int
	Directory string
	Count     int64
}

func (r *ReaddirCount) GetName() (name string) { return "lscount" }
func (r *ReaddirCount) GetLine() (line int)    { return r.Line }
func (r *ReaddirCount) Run(base string) (err error) {
	dir := path.Clean(path.Join(base, r.Directory))

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	if int64(len(entries)) > r.Count {
		return fmt.Errorf("expecting %d but received %d", r.Count, len(entries))
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

type Size struct {
	Line int
	Path string
	Size int64
}

func (s *Size) GetName() (name string) { return "size" }
func (s *Size) GetLine() (line int)    { return s.Line }
func (s *Size) Run(base string) (err error) {
	filepath := path.Clean(path.Join(base, s.Path))

	info, err := os.Stat(filepath)
	if err != nil {
		return fmt.Errorf("failed to get file stat: %w", err)
	}

	if s.Size != info.Size() {
		return fmt.Errorf("file size doesn't match: expect(%d) != received(%d)", s.Size, info.Size())
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

type RemoveCount struct {
	Line  int
	Path  string
	Count int64
}

func (r *RemoveCount) GetName() (name string) { return "rmcount" }
func (r *RemoveCount) GetLine() (line int)    { return r.Line }
func (r *RemoveCount) Run(base string) (err error) {
	fullpath := path.Clean(path.Join(base, r.Path))

	entries, err := os.ReadDir(fullpath)
	if err != nil {
		return fmt.Errorf("failed to read location: %w", err)
	}

	if int64(len(entries)) < r.Count {
		return fmt.Errorf("not enough entries to remove: expect(%d) != received(%d)", r.Count, len(entries))
	}
	for _, entry := range entries[:r.Count] {
		err = os.Remove(path.Join(fullpath, entry.Name()))
		if err != nil {
			return fmt.Errorf("failed to remove path: %w", err)
		}
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

type AppendFile struct {
	Line     int
	File     string
	Contents []string
}

func (a *AppendFile) GetName() (name string) { return "append" }
func (a *AppendFile) GetLine() (line int)    { return a.Line }
func (a *AppendFile) Run(base string) (err error) {
	filepath := path.Clean(path.Join(base, a.File))

	file, err := os.OpenFile(filepath, os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	_, err = file.WriteString(strings.Join(a.Contents, " "))
	if err != nil {
		return fmt.Errorf("failed to write contents: %w", err)
	}

	return nil
}

type TruncateFile struct {
	Line int
	File string
	Size int64
}

func (t *TruncateFile) GetName() (name string) { return "truncate" }
func (t *TruncateFile) GetLine() (line int)    { return t.Line }
func (t *TruncateFile) Run(base string) (err error) {
	filepath := path.Clean(path.Join(base, t.File))

	file, err := os.OpenFile(filepath, os.O_WRONLY, 0)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	err = file.Truncate(t.Size)
	if err != nil {
		return fmt.Errorf("failed to truncate file: %w", err)
	}

	return nil
}

type WriteAtFile struct {
	Line     int
	File     string
	Offset   int64
	Contents []string
}

func (w *WriteAtFile) GetName() (name string) { return "writeat" }
func (w *WriteAtFile) GetLine() (line int)    { return w.Line }
func (w *WriteAtFile) Run(base string) (err error) {
	filepath := path.Clean(path.Join(base, w.File))

	file, err := os.OpenFile(filepath, os.O_WRONLY, 0)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	contentsS := strings.Join(w.Contents, " ")
	_, err = file.WriteAt([]byte(contentsS), w.Offset)
	if err != nil {
		return fmt.Errorf("failed to writeat file: %w", err)
	}

	return nil
}

type RandomFile struct {
	Line int
	File string
	Size int64
}

func (r *RandomFile) GetName() (name string) { return "randomfile" }
func (r *RandomFile) GetLine() (line int)    { return r.Line }
func (r *RandomFile) Run(base string) (err error) {
	filepath := path.Clean(path.Join(base, r.File))

	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	_, err = io.Copy(file, io.LimitReader(rand.Reader, r.Size))
	if err != nil {
		return fmt.Errorf("failed to fill with random data: %w", err)
	}

	return nil
}

type ReadAtFile struct {
	Line     int
	File     string
	Offset   int64
	Size     int64
	Contents []string
}

func (r *ReadAtFile) GetName() (name string) { return "readat" }
func (r *ReadAtFile) GetLine() (line int)    { return r.Line }
func (r *ReadAtFile) Run(base string) (err error) {
	filepath := path.Clean(path.Join(base, r.File))

	file, err := os.OpenFile(filepath, os.O_RDONLY, 0)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	b := make([]byte, r.Size)
	n, err := file.ReadAt(b, r.Offset)
	if err != nil {
		if n == 0 || !errors.Is(err, io.EOF) {
			return fmt.Errorf("failed to read contents at offset: %d: %w", r.Offset, err)
		}
	}

	expect := []byte(strings.Join(r.Contents, " "))
	received := b[:n]
	if !bytes.Equal(expect, received) {
		return fmt.Errorf("contents doesn't match: expect(%s) != received(%s)", expect, received)
	}

	return nil
}

type TouchnFile struct {
	Line       int
	Prefix     string
	Permission fs.FileMode
	Count      int64
}

func (t *TouchnFile) GetName() (name string) { return "touchn" }
func (t *TouchnFile) GetLine() (line int)    { return t.Line }
func (t *TouchnFile) Run(base string) (err error) {
	prefix := path.Clean(path.Join(base, t.Prefix))
	for n := range t.Count {
		filename := prefix + strconv.FormatInt(n, 10)

		err := os.WriteFile(filename, []byte(filename), t.Permission)
		if err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}
	}
	return nil
}
