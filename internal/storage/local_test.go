package storage

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOriginalRoundTripAndTraversal(t *testing.T) {
	store, err := NewLocal(t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })
	key, err := store.Put(context.Background(), strings.NewReader("original bytes"))
	require.NoError(t, err)
	file, err := store.Open(context.Background(), key)
	require.NoError(t, err)
	contents, err := io.ReadAll(file)
	require.NoError(t, err)
	require.NoError(t, file.Close())
	require.Equal(t, "original bytes", string(contents))
	_, err = store.Open(context.Background(), "../outside")
	require.Error(t, err)
}
func TestFailedWriteLeavesNoPublishedFile(t *testing.T) {
	dir := t.TempDir()
	store, err := NewLocal(dir)
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })
	_, err = store.Put(context.Background(), brokenReader{})
	require.Error(t, err)
	files, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Empty(t, files)
}

type brokenReader struct{}

func (brokenReader) Read([]byte) (int, error) { return 0, errors.New("source disconnected") }
