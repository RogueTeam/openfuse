//go:build openbsd

package fsscript

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path"
	"slices"
	"strconv"
	"strings"
	"testing"
)

//go:embed scripts
var scripts embed.FS

type Command interface {
	GetName() (name string)
	GetLine() (line int)
	Run(base string) (err error)
}

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

type Info struct {
	Line int
	Path string
	Mode fs.FileMode
}

func (i *Info) GetName() (name string) { return "info" }
func (i *Info) GetLine() (line int)    { return i.Line }
func (i *Info) Run(base string) (err error) {
	filepath := path.Clean(path.Join(base, i.Path))

	info, err := os.Stat(filepath)
	if err != nil {
		return fmt.Errorf("failed to get file stat: %w", err)
	}

	if i.Mode.Type() != info.Mode().Type() {
		return fmt.Errorf("file type doesn't match: expect(%s) != received(%s)", i.Mode.Type(), info.Mode().Type())
	}

	if i.Mode&0o7777 != info.Mode()&0o7777 {
		return fmt.Errorf("file permissions doesn't match: expect(%o) != received(%o)", i.Mode&0o7777, info.Mode()&0o7777)
	}
	return nil
}

type Test struct {
	Name     string
	Commands []Command
}

// Runs the tests with base as the base directory to construct the filesystem
func (t *Test) Run(base string) (err error) {
	for _, command := range t.Commands {
		err := command.Run(base)
		if err != nil {
			return fmt.Errorf("failed to run command: %s at line %d: %w", command.GetName(), command.GetLine(), err)
		}
	}
	return nil
}

func FileTypeFromString(s string) (mode fs.FileMode, err error) {
	switch s {
	case "dir":
		return fs.ModeDir, nil
	case "file":
		return 0, nil
	default:
		return 0, fmt.Errorf("unknown file type: %s", s)
	}
}

func (t *Test) Append(line int, command string, args []string) (err error) {
	switch command {
	case "mkdir":
		if len(args) != 2 {
			return fmt.Errorf("expecting: mkdir DIRECTORY PERMISSION")
		}

		directory := args[0]
		rawPermission := args[1]
		permission, err := strconv.ParseUint(rawPermission, 8, 32)
		if err != nil {
			return fmt.Errorf("failed to parse permission: %s: %w", rawPermission, err)
		}

		t.Commands = append(t.Commands, &Mkdir{
			Line:       line,
			Directory:  directory,
			Permission: fs.FileMode(permission),
		})
		return nil
	case "ls":
		if len(args) < 2 {
			return fmt.Errorf("expecting: ls DIRECTORY EMPTY | (EXPECT [EXPECT [...]])")
		}

		directory := args[0]
		expect := args[1:]
		if strings.EqualFold(expect[0], "empty") {
			expect = nil
		}

		t.Commands = append(t.Commands, &Readdir{
			Line:      line,
			Directory: directory,
			Expect:    expect,
		})
		return nil
	case "file":
		if len(args) < 3 {
			return fmt.Errorf("expecting: file FILE PERMISSION CONTENTS")
		}

		filename := args[0]
		rawPermission := args[1]
		contents := args[2:]
		permission, err := strconv.ParseUint(rawPermission, 8, 32)
		if err != nil {
			return fmt.Errorf("failed to parse permission: %s: %w", rawPermission, err)
		}

		t.Commands = append(t.Commands, &CreateFile{
			Line:       line,
			File:       filename,
			Permission: fs.FileMode(permission),
			Contents:   contents,
		})
		return nil
	case "cat":
		if len(args) < 2 {
			return fmt.Errorf("expecting: cat FILE CONTENTS")
		}

		filename := args[0]
		contents := args[1:]

		t.Commands = append(t.Commands, &ReadFile{
			Line:     line,
			File:     filename,
			Contents: contents,
		})
		return nil
	case "info":
		if len(args) != 3 {
			return fmt.Errorf("expecting: info PATH PERMISSION TYPE")
		}

		filepath := args[0]
		rawPermission := args[1]
		rawType := args[2]
		permission, err := strconv.ParseUint(rawPermission, 8, 32)
		if err != nil {
			return fmt.Errorf("failed to parse permission: %s: %w", rawPermission, err)
		}

		filetype, err := FileTypeFromString(rawType)
		if err != nil {
			return fmt.Errorf("failed to convert type: %w", err)
		}

		t.Commands = append(t.Commands, &Info{
			Line: line,
			Path: filepath,
			Mode: fs.FileMode(permission) | filetype,
		})
		return nil
	default:
		return fmt.Errorf("unknown command: %s", command)
	}
}

func Parse[T ~string | ~[]byte](s T) (tests []Test, err error) {
	var curr Test
	for index, line := range strings.Split(string(s), "\n") {
		commentIdx := strings.IndexByte(line, '#')
		if commentIdx != -1 {
			line = line[:commentIdx]
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}

		command := parts[0]
		args := parts[1:]

		if command == "=" {
			if curr.Name != "" {
				tests = append(tests, curr)
			}
			curr = Test{
				Name: strings.Join(args, " "),
			}
			continue
		}

		err := curr.Append(1+index, command, args)
		if err != nil {
			return nil, fmt.Errorf("failed to parse line: %d: for command '%s' and args: '%s'", 1+index, command, strings.Join(args, " "))
		}
	}

	if curr.Name != "" {
		tests = append(tests, curr)
	}
	return tests, nil
}

func GenericScripts(tb testing.TB) (tests []Test) {
	err := fs.WalkDir(scripts, ".", func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}
		contents, err := scripts.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read file: %s: %w", path, err)
		}
		newTests, err := Parse(contents)
		if err != nil {
			return fmt.Errorf("failed to parse script: %s: %w", path, err)
		}

		tests = append(tests, newTests...)
		return nil
	})
	if err != nil {
		tb.Fatalf("failed to walk filesystem: %v", err)
	}
	return tests
}
