package storage

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/HackDanel/kanban-lab/backend/internal/service"
)

func TestDiskRoundTrip(t *testing.T) {
	disk, err := NewDisk(t.TempDir())
	require.NoError(t, err)
	key := uuid.NewString()

	n, err := disk.Save(context.Background(), key, strings.NewReader("hello"), 5)
	require.NoError(t, err)
	assert.Equal(t, int64(5), n)

	f, err := disk.Open(key)
	require.NoError(t, err)
	data, err := io.ReadAll(f)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	assert.Equal(t, "hello", string(data))

	require.NoError(t, disk.Delete(key))
	_, err = disk.Open(key)
	assert.ErrorIs(t, err, os.ErrNotExist)
	assert.NoError(t, disk.Delete(key), "deleting a missing file is not an error")
}

func TestDiskRejectsOversizedFiles(t *testing.T) {
	dir := t.TempDir()
	disk, err := NewDisk(dir)
	require.NoError(t, err)

	_, err = disk.Save(context.Background(), uuid.NewString(), strings.NewReader("123456"), 5)
	assert.ErrorIs(t, err, service.ErrFileTooLarge)

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	assert.Empty(t, entries, "no partial file is left behind")
}

func TestDiskRejectsUnsafeKeys(t *testing.T) {
	disk, err := NewDisk(t.TempDir())
	require.NoError(t, err)

	for _, key := range []string{"../../etc/passwd", "", "a/b", strings.Repeat("a", 36)} {
		_, err := disk.Save(context.Background(), key, strings.NewReader("x"), 10)
		assert.Error(t, err, key)
		_, err = disk.Open(key)
		assert.Error(t, err, key)
		assert.Error(t, disk.Delete(key), key)
	}
}

func TestNewDiskFailsOnFile(t *testing.T) {
	file := t.TempDir() + "/file"
	require.NoError(t, os.WriteFile(file, nil, 0o600))
	_, err := NewDisk(file + "/sub")
	assert.Error(t, err)
}
