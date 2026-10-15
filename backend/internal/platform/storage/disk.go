// Package storage keeps attachment files on the local disk.
package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/google/uuid"

	"github.com/HackDanel/kanban-lab/backend/internal/service"
)

// Disk stores files in a directory.
type Disk struct {
	dir string
}

// NewDisk creates the directory if needed.
func NewDisk(dir string) (*Disk, error) {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("create upload dir: %w", err)
	}
	return &Disk{dir: dir}, nil
}

func (d *Disk) path(key string) (string, error) {
	// Only the canonical UUID keys the service generates are accepted, so a key
	// can never point outside the directory.
	if id, err := uuid.Parse(key); err != nil || id.String() != key {
		return "", fmt.Errorf("invalid storage key %q", key)
	}
	return filepath.Join(d.dir, key), nil
}

// Save writes r to key, failing with service.ErrFileTooLarge after max bytes.
func (d *Disk) Save(_ context.Context, key string, r io.Reader, max int64) (int64, error) {
	p, err := d.path(key)
	if err != nil {
		return 0, err
	}
	tmp, err := os.CreateTemp(d.dir, ".upload-*")
	if err != nil {
		return 0, err
	}
	defer func() { _ = os.Remove(tmp.Name()) }()

	n, err := io.Copy(tmp, io.LimitReader(r, max+1))
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return 0, err
	}
	if n > max {
		return 0, service.ErrFileTooLarge
	}
	return n, os.Rename(tmp.Name(), p)
}

// Open opens the file stored under key.
func (d *Disk) Open(key string) (io.ReadSeekCloser, error) {
	p, err := d.path(key)
	if err != nil {
		return nil, err
	}
	return os.Open(p)
}

// Delete removes the file stored under key; a missing file is not an error.
func (d *Disk) Delete(key string) error {
	p, err := d.path(key)
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
