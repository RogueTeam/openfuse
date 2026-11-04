//go:build openbsd

package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"

	"github.com/RogueTeam/openfuse/fuse"
	"github.com/RogueTeam/openfuse/handlers/fshandler"
	"github.com/RogueTeam/openfuse/memfs"
	"golang.org/x/sys/unix"
)

func main() {
	i, err := fuse.Mount(os.Args[1], fuse.Options{
		Name: "memfs",
	}, fshandler.New(memfs.New()))
	if err != nil {
		log.Fatal(err)
	}
	defer i.Close()

	ctx, cancel := signal.NotifyContext(context.TODO(), unix.SIGHUP, unix.SIGINT, unix.SIGTERM, unix.SIGKILL)
	defer cancel()
	err = i.Serve(ctx, slog.Default())
	if err != nil {
		log.Fatal(err)
	}
}
