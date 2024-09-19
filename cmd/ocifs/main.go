package main

import (
	"log/slog"
	"runtime"

	"disorder.dev/ocifs"
	"disorder.dev/shandler"
	"github.com/alecthomas/kong"
)

type CLI struct {
	OciRef  string            `arg:"" name:"oci-ref" help:"OCI image reference"`
	Verbose int               `short:"v" type:"counter" help:"Enable verbose mode"`
	Files   map[string]string `short:"f" placeholder:"src=dest;..." help:"Add a file to the rootfs"`
}

func (c CLI) Run(logger *slog.Logger) error {
	libLogger := logger.WithGroup("rootfs")

	rfs, err := ocifs.NewRootFS(c.OciRef,
		ocifs.WithLogger(libLogger),
		ocifs.WithOSArch(runtime.GOOS, runtime.GOARCH),
	)
	if err != nil {
		return err
	}

	err = rfs.Build()
	if err != nil {
		return err
	}

	for src, dst := range c.Files {
		err = rfs.AddFile(src, dst)
		if err != nil {
			return err
		}
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
		opts = append(opts, shandler.WithLogLevel(shandler.LevelFatal))
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
