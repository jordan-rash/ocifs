package rootfs

import "log/slog"

// WithLogger sets the logger to use for the rootFS module
// Default: No output
func WithLogger(s *slog.Logger) RootFSOption {
	return func(n *rootFS) error {
		if s != nil {
			n.logger = s
		}
		return nil
	}
}

// WithOSArch sets the OS and architecture to local when parsing the OCI reference
// Defaults: OS: linux, Arch: amd64
func WithOSArch(o, a string) RootFSOption {
	return func(n *rootFS) error {
		if o != "" {
			n.os = o
		}
		if a != "" {
			n.arch = a
		}
		return nil
	}
}
