package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/astra-chanUwU/kura-v4/internal/board"
	"github.com/astra-chanUwU/kura-v4/internal/config"
	"github.com/astra-chanUwU/kura-v4/internal/dao"
	"github.com/astra-chanUwU/kura-v4/internal/db"
	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/astra-chanUwU/kura-v4/internal/repository"
	"github.com/astra-chanUwU/kura-v4/internal/storage"
)

var seedExts = map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true, ".avif": true, ".mp4": true, ".webm": true, ".mov": true}

// Seed imports image/video files from dirs (recursively) as public posts.
// Originals are copied byte-for-byte into media storage; files already
// imported (same SHA-256) are skipped, so re-running is safe.
func Seed(ctx context.Context, cfg config.Config, dirs []string, log *slog.Logger) error {
	connectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	conn, err := db.Open(connectCtx, cfg.DatabaseURL)
	cancel()
	if err != nil {
		return err
	}
	defer conn.Close() //nolint:errcheck
	if err := db.Migrate(ctx, conn); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	store, err := storage.NewLocal(cfg.MediaDir)
	if err != nil {
		return err
	}
	defer store.Close() //nolint:errcheck
	svc := board.NewService(repository.NewPosts(dao.NewPosts(conn)), store)

	imported, skipped := 0, 0
	for _, dir := range dirs {
		err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") || !seedExts[strings.ToLower(filepath.Ext(entry.Name()))] {
				return nil
			}
			sha, size, contentType, err := hashFile(path)
			if err != nil {
				return err
			}
			if !board.IsAllowedMedia(contentType) {
				skipped++ // e.g. an HTML error page saved with a .gif name
				return nil
			}
			exists, err := svc.ExistsBySHA(ctx, sha)
			if err != nil {
				return err
			}
			if exists {
				skipped++
				return nil
			}
			file, err := os.Open(path) //nolint:gosec // operator-supplied seed path
			if err != nil {
				return err
			}
			key, _, putErr := store.PutHashed(ctx, file)
			closeErr := file.Close()
			if putErr != nil {
				return putErr
			}
			if closeErr != nil {
				return closeErr
			}
			meta := board.ParseSeedName(entry.Name())
			if _, err := svc.Create(ctx, model.Post{OriginalName: entry.Name(), StorageKey: key, ContentType: contentType, SizeBytes: size,
				Artist: meta.Artist, Source: meta.Source, SourceID: meta.SourceID, Tags: meta.Tags}, sha); err != nil {
				_ = store.Delete(ctx, key)
				return err
			}
			imported++
			return nil
		})
		if err != nil {
			return err
		}
	}
	log.Info("seed complete", "imported", imported, "skipped_duplicates", skipped)
	return nil
}

func hashFile(path string) (string, int64, string, error) {
	file, err := os.Open(path) //nolint:gosec // operator-supplied seed path
	if err != nil {
		return "", 0, "", err
	}
	defer file.Close() //nolint:errcheck
	head := make([]byte, 512)
	n, _ := file.Read(head)
	contentType := board.SniffContentType(head[:n], path)
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", 0, "", err
	}
	h := sha256.New()
	size, err := io.Copy(h, file)
	return hex.EncodeToString(h.Sum(nil)), size, contentType, err
}
