package goproxy

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

// Cacher defines a set of intuitive methods used to cache module files for the
// [Goproxy].
type Cacher interface {
	// Get gets the matched cache for the name. It returns the
	// [os.ErrNotExist] if not found.
	//
	// It is the caller's responsibility to close the returned
	// [io.ReadCloser].
	//
	// Note that the returned [io.ReadCloser] can optionally implement the
	// following interfaces:
	//   - [io.Seeker], for the Range request header.
	//   - interface{ ModTime() time.Time }, for the Last-Modified response
	//     header.
	//   - interface{ Checksum() []byte }, for the ETag response header.
	Get(ctx context.Context, name string) (io.ReadCloser, error)

	// Set sets the content as a cache with the name.
	Set(ctx context.Context, name string, content io.ReadSeeker) error
}

// DirCacher implements the [Cacher] using a directory on the local filesystem.
// If the directory does not exist, it will be created with 0750 permissions.
type DirCacher string

// Get implements the [Cacher].
func (dc DirCacher) Get(
	ctx context.Context,
	name string,
) (io.ReadCloser, error) {
	f, err := os.Open(filepath.Join(string(dc), filepath.FromSlash(name)))
	if err != nil {
		return nil, err
	}

	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}

	return &struct {
		*os.File
		os.FileInfo
	}{f, fi}, nil
}

// Set implements the [Cacher].
func (dc DirCacher) Set(
	ctx context.Context,
	name string,
	content io.ReadSeeker,
) error {
	file := filepath.Join(string(dc), filepath.FromSlash(name))

	dir := filepath.Dir(file)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return err
	}

	f, err := ioutil.TempFile(dir, fmt.Sprintf(
		".%s.tmp*",
		filepath.Base(file),
	))
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())

	if _, err := io.Copy(f, content); err != nil {
		return err
	}

	if err := f.Close(); err != nil {
		return err
	}

	return os.Rename(f.Name(), file)
}
