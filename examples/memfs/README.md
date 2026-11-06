# memfs

A read/write filesystem that lives in memory, using the `memfs` package
from this repository. You can create, edit, rename and delete files and
directories. Everything is lost when the program exits.

## Run

Mounting needs root, so build first and run the binary with doas:

    $ go build
    $ mkdir -p /tmp/memfs
    $ doas ./memfs /tmp/memfs

From another terminal, as root:

    # cd /tmp/memfs
    # mkdir notes
    # echo "buy milk" > notes/todo.txt
    # mv notes/todo.txt notes/today.txt
    # cat notes/today.txt
    buy milk
    # ls -la notes

Press Ctrl-C to stop it. That also unmounts `/tmp/memfs`. Leave the
directory first, or the unmount will fail because it is busy.
