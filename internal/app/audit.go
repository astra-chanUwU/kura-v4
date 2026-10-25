package app

import (
	"context"
	"fmt"
	"image"
	"io"
	"math/bits"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/astra-chanUwU/kura-v4/internal/board"
	"github.com/astra-chanUwU/kura-v4/internal/config"
	"github.com/astra-chanUwU/kura-v4/internal/dao"
	"github.com/astra-chanUwU/kura-v4/internal/db"
	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/astra-chanUwU/kura-v4/internal/repository"
	"github.com/astra-chanUwU/kura-v4/internal/storage"
	"golang.org/x/image/draw"
)

// Audit scans the library for health problems and writes a plain-text
// report. It never deletes or modifies posts or originals; the only
// writes are lazily generated thumbnails for video frames it hashes.
func Audit(ctx context.Context, cfg config.Config, out io.Writer) error {
	conn, err := db.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer conn.Close() //nolint:errcheck
	store, err := storage.NewLocal(cfg.MediaDir)
	if err != nil {
		return err
	}
	defer store.Close() //nolint:errcheck
	svc := board.NewService(repository.NewPosts(dao.NewPosts(conn)), store)

	// Load every post in pages.
	var posts []model.Post
	for offset := int32(0); ; offset += 500 {
		page, err := svc.List(ctx, 500, offset, true) // operator audit sees every rating
		if err != nil {
			return err
		}
		posts = append(posts, page...)
		if len(page) < 500 {
			break
		}
	}

	var broken []string
	hashes := make(map[string]uint64, len(posts)) // post id -> perceptual hash
	referenced := map[string]bool{}
	noTags, noArtist, noDims, noThumb := 0, 0, 0, 0

	jobs := make(chan model.Post)
	var mu sync.Mutex
	var wg sync.WaitGroup
	worker := func() {
		defer wg.Done()
		for p := range jobs {
			reason := auditPost(ctx, svc, store, cfg.MediaDir, p, &hashes, &mu)
			mu.Lock()
			if reason != "" {
				broken = append(broken, fmt.Sprintf("%s (%s)", p.OriginalName, reason))
			}
			mu.Unlock()
		}
	}
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go worker()
	}
	for _, p := range posts {
		referenced[p.StorageKey] = true
		if p.ThumbnailKey != "" {
			referenced[p.ThumbnailKey] = true
		} else {
			noThumb++
		}
		if len(p.Tags) == 0 {
			noTags++
		}
		if p.Artist == "" {
			noArtist++
		}
		if p.Width <= 0 || p.Height <= 0 {
			noDims++
		}
		jobs <- p
	}
	close(jobs)
	wg.Wait()

	// Exact duplicates: identical SHA-256 already share one stored file,
	// so these are duplicate posts pointing at the same art.
	var exactGroups []string
	rows, err := conn.QueryContext(ctx, `SELECT sha256, count(*), string_agg(title, ' | ') FROM posts WHERE sha256<>'' GROUP BY sha256 HAVING count(*)>1 ORDER BY count(*) DESC`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var sha string
		var n int
		var titles string
		if err := rows.Scan(&sha, &n, &titles); err == nil {
			exactGroups = append(exactGroups, fmt.Sprintf("%d posts, same file: %s", n, truncate(titles, 120)))
		}
	}
	rows.Close() //nolint:errcheck
	if err := rows.Err(); err != nil {
		return err
	}

	// Hashing may have generated fresh thumbnails; pick up their keys
	// so they are not mistaken for orphans.
	if thumbRows, err := conn.QueryContext(ctx, `SELECT thumbnail_key FROM posts WHERE thumbnail_key<>''`); err == nil {
		for thumbRows.Next() {
			var key string
			if err := thumbRows.Scan(&key); err == nil {
				referenced[key] = true
			}
		}
		thumbRows.Close() //nolint:errcheck
	}

	// Orphan files: in the media dir but referenced by no post.
	var orphans []string
	entries, err := os.ReadDir(cfg.MediaDir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() || strings.HasSuffix(e.Name(), ".partial") {
			continue
		}
		if !referenced[e.Name()] {
			orphans = append(orphans, e.Name())
		}
	}

	nearGroups := nearDuplicateGroups(posts, hashes)

	fmt.Fprintf(out, "Kura library audit — %d posts\n\n", len(posts)) //nolint:errcheck
	section(out, "Broken / mismatched files", broken, "post title (reason)")
	section(out, "Exact duplicate posts (same bytes)", exactGroups, "")
	section(out, "Near-duplicate groups (same art, different size/format)", nearGroups, "")
	section(out, "Orphan files (stored, referenced by no post)", orphans, "")
	fmt.Fprintf(out, "Metadata gaps\n")                                                                                               //nolint:errcheck
	fmt.Fprintf(out, "  no tags: %d · no artist: %d · no dimensions: %d · no thumbnail yet: %d\n", noTags, noArtist, noDims, noThumb) //nolint:errcheck
	return nil
}

