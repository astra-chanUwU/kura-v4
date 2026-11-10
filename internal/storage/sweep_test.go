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
	old := filepath.Join(dir, "old.partial")
	kept := filepath.Join(dir, "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	require.NoError(t, os.WriteFile(fresh, []byte("new"), 0o600))
	require.NoError(t, os.WriteFile(old, []byte("stale"), 0o600))
	require.NoError(t, os.WriteFile(kept, []byte("original"), 0o600))
	past := time.Now().Add(-2 * time.Hour)
	require.NoError(t, os.Chtimes(old, past, past))

	removed, err := SweepPartials(dir, time.Hour)
	require.NoError(t, err)
	require.Equal(t, 1, removed)
	_, err = os.Stat(fresh)
	require.NoError(t, err)
	_, err = os.Stat(kept)
	require.NoError(t, err)
	_, err = os.Stat(old)
	require.True(t, os.IsNotExist(err))
}
