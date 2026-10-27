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
			if entry.IsDir() || !isSeedMedia(entry.Name()) {
				return nil
			}
			_, wasSkipped, err := importSeedFile(ctx, store, svc, path, entry.Name(), nil, func(contentType, sha string, meta board.SeedMeta) []string {
				return seedTags(dir, contentType, sha, meta.Tags)
			})
			if err != nil {
				return err
			}
			if wasSkipped {
				skipped++ // duplicate hash, or a non-media file saved with an image extension
			} else {
				imported++
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	log.Info("seed complete", "imported", imported, "skipped_duplicates", skipped)
	return nil
}

func isSeedMedia(name string) bool {
	return !strings.HasPrefix(name, ".") && seedExts[strings.ToLower(filepath.Ext(name))]
}

// seedFile is one media file under a directory passed to seed or demo.
// Dir is that directory, not the file's parent: folder tags use its base name.
type seedFile struct {
	Path string
	Name string
	Dir  string
}

func collectSeedFiles(dirs []string) ([]seedFile, error) {
	var files []seedFile
	for _, dir := range dirs {
		err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || !isSeedMedia(entry.Name()) {
				return nil
			}
			files = append(files, seedFile{Path: path, Name: entry.Name(), Dir: dir})
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return files, nil
}

// importSeedFile stores one media file and creates its post. skipped is
// true for a duplicate hash or a type the board will not accept. tagsFor
// runs only when a post will be created. A nil owner matches `kura seed`.
func importSeedFile(ctx context.Context, store *storage.Local, svc *board.Service, path, name string, ownerID *string, tagsFor func(contentType, sha string, meta board.SeedMeta) []string) (model.Post, bool, error) {
	sha, size, contentType, err := hashFile(path)
	if err != nil {
		return model.Post{}, false, err
	}
	if !board.IsAllowedMedia(contentType) {
		return model.Post{}, true, nil
	}
	exists, err := svc.ExistsBySHA(ctx, sha)
	if err != nil {
		return model.Post{}, false, err
	}
	if exists {
		return model.Post{}, true, nil
	}
	file, err := os.Open(path) //nolint:gosec // operator-supplied seed path
	if err != nil {
		return model.Post{}, false, err
	}
	key, _, putErr := store.PutHashed(ctx, file)
	closeErr := file.Close()
	if putErr != nil {
		return model.Post{}, false, putErr
	}
	if closeErr != nil {
		return model.Post{}, false, closeErr
	}
	meta := board.ParseSeedName(name)
	post, err := svc.Create(ctx, model.Post{OwnerID: ownerID, OriginalName: name, StorageKey: key, ContentType: contentType, SizeBytes: size,
		Artist: meta.Artist, Source: meta.Source, SourceID: meta.SourceID, Tags: tagsFor(contentType, sha, meta)}, sha)
	if err != nil {
		_ = store.Delete(ctx, key)
		return model.Post{}, false, err
	}
	return post, false, nil
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

func sidecarPath(imagePath string) string {
	return strings.TrimSuffix(imagePath, filepath.Ext(imagePath)) + ".tags"
}

// readSidecarTags reads a one-line booru tag file next to an image.
// ok is false when the sidecar is absent (a download may still be in
// progress); a present file yields its tags, which may be empty.
func readSidecarTags(imagePath string) (tags []string, ok bool, err error) {
	data, err := os.ReadFile(sidecarPath(imagePath)) //nolint:gosec // sidecar beside an operator-supplied seed path
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return parseSidecarTags(string(data)), true, nil
}

// parseSidecarTags takes the first line of a sidecar. Tags are
// space-separated; anything after the first line is ignored so a
// partial trailing write does not become extra tags.
func parseSidecarTags(content string) []string {
	if i := strings.IndexAny(content, "\r\n"); i >= 0 {
		content = content[:i]
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return nil
	}
	return strings.Fields(content)
}

// tagsForSeedFile uses a sidecar's tags plus the seed tag when the
// sidecar is present. seed stays first so the 30-tag cap cannot drop
// it. Without a sidecar the fake seedTags set is used, which is what
// `kura seed` always does.
func tagsForSeedFile(dir, contentType, sha string, filenameTags, sidecar []string, hasSidecar bool) []string {
	if !hasSidecar {
		return seedTags(dir, contentType, sha, filenameTags)
	}
	return board.NormalizeTags(append([]string{"seed"}, sidecar...))
}
