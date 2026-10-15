// Package storage keeps attachment files on the local disk.
package storage

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"

	"github.com/google/uuid"

	"github.com/HackDanel/kanban-lab/backend/internal/service"
)

// Disk stores files in a directory. All access goes through os.Root, so no
// key can reach outside the directory even if validation were bypassed.
type Disk struct {
	root *os.Root
}

// NewDisk creates the directory if needed.
func NewDisk(dir string) (*Disk, error) {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("create upload dir: %w", err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, fmt.Errorf("open upload dir: %w", err)
	}
	return &Disk{root: root}, nil
}

// checkKey admits only the canonical UUID keys the service generates.
func checkKey(key string) error {
	if id, err := uuid.Parse(key); err != nil || id.String() != key {
		return fmt.Errorf("invalid storage key %q", key)
	}
	return nil
}

// Save writes r to key, failing with service.ErrFileTooLarge after max bytes.
func (d *Disk) Save(_ context.Context, key string, r io.Reader, max int64) (int64, error) {
	if err := checkKey(key); err != nil {
		return 0, err
	}
	suffix := make([]byte, 8)
	if _, err := rand.Read(suffix); err != nil {
		return 0, err
	}
	tmpName := ".upload-" + hex.EncodeToString(suffix)
	tmp, err := d.root.OpenFile(tmpName, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o640)
	if err != nil {
		return 0, err
	}
	defer func() { _ = d.root.Remove(tmpName) }()

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
	return n, d.root.Rename(tmpName, key)
}

// Open opens the file stored under key.
func (d *Disk) Open(key string) (io.ReadSeekCloser, error) {
	if err := checkKey(key); err != nil {
		return nil, err
	}
	return d.root.Open(key)
}

// Delete removes the file stored under key; a missing file is not an error.
func (d *Disk) Delete(key string) error {
	if err := checkKey(key); err != nil {
		return err
	}
	if err := d.root.Remove(key); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}
