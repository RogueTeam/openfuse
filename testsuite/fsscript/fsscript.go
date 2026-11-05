//go:build openbsd

package fsscript

import (
	"embed"
	"fmt"
	"io/fs"
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
	case "perm":
		if len(args) != 2 {
			return fmt.Errorf("expecting: perm PATH PERMISSION")
		}

		filepath := args[0]
		rawPermission := args[1]
		permission, err := strconv.ParseUint(rawPermission, 8, 32)
		if err != nil {
			return fmt.Errorf("failed to parse permission: %s: %w", rawPermission, err)
		}

		t.Commands = append(t.Commands, &Permission{
			Line: line,
			Path: filepath,
			Mode: fs.FileMode(permission),
		})
		return nil
	case "type":
		if len(args) != 2 {
			return fmt.Errorf("expecting: type PATH TYPE")
		}

		filepath := args[0]
		rawType := args[1]
		filetype, err := FileTypeFromString(rawType)
		if err != nil {
			return fmt.Errorf("failed to convert type: %w", err)
		}

		t.Commands = append(t.Commands, &Type{
			Line: line,
			Path: filepath,
			Mode: filetype,
		})
		return nil
	case "rm", "rmdir":
		if len(args) != 1 {
			return fmt.Errorf("expecting: rm/rmdir PATH")
		}

		filepath := args[0]
		t.Commands = append(t.Commands, &Remove{
			Line: line,
			Path: filepath,
		})
		return nil
	case "mv":
		if len(args) != 2 {
			return fmt.Errorf("expecting: mv OLDNAME NEWNAME")
		}

		oldname := args[0]
		newname := args[1]
		t.Commands = append(t.Commands, &Rename{
			Line:    line,
			Oldname: oldname,
			Newname: newname,
		})
		return nil
	case "chmod":
		if len(args) != 2 {
			return fmt.Errorf("expecting: chmod PATH PERMISSION")
		}

		filepath := args[0]
		rawPermission := args[1]
		permission, err := strconv.ParseUint(rawPermission, 8, 32)
		if err != nil {
			return fmt.Errorf("failed to parse permission: %s: %w", rawPermission, err)
		}
		t.Commands = append(t.Commands, &Chmod{
			Line: line,
			Path: filepath,
			Mode: fs.FileMode(permission),
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

		if curr.Name == "" {
			return nil, fmt.Errorf("commands require a namespace: %d", 1+index)
		}
		err := curr.Append(1+index, command, args)
		if err != nil {
			return nil, fmt.Errorf("failed to parse line: %d: for command '%s' and args: '%s': %w", 1+index, command, strings.Join(args, " "), err)
		}
	}

	if curr.Name != "" {
		tests = append(tests, curr)
	}

	if len(tests) == 0 {
		return nil, fmt.Errorf("no tests parsed")
	}
	return tests, nil
}

func GenericScripts(tb testing.TB) (tests []Test) {
	err := fs.WalkDir(scripts, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walk error: %w", err)
		}
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
