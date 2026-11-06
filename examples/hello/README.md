# hello

The smallest example. Two read-only files kept in a `fstest.MapFS`:

- `hello.txt`
- `mounted-at.txt`, with the time the filesystem was mounted

Any `fs.FS` can be mounted the same way, through `fshandler.New`.

## Run

Mounting needs root, so build first and run the binary with doas:

    $ go build
    $ mkdir -p /tmp/hello
    $ doas ./hello /tmp/hello

From another terminal:

    $ doas ls -l /tmp/hello
    $ doas cat /tmp/hello/hello.txt
    Hello from openfuse!

Press Ctrl-C to stop it. That also unmounts `/tmp/hello`.
