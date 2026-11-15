package app

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSweepAbandonedUploadsKeepsFilesNewerThanOneHour(t *testing.T) {
	dir := t.TempDir()
	fresh := filepath.Join(dir, "fresh.partial")
	recent := filepath.Join(dir, "recent.partial")
	old := filepath.Join(dir, "old.partial")
	original := filepath.Join(dir, "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	require.NoError(t, os.WriteFile(fresh, []byte("a"), 0o600))
	require.NoError(t, os.WriteFile(recent, []byte("b"), 0o600))
	require.NoError(t, os.WriteFile(old, []byte("c"), 0o600))
	require.NoError(t, os.WriteFile(original, []byte("original-bytes"), 0o600))
	recentAt := time.Now().Add(-59 * time.Minute)
	past := time.Now().Add(-2 * time.Hour)
	require.NoError(t, os.Chtimes(recent, recentAt, recentAt))
	require.NoError(t, os.Chtimes(old, past, past))
	require.NoError(t, os.Chtimes(original, past, past))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	sweepAbandonedUploads(ctx, dir, slog.New(slog.NewTextHandler(io.Discard, nil)))

	for _, path := range []string{fresh, recent, original} {
		_, err := os.Stat(path)
		require.NoError(t, err)
	}
	_, err := os.Stat(old)
	require.True(t, os.IsNotExist(err))
	kept, err := os.ReadFile(original)
	require.NoError(t, err)
	require.Equal(t, "original-bytes", string(kept))
}
