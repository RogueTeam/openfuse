# openfuse

[Documentation](https://pkg.go.dev/github.com/RogueTeam/openfuse/v8)

> IMPORTANT: AI USAGE IS NOT ALLOWED IN THIS PROJECT

A pure Go, BSD Licensed, handcrafted fuse library so you can bring your FUSE ideas to the best operating system in the entire world.

openfuse talks to [fuse(4)](https://man.openbsd.org/fuse.4) directly. There is no cgo and no libfuse,
so a filesystem builds with `GOOS=openbsd go build` cross-compiles it from any machine.

Filesystems are plain `fs.FS` values. Anything that implements `fs.FS`
can be mounted: `embed.FS`, `os.DirFS`, a zip archive, or your own type.

## Requirements

- OpenBSD
- Go 1.27 or newer
- root, to mount

## Install

    go get github.com/RogueTeam/openfuse/v8

## A read/write filesystem in memory

`memfs` keeps everything in memory, and supports creating, writing,
renaming and deleting files and directories.

```go
package main

import (
	"context"
	"log"
	"log/slog"

	"github.com/RogueTeam/openfuse/v8/fuse"
	"github.com/RogueTeam/openfuse/v8/handlers/fshandler"
	"github.com/RogueTeam/openfuse/v8/memfs"
)

func main() {
	f, err := fuse.Mount("/mnt/memfs", fuse.Options{Name: "memfs"}, fshandler.New(memfs.New()))
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	err = f.Serve(context.Background(), slog.Default())
	if err != nil {
		log.Print(err)
	}
}
```

Build it and run it with `doas`. This is the shortest version: it
doesn't unmount when you press Ctrl-C. [examples/memfs](examples/memfs)
does.

## Read-only filesystems

Mounting something else only changes the `fs.FS` you pass in.

Files compiled into the binary:

```go
//go:embed static
var static embed.FS

f, err := fuse.Mount("/mnt/static", fuse.Options{Name: "static", ReadOnly: true}, fshandler.New(static))
```

A zip archive, without extracting it:

```go
archive, err := zip.OpenReader("backup.zip")
if err != nil {
	log.Fatal(err)
}
defer archive.Close()

f, err := fuse.Mount("/mnt/backup", fuse.Options{Name: "backup", ReadOnly: true}, fshandler.New(archive))
```

## Writing your own

A filesystem only has to implement `fs.FS` to be mounted read-only.
To support writes, it implements the optional interfaces in
`handlers/fshandler`: `MkdirFS`, `RenameFS`, `UnlinkFS` and the rest,
or all of them at once as `FullFS`. Its files implement `FullFile`,
and its directories `FullDir`. The `memfs` package is a complete
example.

## Examples

[examples](examples) has small programs you can build and run:

- [hello](examples/hello): two files made up in code
- [memfs](examples/memfs): read/write, in memory
- [mirror](examples/mirror): an existing directory, through `os.DirFS`
- [embed](examples/embed): files compiled into the binary
- [zip](examples/zip): a zip archive

Each one has its own `go.mod`, so they don't add anything to this module.

## Tests

The tests mount real filesystems, so they need root:

    doas go test ./...

Most of the filesystem behaviour is tested with short scripts in
`testsuite/fsscript/scripts`. The same scripts run against a normal
directory and against a memfs mount, so both have to agree.

## Proverbs for this project

- Less dependencies means less things to worry about.
- `golang.org/x` dependencies count as no dependencies.
- Being correct is more important than being fast.

## Versioning

- Version numbers are always in sync to the latest OpenBSD version.

## License

BSD 3-Clause. See [LICENSE](LICENSE).
