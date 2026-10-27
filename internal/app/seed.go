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
			tags := seedTags(dir, contentType, sha, meta.Tags)
			if _, err := svc.Create(ctx, model.Post{OriginalName: entry.Name(), StorageKey: key, ContentType: contentType, SizeBytes: size,
				Artist: meta.Artist, Source: meta.Source, SourceID: meta.SourceID, Tags: tags}, sha); err != nil {
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

// Fake-tag pools for seeded test data, keyed by a hint from the folder
// name. Tags are deliberately plausible-but-fake: this is a test board.
var seedPools = []struct {
	hint string
	pool []string
}{
	{"anime", []string{"anime", "illustration", "1girl", "2girls", "school_uniform", "long_hair", "smile", "outdoors"}},
	{"meme", []string{"meme", "funny", "reaction", "text", "screenshot"}},
	{"art", []string{"artwork", "painting", "digital_art", "sketch", "colorful", "detailed", "fantasy"}},
	{"iran", []string{"photo", "outdoors", "travel", "city", "landscape", "day"}},
	{"legacy", []string{"retro", "mac", "screenshot", "computer", "vintage", "desktop"}},
	{"screenshot", []string{"screenshot", "desktop", "ui", "window", "dark"}},
	{"folder", []string{"anime", "illustration", "twitter", "1girl", "long_hair", "outdoors", "indoors", "smile"}},
}

var seedDefaultPool = []string{"photo", "image", "candid", "colorful", "detailed", "portrait", "landscape", "day"}

// seedTags builds a deterministic fake tag set for one seeded file: the
// source folder, the medium, the filename's own tags, plus three pool
// tags picked by the file's SHA-256 (same file -> same tags, always).
func seedTags(dir, contentType, sha string, filenameTags []string) []string {
	folder := board.NormalizeTags([]string{filepath.Base(dir)})
	tags := []string{"seed"}
	tags = append(tags, folder...)
	switch {
	case strings.HasPrefix(contentType, "video/"):
		tags = append(tags, "video")
	case contentType == "image/gif":
		tags = append(tags, "gif")
	default:
		tags = append(tags, "image")
	}
	tags = append(tags, filenameTags...)
	pool := seedDefaultPool
	hint := strings.ToLower(filepath.Base(dir))
	for _, candidate := range seedPools {
		if strings.Contains(hint, candidate.hint) {
			pool = candidate.pool
			break
		}
	}
	seen := map[string]bool{}
	for _, t := range tags {
		seen[t] = true
	}
	picked := 0
	for i := 0; i+2 < len(sha) && picked < 3; i += 2 {
		var b int
		if _, err := fmt.Sscanf(sha[i:i+2], "%02x", &b); err != nil {
			break
		}
		tag := pool[b%len(pool)]
		if seen[tag] {
			continue
		}
		seen[tag] = true
		tags = append(tags, tag)
		picked++
	}
	return board.NormalizeTags(tags)
}
