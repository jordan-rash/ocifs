package main

import (
	"io"
	"log/slog"
	rootfs "ocilayers"
	"runtime"

	"disorder.dev/shandler"
	"github.com/alecthomas/kong"
)

type CLI struct {
	OciRef  string `arg:"" name:"oci-ref" help:"OCI image reference"`
	Verbose int    `short:"v" type:"counter" help:"Enable verbose mode"`
}

func (c CLI) Run(logger *slog.Logger) error {
	libLogger := logger.WithGroup("rootfs")

	rfs, err := rootfs.NewRootFS(c.OciRef,
		rootfs.WithLogger(libLogger),
		rootfs.WithOSArch(runtime.GOOS, runtime.GOARCH),
	)
	if err != nil {
		return err
	}

	err = rfs.Create()
	if err != nil {
		return err
	}

	return nil
}

func main() {
	cli := new(CLI)
	ctx := kong.Parse(cli)
	opts := []shandler.HandlerOption{
		shandler.WithColor(),
		shandler.WithShortLevels(),
	}

	switch cli.Verbose {
	case 0:
		opts = append(opts, shandler.WithStdOut(io.Discard), shandler.WithStdErr(io.Discard))
	case 1:
		opts = append(opts, shandler.WithLogLevel(slog.LevelError))
	case 2:
		opts = append(opts, shandler.WithLogLevel(slog.LevelInfo))
	case 3:
		opts = append(opts, shandler.WithLogLevel(slog.LevelDebug))
	default:
		opts = append(opts, shandler.WithLogLevel(shandler.LevelTrace))
	}

	logger := slog.New(shandler.NewHandler(opts...))

	err := ctx.Run(logger)
	ctx.FatalIfErrorf(err)
}
