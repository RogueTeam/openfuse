# zip

Mounts a zip archive read-only, so you can browse it without extracting
it. `archive/zip` already implements `fs.FS`, so the reader is handed
straight to `fshandler.New`.

Files in a zip can't be read from the middle, so the first read on an
open file decompresses the whole file into memory. Very large files
inside the archive will use a lot of memory.

## Run

Mounting needs root, so build first and run the binary with doas:

    $ go build
    $ mkdir -p /tmp/zip
    $ doas ./zip ~/some-archive.zip /tmp/zip

If you don't have a zip file around, make one (`pkg_add zip` if needed):

    $ cd /usr/share/misc && zip -r ~/misc.zip . && cd -
    $ doas ./zip ~/misc.zip /tmp/zip

From another terminal:

    $ doas ls -l /tmp/zip
    $ doas cat /tmp/zip/airport

Press Ctrl-C to stop it. That also unmounts `/tmp/zip`.
