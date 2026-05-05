package inertia

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
)

// VersionProvider returns the current Inertia asset version.
type VersionProvider interface {
	// Version returns the current asset version.
	Version(ctx context.Context) (any, error)
}

// VersionProviderFunc adapts a function to VersionProvider.
type VersionProviderFunc func(ctx context.Context) (any, error)

// Version calls f(ctx).
func (f VersionProviderFunc) Version(ctx context.Context) (any, error) {
	return f(ctx)
}

// StaticVersion returns a provider that always returns version.
func StaticVersion(version any) VersionProvider {
	return VersionProviderFunc(func(ctx context.Context) (any, error) {
		return version, nil
	})
}

// VersionFromFileHash returns a provider that hashes the file at path.
func VersionFromFileHash(path string) VersionProvider {
	return VersionProviderFunc(func(ctx context.Context) (any, error) {
		file, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer file.Close()
		return hashReader(file)
	})
}

// VersionFromFSFileHash returns a provider that hashes a file from fsys.
func VersionFromFSFileHash(fsys fs.FS, path string) VersionProvider {
	return VersionProviderFunc(func(ctx context.Context) (any, error) {
		file, err := fsys.Open(path)
		if err != nil {
			return nil, err
		}
		defer file.Close()
		return hashReader(file)
	})
}

func hashReader(reader io.Reader) (string, error) {
	hash := sha256.New()
	if _, err := io.Copy(hash, reader); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func stringifyVersion(version any) string {
	if version == nil {
		return ""
	}
	return fmt.Sprint(version)
}
