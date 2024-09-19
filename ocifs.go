package ocifs

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

type rootFS struct {
	logger *slog.Logger

	ociRef *Reference
	os     string
	arch   string

	outputDir string
	buildDir  string
}

type RootFSOption func(*rootFS) error

func NewRootFS(ociRef string, opts ...RootFSOption) (*rootFS, error) {
	if ociRef == "" {
		return nil, errors.New("ociIn cannot be empty")
	}

	rootfs := &rootFS{
		logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
		ociRef:    &Reference{RawRef: ociRef},
		outputDir: ".",
		os:        "linux",
		arch:      "amd64",
		buildDir:  filepath.Join(os.TempDir(), "rootfs-*"),
	}
	defer os.RemoveAll(rootfs.buildDir)

	var optErr error
	for _, opt := range opts {
		optErr = errors.Join(optErr, opt(rootfs))
	}
	if optErr != nil {
		return nil, fmt.Errorf("failed to apply options: %w", optErr)
	}

	err := rootfs.Validate()
	if err != nil {
		return nil, err
	}

	return rootfs, nil
}

func (r rootFS) Validate() error {
	var errs error

	return errs
}

func (r *rootFS) Create() error {
	r.logger.Info("Parsing OCI Reference")
	err := r.parseOCIRef()
	if err != nil {
		return err
	}

	r.logger.Info("Downloading and extracting layers")
	err = r.downloadExtractLayers()
	if err != nil {
		return err
	}

	r.logger.Info("Creating rootfs")
	err = r.makeRootFS()
	if err != nil {
		return err
	}
	return nil
}
