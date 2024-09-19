package ocifs

import (
	"log/slog"
	"os"
	"testing"
)

func TestOCIParse(t *testing.T) {
	var tests = []struct {
		RawRef     string
		Reference  string
		Registry   string
		Repository string
	}{
		{"docker.io/library/ubuntu:latest", "latest", "docker.io", "library/ubuntu"},
		{"docker.io/synadia/nex-rootfs:alpine", "alpine", "docker.io", "synadia/nex-rootfs"},
		{"ghcr.io/actions/setup-go:5.0.2", "5.0.2", "ghcr.io", "actions/setup-go"},
	}

	for i, test := range tests {
		t.Logf("[%d/%d] Testing %s...", i+1, len(tests), test.RawRef)
		tempDir := t.TempDir()
		rfs := &rootFS{
			logger:    slog.New(slog.NewTextHandler(os.Stdout, nil)),
			ociRef:    &Reference{RawRef: test.RawRef},
			os:        "linux",
			arch:      "amd64",
			outputDir: tempDir,
			buildDir:  tempDir,
		}

		err := rfs.parseOCIRef()
		if err != nil {
			t.Fatal(err)
		}

		if rfs.ociRef.Repo.Reference.Reference != test.Reference {
			t.Fatalf("Failed to parse tag. Expected: %s, got %s", test.Reference, rfs.ociRef.Repo.Reference.Reference)
		}

		if rfs.ociRef.Repo.Reference.Registry != test.Registry {
			t.Fatalf("Failed to parse registry. Expected: %s, got %s", test.Registry, rfs.ociRef.Repo.Reference.Registry)
		}

		if rfs.ociRef.Repo.Reference.Repository != test.Repository {
			t.Fatalf("Failed to parse repository. Expected: %s, got %s", test.Repository, rfs.ociRef.Repo.Reference.Repository)
		}
	}
}
