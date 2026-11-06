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
	"testing/fstest"
	"time"

	"github.com/RogueTeam/openfuse/v8/fuse"
	"github.com/RogueTeam/openfuse/v8/handlers/fshandler"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: %s MOUNTPOINT\n", os.Args[0])
		os.Exit(2)
	}

	now := time.Now()
	files := fstest.MapFS{
		"hello.txt": {
			Data:    []byte("Hello from openfuse!\n"),
			Mode:    0o444,
			ModTime: now,
		},
		"mounted-at.txt": {
			Data:    []byte(now.Format(time.RFC1123) + "\n"),
			Mode:    0o444,
			ModTime: now,
		},
	}

	f, err := fuse.Mount(os.Args[1], fuse.Options{Name: "hello", ReadOnly: true}, fshandler.New(files))
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
