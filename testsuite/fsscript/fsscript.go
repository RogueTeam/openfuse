//go:build openbsd

package fsscript

import (
	"embed"
	"fmt"
	"io/fs"
	"strings"
	"testing"
)

//go:embed scripts
var scripts embed.FS

type Test struct {
	Name     string
	Commands []Command
}

// Runs the tests with base as the base directory to construct the filesystem
func (t *Test) Run(ctx *CommandContext, base string) (err error) {
	for _, command := range t.Commands {
		err := command.Run(ctx, base)
		if err != nil {
			return fmt.Errorf("failed to run command: %s at line %d: %w", command.GetName(), command.GetLine(), err)
		}
	}
	return nil
}

func (t *Test) Append(line int, fail bool, command string, args []string) (err error) {
	cmd, err := NewCommand(line, command, args)
	if err != nil {
		return fmt.Errorf("failed to construct command: %w", err)
	}

	if fail {
		t.Commands = append(t.Commands, &Fail{Command: cmd})
	} else {
		t.Commands = append(t.Commands, cmd)
	}
	return nil
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

		var fail bool
		var command string
		var args []string
		if parts[0] == "!" {
			if len(parts[1:]) == 0 {
				return nil, fmt.Errorf("no command provided for failure mode at line: %d", 1+index)
			}
			fail = true
			command = parts[1]
			args = parts[2:]
		} else {
			command = parts[0]
			args = parts[1:]
		}

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
		err := curr.Append(1+index, fail, command, args)
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

type Group struct {
	File  string
	Tests []Test
}

func GenericScripts(tb testing.TB) (groups []Group) {
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
		tests, err := Parse(contents)
		if err != nil {
			return fmt.Errorf("failed to parse script: %s: %w", path, err)
		}

		groups = append(groups, Group{
			File:  path,
			Tests: tests,
		})
		return nil
	})
	if err != nil {
		tb.Fatalf("failed to walk filesystem: %v", err)
	}
	return groups
}
