package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBackupRefusesEmptyDatabaseURL(t *testing.T) {
	dir := t.TempDir()
	_, err := backupTo(context.Background(), "  ", ".local/media", dir, func(string) (string, error) {
		t.Fatal("pg_dump should not be looked up")
		return "", nil
	}, nil)
	require.EqualError(t, err, "DATABASE_URL is required")
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Empty(t, entries)
}

func TestBackupCreatesNothingWhenPgDumpIsMissing(t *testing.T) {
	dir := t.TempDir()
	_, err := backupTo(context.Background(), "postgres://kura:secret@localhost:5432/kura", ".local/media", dir, func(string) (string, error) {
		return "", os.ErrNotExist
	}, func(context.Context, string, string, string) error {
		t.Fatal("pg_dump should not run")
		return nil
	})
	require.EqualError(t, err, "pg_dump is not installed")
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Empty(t, entries)
}

func TestBackupWritesDumpAndManifestWithoutPassword(t *testing.T) {
	dir := t.TempDir()
	media := t.TempDir()
	got, err := backupTo(context.Background(), "postgres://kura:secret@localhost:5432/kura", media, dir, func(name string) (string, error) {
		require.Equal(t, "pg_dump", name)
		return "pg_dump", nil
	}, func(_ context.Context, _, databaseURL, outFile string) error {
		require.Contains(t, databaseURL, "secret")
		return os.WriteFile(outFile, []byte("dump"), 0o600)
	})
	require.NoError(t, err)
	body, err := os.ReadFile(filepath.Join(got, "manifest.txt"))
	require.NoError(t, err)
	text := string(body)
	require.Contains(t, text, "database: kura")
	require.NotContains(t, text, "secret")
	require.Contains(t, text, "media:")
	_, err = os.Stat(filepath.Join(got, "db.dump"))
	require.NoError(t, err)
}
