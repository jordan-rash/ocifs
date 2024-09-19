package rootfs

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

func extractLayer(tarGzReader io.Reader, rootfsDir string) error {
	gzReader, err := gzip.NewReader(tarGzReader)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	// Create a tar reader
	tarReader := tar.NewReader(gzReader)

	// Extract files from the tarball
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tarball: %w", err)
		}

		// Handle each type of header
		target := filepath.Join(rootfsDir, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			// Create directory
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
		case tar.TypeReg:
			// Create regular file
			outFile, err := os.Create(target)
			if err != nil {
				return fmt.Errorf("failed to create file: %w", err)
			}
			defer outFile.Close()

			if _, err := io.Copy(outFile, tarReader); err != nil {
				return fmt.Errorf("failed to write file: %w", err)
			}
		}
	}

	return nil
}

// ProgressReader wraps an io.Reader and reports progress
type ProgressReader struct {
	Action     string
	Reader     io.Reader
	Total      int64
	Downloaded int64
	LastPrint  time.Time
	Title      string
	Logger     *slog.Logger
}

func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.Reader.Read(p)
	pr.Downloaded += int64(n)
	now := time.Now()
	if now.Sub(pr.LastPrint) >= time.Millisecond*100 || err == io.EOF {
		pr.printProgress()
		pr.LastPrint = now
	}
	return n, err
}

func (pr *ProgressReader) printProgress() {
	percent := float64(pr.Downloaded) / float64(pr.Total) * 100
	barWidth := 50
	filled := int(percent / 100 * float64(barWidth))
	bar := "[" + string(repeat('=', filled)) + string(repeat(' ', barWidth-filled)) + "]"
	if pr.Logger.Enabled(context.TODO(), slog.LevelInfo) {
		fmt.Printf("\r%s [%s]: %s %.2f%%", pr.Action, pr.Title, bar, percent)
	}
}

func repeat(char rune, count int) []rune {
	result := make([]rune, count)
	for i := range result {
		result[i] = char
	}
	return result
}
