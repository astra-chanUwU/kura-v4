package app

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Backup writes a PostgreSQL custom-format dump and a manifest under
// .local/backups. It does not copy or delete media. The operator copies
// the media directory named in the manifest separately.
func Backup(ctx context.Context, databaseURL, mediaDir string) (string, error) {
	return backupTo(ctx, databaseURL, mediaDir, filepath.Join(".local", "backups"), exec.LookPath, runPgDump)
}

func backupTo(ctx context.Context, databaseURL, mediaDir, destRoot string, lookPath func(string) (string, error), dump func(context.Context, string, string, string) error) (string, error) {
	if strings.TrimSpace(databaseURL) == "" {
		return "", fmt.Errorf("DATABASE_URL is required")
	}
	name, err := databaseName(databaseURL)
	if err != nil {
		return "", err
	}
	pgDump, err := lookPath("pg_dump")
	if err != nil {
		return "", fmt.Errorf("pg_dump is not installed")
	}
	media, err := filepath.Abs(mediaDir)
	if err != nil {
		return "", err
	}
	dir := filepath.Join(destRoot, time.Now().UTC().Format("20060102T150405Z"))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	outFile := filepath.Join(dir, "db.dump")
	if err := dump(ctx, pgDump, databaseURL, outFile); err != nil {
		_ = os.RemoveAll(dir)
		return "", fmt.Errorf("pg_dump failed")
	}
	manifest := fmt.Sprintf("time: %s\ndatabase: %s\nmedia: %s\n", time.Now().UTC().Format(time.RFC3339), name, media)
	if err := os.WriteFile(filepath.Join(dir, "manifest.txt"), []byte(manifest), 0o600); err != nil {
		_ = os.RemoveAll(dir)
		return "", err
	}
	return dir, nil
}

func databaseName(databaseURL string) (string, error) {
	parsed, err := url.Parse(databaseURL)
	if err != nil || parsed.Path == "" {
		return "", fmt.Errorf("DATABASE_URL has no database name")
	}
	name := strings.TrimPrefix(parsed.Path, "/")
	if name == "" || strings.Contains(name, "/") {
		return "", fmt.Errorf("DATABASE_URL has no database name")
	}
	return name, nil
}

func runPgDump(ctx context.Context, pgDump, databaseURL, outFile string) error {
	cmd := exec.CommandContext(ctx, pgDump, "-Fc", "--dbname", databaseURL, "-f", outFile)
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}
