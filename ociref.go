package ocifs

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
)

type Reference struct {
	RawRef     string
	Repo       *remote.Repository
	Descriptor v1.Descriptor
	Manifest   *v1.Manifest
}

func (r *rootFS) parseOCIRef() error {
	repo, err := remote.NewRepository(r.ociRef.RawRef)
	if err != nil {
		return err
	}
	r.ociRef.Repo = repo

	// Set up authentication if needed
	repo.Client = &auth.Client{
		Cache: auth.DefaultCache,
		Credential: func(ctx context.Context, registry string) (auth.Credential, error) {
			switch r.ociRef.Repo.Reference.Registry {
			case "docker.io":
				authGet, err := http.Get(fmt.Sprintf("https://auth.docker.io/token?service=registry.docker.io&scope=repository:%s:pull", repo.Reference.Repository))
				if err != nil {
					return auth.Credential{}, fmt.Errorf("failed to fetch auth token: %w", err)
				}
				defer authGet.Body.Close()
				authBody := make(map[string]any)
				if err := json.NewDecoder(authGet.Body).Decode(&authBody); err != nil {
					return auth.Credential{}, fmt.Errorf("failed to decode auth response: %w", err)
				}
				token, ok := authBody["token"].(string)
				if !ok {
					return auth.Credential{}, errors.New("failed to parse token")
				}
				cred := auth.Credential{
					AccessToken: token,
				}
				return cred, nil
			case "ghcr.io":
				cred := auth.Credential{
					// TODO: Do this better
					Username: "jordan-rash",
					Password: os.Getenv("GH_PAT"),
				}
				return cred, nil
			default:
				return auth.EmptyCredential, nil
			}
		},
	}

	// Resolve the reference to get the descriptor
	r.ociRef.Descriptor, err = repo.Resolve(context.TODO(), repo.Reference.Reference)
	if err != nil {
		return err
	}

	// Fetch the manifest or index
	manifestBytes, err := repo.Fetch(context.TODO(), r.ociRef.Descriptor)
	if err != nil {
		return err
	}

	// •	OCI Index (v1.Index): application/vnd.oci.image.index.v1+json
	//   •	Represents a multi-platform image.
	//   •	Contains a list of manifests, each with a Platform field indicating the OS and architecture.
	// •	OCI Manifest (v1.Manifest): application/vnd.oci.image.manifest.v1+json
	//   •	Describes an image for a specific platform.
	//   •	Contains a list of layers and a config object.
	switch r.ociRef.Descriptor.MediaType {
	case v1.MediaTypeImageManifest:
		var manifest v1.Manifest
		indexDecoder := json.NewDecoder(manifestBytes)
		if err := indexDecoder.Decode(&manifest); err != nil {
			return err
		}
		r.ociRef.Manifest = &manifest
	case v1.MediaTypeImageIndex:
		var index v1.Index
		manifestDecoder := json.NewDecoder(manifestBytes)
		if err := manifestDecoder.Decode(&index); err != nil {
			return err
		}
		for _, manifest := range index.Manifests {
			if manifest.MediaType == v1.MediaTypeImageManifest && manifest.Platform.OS == r.os && manifest.Platform.Architecture == r.arch {
				manifestBytes, err := repo.Fetch(context.TODO(), manifest)
				if err != nil {
					return err
				}
				var manifest v1.Manifest
				indexDecoder := json.NewDecoder(manifestBytes)
				if err := indexDecoder.Decode(&manifest); err != nil {
					return err
				}
				r.ociRef.Manifest = &manifest
				break
			}
		}
	}

	if r.ociRef.Manifest == nil {
		r.logger.Error("no matching manifest found", slog.String("os", r.os), slog.String("arch", r.arch))
		return errors.New("no matching manifest found")
	}

	return nil
}

func (r *rootFS) downloadExtractLayers() error {
	for _, layer := range r.ociRef.Manifest.Layers {
		r.logger.Debug("Downloading layer", slog.String("digest", layer.Digest.Encoded()), slog.Int64("size", layer.Size), slog.String("mediaType", layer.MediaType))
		reader, err := r.ociRef.Repo.Blobs().Fetch(context.TODO(), layer)
		if err != nil {
			return err
		}
		defer reader.Close()

		bs := new(bytes.Buffer)
		pr := ProgressReader{
			Action: "Downloading",
			Reader: reader,
			Total:  layer.Size,
			Title:  layer.Digest.Encoded()[len(layer.Digest.Encoded())-8:],
			Logger: r.logger,
		}

		_, err = io.Copy(bs, &pr)
		if err != nil {
			return err
		}
		fmt.Println()

		r.logger.Debug("Extracting layer", slog.String("digest", layer.Digest.Encoded()))
		err = extractLayer(bs, r.buildDir)
		if err != nil {
			return err
		}
	}
	return nil
}

// └─❯ mke2fs -t ext4 -d rootfs rootfs.ext4 150M
// └─❯ resize2fs -M ./rootfs.ext4
func (r *rootFS) makeRootFS() error {
	err := os.MkdirAll(r.outputDir, 0755)
	if err != nil {
		return err
	}

	r.logger.Debug("Creating rootfs.ext4")
	err = exec.Command("mke2fs", "-t", "ext4", "-d", r.buildDir, filepath.Join(r.outputDir, "rootfs.ext4"), "150M").Run()
	if err != nil {
		return err
	}

	r.logger.Debug("Resizing rootfs.ext4")
	err = exec.Command("resize2fs", "-M", filepath.Join(r.outputDir, "rootfs.ext4")).Run()
	if err != nil {
		return err
	}
	return nil
}
