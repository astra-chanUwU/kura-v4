package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSweepPartialsRemovesOnlyOldTemps(t *testing.T) {
	dir := t.TempDir()
	fresh := filepath.Join(dir, "fresh.partial")
	recent := filepath.Join(dir, "recent.partial")
	old := filepath.Join(dir, "old.partial")
	kept := filepath.Join(dir, "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	nested := filepath.Join(dir, "nested.partial")
	require.NoError(t, os.WriteFile(fresh, []byte("new"), 0o600))
	require.NoError(t, os.WriteFile(recent, []byte("young"), 0o600))
	require.NoError(t, os.WriteFile(old, []byte("stale"), 0o600))
	require.NoError(t, os.WriteFile(kept, []byte("original"), 0o600))
	require.NoError(t, os.Mkdir(nested, 0o700))
	hourAgo := time.Now().Add(-59 * time.Minute)
	require.NoError(t, os.Chtimes(recent, hourAgo, hourAgo))
	past := time.Now().Add(-2 * time.Hour)
	require.NoError(t, os.Chtimes(old, past, past))
	require.NoError(t, os.Chtimes(kept, past, past))
	require.NoError(t, os.Chtimes(nested, past, past))

	removed, err := SweepPartials(dir, time.Hour)
	require.NoError(t, err)
	require.Equal(t, 1, removed)
	for _, path := range []string{fresh, recent, kept} {
		_, err = os.Stat(path)
		require.NoError(t, err)
	}
	info, err := os.Stat(nested)
	require.NoError(t, err)
	require.True(t, info.IsDir())
	_, err = os.Stat(old)
	require.True(t, os.IsNotExist(err))
}

func TestSweepPartialsRejectsNonPositiveCutoff(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "abandoned.partial")
	require.NoError(t, os.WriteFile(path, []byte("keep"), 0o600))
	past := time.Now().Add(-2 * time.Hour)
	require.NoError(t, os.Chtimes(path, past, past))
	n, err := SweepPartials(dir, 0)
	require.Error(t, err)
	require.Equal(t, 0, n)
	_, err = os.Stat(path)
	require.NoError(t, err)
}
