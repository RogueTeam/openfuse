# mirror

Mounts an existing directory somewhere else, read-only. The files come
from `os.DirFS`, so nothing in this example knows about FUSE except the
mount itself.

Changes made in the source directory show up in the mount. Writes
through the mount are refused.

Symlinks are followed, so they show up as the file or directory they
point to.

## Run

Mounting needs root, so build first and run the binary with doas:

    $ go build
    $ mkdir -p /tmp/mirror
    $ doas ./mirror /etc /tmp/mirror

From another terminal:

    $ doas ls /tmp/mirror
    $ doas cat /tmp/mirror/myname
    $ doas touch /tmp/mirror/newfile
    touch: /tmp/mirror/newfile: Read-only file system

Press Ctrl-C to stop it. That also unmounts `/tmp/mirror`.
