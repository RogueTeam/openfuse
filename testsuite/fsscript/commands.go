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

type Command interface {
	GetName() (name string)
	GetLine() (line int)
	Run(ctx *CommandContext, base string) (err error)
}

type CommandContext struct {
	Handles map[string]fs.File
}

func (c *CommandContext) Close() (err error) {
	for name, h := range c.Handles {
		h.Close()
		delete(c.Handles, name)
	}
	return nil
}

func NewContext() (ctx *CommandContext) {
	return &CommandContext{
		Handles: make(map[string]fs.File),
	}
}

type Mkdir struct {
	Line       int
	Directory  string
	Permission fs.FileMode
}

func (m *Mkdir) GetName() (name string) { return "mkdir" }
func (m *Mkdir) GetLine() (line int)    { return m.Line }
func (m *Mkdir) Run(ctx *CommandContext, base string) (err error) {
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
func (r *Readdir) Run(ctx *CommandContext, base string) (err error) {
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
func (r *ReaddirCount) Run(ctx *CommandContext, base string) (err error) {
	dir := path.Clean(path.Join(base, r.Directory))

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	if int64(len(entries)) != r.Count {
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
func (c *CreateFile) Run(ctx *CommandContext, base string) (err error) {
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
func (r *ReadFile) Run(ctx *CommandContext, base string) (err error) {
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
func (p *Permission) Run(ctx *CommandContext, base string) (err error) {
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
func (t *Type) Run(ctx *CommandContext, base string) (err error) {
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
func (s *Size) Run(ctx *CommandContext, base string) (err error) {
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
func (r *Remove) Run(ctx *CommandContext, base string) (err error) {
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
func (r *RemoveCount) Run(ctx *CommandContext, base string) (err error) {
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
func (r *Rename) Run(ctx *CommandContext, base string) (err error) {
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
func (c *Chmod) Run(ctx *CommandContext, base string) (err error) {
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
func (a *AppendFile) Run(ctx *CommandContext, base string) (err error) {
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
func (t *TruncateFile) Run(ctx *CommandContext, base string) (err error) {
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
func (w *WriteAtFile) Run(ctx *CommandContext, base string) (err error) {
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
func (r *RandomFile) Run(ctx *CommandContext, base string) (err error) {
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
func (r *ReadAtFile) Run(ctx *CommandContext, base string) (err error) {
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
func (t *TouchnFile) Run(ctx *CommandContext, base string) (err error) {
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

type Fail struct {
	Command Command
}

func (f *Fail) GetName() (name string) { return "! " + f.Command.GetName() }
func (f *Fail) GetLine() (line int)    { return f.Command.GetLine() }
func (f *Fail) Run(ctx *CommandContext, base string) (err error) {
	err = f.Command.Run(ctx, base)
	if err == nil {
		return fmt.Errorf("expecting a failure")
	}
	return nil
}

type Open struct {
	Line       int
	Path       string
	Name       string
	Flag       int
	Permission fs.FileMode
}

func (o *Open) GetName() (name string) { return "open" }
func (o *Open) GetLine() (line int)    { return o.Line }
func (o *Open) Run(ctx *CommandContext, base string) (err error) {
	filepath := path.Clean(path.Join(base, o.Path))

	if _, found := ctx.Handles[o.Name]; found {
		return fmt.Errorf("name already in use: %s", o.Name)
	}

	file, err := os.OpenFile(filepath, o.Flag, o.Permission)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	ctx.Handles[o.Name] = file
	return nil
}

type Close struct {
	Line int
	Name string
}

func (c *Close) GetName() (name string) { return "close" }
func (c *Close) GetLine() (line int)    { return c.Line }
func (c *Close) Run(ctx *CommandContext, base string) (err error) {
	if handle, found := ctx.Handles[c.Name]; found {
		return handle.Close()

	}

	return fmt.Errorf("name is not in use: %s", c.Name)
}

type ReadAll struct {
	Line     int
	Name     string
	Contents []string
}

func (r *ReadAll) GetName() (name string) { return "readall" }
func (r *ReadAll) GetLine() (line int)    { return r.Line }
func (r *ReadAll) Run(ctx *CommandContext, base string) (err error) {
	if handle, found := ctx.Handles[r.Name]; found {
		contents, err := io.ReadAll(handle)
		if err != nil {
			return fmt.Errorf("failed to read contents: %w", err)
		}

		contentsS := string(contents)
		expect := strings.Join(r.Contents, " ")
		if expect != contentsS {
			return fmt.Errorf("contents doesn't match: expect(%s) != received(%s)", expect, contentsS)
		}
		return nil
	}
	return fmt.Errorf("name is not in use: %s", r.Name)
}

func NewCommand(line int, command string, args []string) (cmd Command, err error) {
	switch command {
	case "mkdir":
		if len(args) != 2 {
			return nil, fmt.Errorf("expecting: mkdir DIRECTORY PERMISSION")
		}

		directory := args[0]
		rawPermission := args[1]
		permission, err := strconv.ParseUint(rawPermission, 8, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse permission: %s: %w", rawPermission, err)
		}

		return &Mkdir{
			Line:       line,
			Directory:  directory,
			Permission: fs.FileMode(permission),
		}, nil
	case "ls":
		if len(args) < 2 {
			return nil, fmt.Errorf("expecting: ls DIRECTORY EMPTY | (EXPECT [EXPECT [...]])")
		}

		directory := args[0]
		expect := args[1:]
		if strings.EqualFold(expect[0], "empty") {
			expect = nil
		}

		return &Readdir{
			Line:      line,
			Directory: directory,
			Expect:    expect,
		}, nil
	case "lscount":
		if len(args) != 2 {
			return nil, fmt.Errorf("expecting: lscount DIRECTORY COUNT")
		}

		directory := args[0]
		rawCount := args[1]
		count, err := strconv.ParseInt(rawCount, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("failed to parse count: %s: %w", rawCount, err)
		}

		return &ReaddirCount{
			Line:      line,
			Directory: directory,
			Count:     count,
		}, nil
	case "file":
		if len(args) < 3 {
			return nil, fmt.Errorf("expecting: file FILE PERMISSION CONTENTS")
		}

		filename := args[0]
		rawPermission := args[1]
		contents := args[2:]

		permission, err := strconv.ParseUint(rawPermission, 8, 32)
		if err != nil {
			return nil, fmt.Errorf("failed to parse permission: %s: %w", rawPermission, err)
		}

		if strings.EqualFold(contents[0], "empty") {
			contents = nil
		}

		return &CreateFile{
			Line:       line,
			File:       filename,
			Permission: fs.FileMode(permission),
			Contents:   contents,
		}, nil
	case "cat":
		if len(args) < 2 {
			return nil, fmt.Errorf("expecting: cat FILE CONTENTS")
		}

		filename := args[0]
		contents := args[1:]
		if strings.EqualFold(contents[0], "empty") {
			contents = nil
		}

		return &ReadFile{
			Line:     line,
			File:     filename,
			Contents: contents,
		}, nil
	case "perm":
		if len(args) != 2 {
			return nil, fmt.Errorf("expecting: perm PATH PERMISSION")
		}

		filepath := args[0]
		rawPermission := args[1]
		permission, err := strconv.ParseUint(rawPermission, 8, 32)
		if err != nil {
			return nil, fmt.Errorf("failed to parse permission: %s: %w", rawPermission, err)
		}

		return &Permission{
			Line: line,
			Path: filepath,
			Mode: fs.FileMode(permission),
		}, nil
	case "type":
		if len(args) != 2 {
			return nil, fmt.Errorf("expecting: type PATH TYPE")
		}

		filepath := args[0]
		rawType := args[1]
		filetype, err := FileTypeFromString(rawType)
		if err != nil {
			return nil, fmt.Errorf("failed to convert type: %w", err)
		}

		return &Type{
			Line: line,
			Path: filepath,
			Mode: filetype,
		}, nil
	case "size":
		if len(args) != 2 {
			return nil, fmt.Errorf("expecting: size PATH SIZE")
		}

		filepath := args[0]
		rawSize := args[1]
		size, err := strconv.ParseInt(rawSize, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse size: %s: %w", rawSize, err)
		}

		return &Size{
			Line: line,
			Path: filepath,
			Size: size,
		}, nil
	case "rm", "rmdir":
		if len(args) != 1 {
			return nil, fmt.Errorf("expecting: rm/rmdir PATH")
		}

		filepath := args[0]
		return &Remove{
			Line: line,
			Path: filepath,
		}, nil
	case "rmcount":
		if len(args) != 2 {
			return nil, fmt.Errorf("expecting: rmcount DIRPATH COUNT")
		}

		filepath := args[0]
		rawCount := args[1]
		count, err := strconv.ParseInt(rawCount, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse count: %s: %w", rawCount, err)
		}

		return &RemoveCount{
			Line:  line,
			Path:  filepath,
			Count: count,
		}, nil
	case "mv":
		if len(args) != 2 {
			return nil, fmt.Errorf("expecting: mv OLDNAME NEWNAME")
		}

		oldname := args[0]
		newname := args[1]
		return &Rename{
			Line:    line,
			Oldname: oldname,
			Newname: newname,
		}, nil
	case "chmod":
		if len(args) != 2 {
			return nil, fmt.Errorf("expecting: chmod PATH PERMISSION")
		}

		filepath := args[0]
		rawPermission := args[1]
		permission, err := strconv.ParseUint(rawPermission, 8, 32)
		if err != nil {
			return nil, fmt.Errorf("failed to parse permission: %s: %w", rawPermission, err)
		}
		return &Chmod{
			Line: line,
			Path: filepath,
			Mode: fs.FileMode(permission),
		}, nil
	case "append":
		if len(args) < 2 {
			return nil, fmt.Errorf("expecting: append FILE CONTENTS")
		}

		filename := args[0]
		contents := args[1:]

		return &AppendFile{
			Line:     line,
			File:     filename,
			Contents: contents,
		}, nil
	case "truncate":
		if len(args) != 2 {
			return nil, fmt.Errorf("expecting: truncate FILE SIZE")
		}

		filename := args[0]
		rawSize := args[1]
		size, err := strconv.ParseInt(rawSize, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse size: %w", err)
		}

		return &TruncateFile{
			Line: line,
			File: filename,
			Size: size,
		}, nil
	case "writeat":
		if len(args) < 3 {
			return nil, fmt.Errorf("expecting: writeat FILE OFFSET CONTENTS")
		}

		filename := args[0]
		rawOffset := args[1]
		contents := args[2:]
		offset, err := strconv.ParseInt(rawOffset, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse offset: %w", err)
		}

		return &WriteAtFile{
			Line:     line,
			File:     filename,
			Offset:   offset,
			Contents: contents,
		}, nil
	case "randomfile":
		if len(args) != 2 {
			return nil, fmt.Errorf("expecting: randomfile FILE SIZE")
		}

		filename := args[0]
		rawSize := args[1]
		size, err := strconv.ParseInt(rawSize, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse size: %w", err)
		}

		return &RandomFile{
			Line: line,
			File: filename,
			Size: size,
		}, nil
	case "readat":
		if len(args) < 3 {
			return nil, fmt.Errorf("expecting: readat FILE OFFSET SIZE CONTENTS")
		}

		filename := args[0]
		rawOffset := args[1]
		rawSize := args[2]
		contents := args[3:]

		offset, err := strconv.ParseInt(rawOffset, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse offset: %w", err)
		}

		size, err := strconv.ParseInt(rawSize, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse size: %w", err)
		}

		return &ReadAtFile{
			Line:     line,
			File:     filename,
			Offset:   offset,
			Size:     size,
			Contents: contents,
		}, nil
	case "touchn":
		if len(args) != 3 {
			return nil, fmt.Errorf("expecting: touchn PREFIX PERMISSION COUNT")
		}

		prefix := args[0]
		rawPermission := args[1]
		rawCount := args[2]

		permission, err := strconv.ParseUint(rawPermission, 8, 32)
		if err != nil {
			return nil, fmt.Errorf("failed to parse permission: %s: %w", rawPermission, err)
		}

		count, err := strconv.ParseInt(rawCount, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse count: %s: %w", rawCount, err)
		}

		return &TouchnFile{
			Line:       line,
			Prefix:     prefix,
			Permission: fs.FileMode(permission),
			Count:      count,
		}, nil
	case "open":
		if len(args) != 4 {
			return nil, fmt.Errorf("expecting: open PATH HANDLE_NAME FLAG PERM")
		}

		path := args[0]
		name := args[1]
		rawFlag := args[2]
		rawPermission := args[3]

		flag, err := OpenFlagFromString(rawFlag)
		if err != nil {
			return nil, fmt.Errorf("failed to parse open flag from string: %w", err)
		}

		permission, err := strconv.ParseUint(rawPermission, 8, 32)
		if err != nil {
			return nil, fmt.Errorf("failed to parse permission: %s: %w", rawPermission, err)
		}

		return &Open{
			Line:       line,
			Path:       path,
			Name:       name,
			Flag:       flag,
			Permission: fs.FileMode(permission),
		}, nil
	case "close":
		if len(args) != 1 {
			return nil, fmt.Errorf("expecting: close HANDLE_NAME")
		}

		name := args[0]

		return &Close{
			Line: line,
			Name: name,
		}, nil
	case "readall":
		if len(args) < 2 {
			return nil, fmt.Errorf("expecting: readall HANDLE_NAME CONTENTS")
		}

		name := args[0]
		contents := args[1:]
		if strings.EqualFold(contents[0], "empty") {
			contents = nil
		}

		return &ReadAll{
			Line:     line,
			Name:     name,
			Contents: contents,
		}, nil
	default:
		return nil, fmt.Errorf("unknown command: %s", command)
	}
}
