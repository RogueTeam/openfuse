//go:build openbsd

package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/RogueTeam/openfuse/v8/fuse"
	"github.com/RogueTeam/openfuse/v8/handlers/fshandler"
	"github.com/RogueTeam/openfuse/v8/memfs"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: %s MOUNTPOINT\n", os.Args[0])
		os.Exit(2)
	}

	f, err := fuse.Mount(os.Args[1], fuse.Options{Name: "memfs"}, fshandler.New(memfs.New()))
	if err != nil {
		log.Fatal(err)
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		<-sig
		if err := f.Close(); err != nil {
			log.Print(err)
		}
		os.Exit(0)
	}()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	err = f.Serve(context.Background(), logger)
	switch {
	case f.Finished.Load():
		// The signal handler is unmounting and will exit.
		select {}
	case errors.Is(err, fuse.ErrDestroyed):
		// Unmounted from outside with umount(8).
		return
	default:
		f.Close()
		log.Fatal(err)
	}
}
