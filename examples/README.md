# Examples

Each directory is a small program with its own `go.mod`, so the
examples never touch the dependencies of the main module. The `replace`
line in each `go.mod` points at this repository, so the examples always
build against your local copy.

| Example | What it mounts | Writable |
|---|---|---|
| [hello](hello) | Two files built in code with `fstest.MapFS` | no |
| [memfs](memfs) | An empty filesystem kept in memory | yes |
| [mirror](mirror) | An existing directory, through `os.DirFS` | no |
| [embed](embed) | Files compiled into the binary with `//go:embed` | no |
| [zip](zip) | The contents of a zip archive | no |

All of them run only on OpenBSD and need root to mount. Ctrl-C stops
the program and unmounts the directory.