func section(out io.Writer, name string, items []string, _ string) {
	fmt.Fprintf(out, "%s: %d\n", name, len(items)) //nolint:errcheck
	show := items
	if len(show) > 20 {
		show = show[:20]
	}
	for _, item := range show {
		fmt.Fprintf(out, "  - %s\n", item) //nolint:errcheck
	}
	if len(items) > len(show) {
		fmt.Fprintf(out, "  … and %d more\n", len(items)-len(show)) //nolint:errcheck
	}
	fmt.Fprintln(out) //nolint:errcheck
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// auditPost checks one post's file and computes its perceptual hash.
// It returns a non-empty reason when the post looks broken.
func auditPost(ctx context.Context, svc *board.Service, store *storage.Local, mediaDir string, p model.Post, hashes *map[string]uint64, mu *sync.Mutex) string {
	path := filepath.Join(mediaDir, p.StorageKey)
	info, err := os.Stat(path)
	if err != nil {
		return "file missing from media storage"
	}
	if info.Size() == 0 {
		return "file is empty"
	}
	if p.SizeBytes > 0 && info.Size() != p.SizeBytes {
		return fmt.Sprintf("size mismatch (db %d, file %d)", p.SizeBytes, info.Size())
	}
	file, err := store.Open(ctx, p.StorageKey)
	if err != nil {
		return "cannot open file"
	}
	defer file.Close() //nolint:errcheck
	head := make([]byte, 512)
	n, _ := file.Read(head)
	if sniffed := board.SniffContentType(head[:n], p.OriginalName); sniffed != p.ContentType {
		return fmt.Sprintf("type mismatch (db %s, file looks like %s)", p.ContentType, sniffed)
	}
	// Perceptual hash from the original (or the video's frame).
	if seeker, ok := file.(io.Seeker); ok {
		_, _ = seeker.Seek(0, io.SeekStart)
		if h, err := dHash(file); err == nil {
			mu.Lock()
			(*hashes)[p.ID] = h
			mu.Unlock()
			return ""
		}
	}
	if stream, _, err := svc.Thumbnail(ctx, p); err == nil {
		defer stream.Close() //nolint:errcheck
		if h, err := dHash(stream); err == nil {
			mu.Lock()
			(*hashes)[p.ID] = h
			mu.Unlock()
			return ""
		}
	}
	return "cannot decode image data"
}

// dHash is a 64-bit difference hash: same art at different sizes or
// encodings lands within a few bits of the same value.
func dHash(r io.Reader) (uint64, error) {
	img, _, err := image.Decode(r)
	if err != nil {
		return 0, err
	}
	const w, h = 9, 8
	small := image.NewGray(image.Rect(0, 0, w, h))
	draw.ApproxBiLinear.Scale(small, small.Bounds(), img, img.Bounds(), draw.Src, nil)
	var hash uint64
	for y := 0; y < h; y++ {
		for x := 0; x < w-1; x++ {
			hash <<= 1
			if small.GrayAt(x, y).Y > small.GrayAt(x+1, y).Y {
				hash |= 1
			}
		}
	}
	return hash, nil
}

// nearDuplicateGroups clusters posts whose hashes are within a small
// Hamming distance (union-find over all pairs).
func nearDuplicateGroups(posts []model.Post, hashes map[string]uint64) []string {
	byID := map[string]model.Post{}
	ids := make([]string, 0, len(hashes))
	for _, p := range posts {
		byID[p.ID] = p
		if _, ok := hashes[p.ID]; ok {
			ids = append(ids, p.ID)
		}
	}
	parent := map[string]string{}
	var find func(string) string
	find = func(x string) string {
		if parent[x] == x {
			return x
		}
		parent[x] = find(parent[x])
		return parent[x]
	}
	for _, id := range ids {
		parent[id] = id
	}
	for i := 0; i < len(ids); i++ {
		for j := i + 1; j < len(ids); j++ {
			if bits.OnesCount64(hashes[ids[i]]^hashes[ids[j]]) <= 6 {
				ri, rj := find(ids[i]), find(ids[j])
				if ri != rj {
					parent[ri] = rj
				}
			}
		}
	}
	groups := map[string][]model.Post{}
	for _, id := range ids {
		root := find(id)
		groups[root] = append(groups[root], byID[id])
	}
	var out []string
	for _, members := range groups {
		if len(members) < 2 {
			continue
		}
		sort.Slice(members, func(a, b int) bool { return members[a].SizeBytes > members[b].SizeBytes })
		parts := make([]string, 0, len(members))
		for _, m := range members {
			parts = append(parts, fmt.Sprintf("%s [%dx%d, %s]", truncate(m.OriginalName, 40), m.Width, m.Height, m.ContentType))
		}
		out = append(out, fmt.Sprintf("%d versions: %s", len(members), strings.Join(parts, " · ")))
	}
	sort.Strings(out)
	return out
}
