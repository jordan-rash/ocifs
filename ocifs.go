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

	built bool
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
		built:     false,
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
	if !r.built {
		return errors.New("rootfs is not built yet. Call Build() first")
	}
	r.logger.Info("Creating rootfs")
	err := r.makeRootFS()
	if err != nil {
		return err
	}
	return nil
}

// Build will download and extract the layers and stop.
// You can then add what you need to the buildDir
func (r *rootFS) Build() error {
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

	r.built = true
	return nil
}

// Adds a file to the rootfs.
// src should be a local file, dest should be the absolute path in the rootfs
func (r *rootFS) AddFile(src, dest string) error {
	r.logger.Info("Adding file to rootfs", slog.String("local", src), slog.String("dest", dest))
	if _, err := os.Stat(src); os.IsNotExist(err) {
		return fmt.Errorf("file %s does not exist", src)
	}
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", src, err)
	}
	defer srcFile.Close()

	mvDest := filepath.Join(r.buildDir, filepath.Dir(dest))
	err = os.MkdirAll(mvDest, 0755)
	if err != nil {
		return fmt.Errorf("failed to create directory for file %s: %w", mvDest, err)
	}

	rfsFileName := filepath.Base(dest)
	if rfsFileName == "." {
		rfsFileName = filepath.Base(src)
	}

	dstFile, err := os.Create(filepath.Join(mvDest, rfsFileName))
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}

	return nil
}
