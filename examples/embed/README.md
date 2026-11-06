# embed

Serves the `files/` directory, which is built into the binary with
`//go:embed`. The binary can be copied anywhere and still mount the same
files.

`fs.Sub` drops the `files/` prefix, so its contents appear at the root
of the mount.

## Run

Mounting needs root, so build first and run the binary with doas:

    $ go build
    $ mkdir -p /tmp/embed
    $ doas ./embed /tmp/embed

From another terminal:

    $ doas ls -R /tmp/embed
    README.txt notes

    /tmp/embed/notes:
    todo.txt
    $ doas cat /tmp/embed/notes/todo.txt

Press Ctrl-C to stop it. That also unmounts `/tmp/embed`.

To serve different files, change what is in `files/` and run `go build`
again.
