package app

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/astra-chanUwU/kura-v4/internal/accounts"
	"github.com/astra-chanUwU/kura-v4/internal/board"
	"github.com/astra-chanUwU/kura-v4/internal/collections"
	"github.com/astra-chanUwU/kura-v4/internal/config"
	"github.com/astra-chanUwU/kura-v4/internal/dao"
	"github.com/astra-chanUwU/kura-v4/internal/db"
	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/astra-chanUwU/kura-v4/internal/moderation"
	"github.com/astra-chanUwU/kura-v4/internal/repository"
	"github.com/astra-chanUwU/kura-v4/internal/storage"
)

// demoPassword is shared by every demo account. The password policy
// rejects anything shorter than 15 characters, so the 14-character
// "kura-demo-2026" cannot be used.
const demoPassword = "kura-demo-2026!"

const (
	demoBootstrapActor = "cli:demo"
	demoTagEdits       = 40
	demoHidePosts      = 5
	demoDeletePosts    = 3
	// Fixed PCG seed: the same files produce the same owners, edits and lists.
	demoSeedHi uint64 = 0x6b75726164656d6f
	demoSeedLo uint64 = 0x3230323600000004
)

type demoAccount struct {
	Username         string
	DisplayName      string
	Role             model.Role
	Bio              string
	Location         string
	Website          string
	Theme            string
	Font             string
	Include          []string
	Exclude          []string
	ShowNSFW         bool
	Avatar           bool
	Weight           int
	FavMin           int
	FavMax           int
	ExtraRoleHistory bool
}

// demoAccounts is the roster. Weights are upload share (hana and ren
// carry the board, sora barely uploads). Favorite ranges sit inside
// 15–80, with the heavy uploaders also the heavy favoriters.
var demoAccounts = []demoAccount{
	{Username: "astra", DisplayName: "Astra Vale", Role: model.RoleSuperAdmin, Bio: "Keeps the rollback log honest and the last super admin seat warm. If a tag is wrong, it does not stay wrong.", Location: "Clock tower", Website: "https://example.com/astra", Theme: "umineko", Font: "serif", Include: []string{"illustration", "outdoors"}, Exclude: []string{"meme"}, Weight: 8, FavMin: 20, FavMax: 35, Avatar: true},
	{Username: "noa", DisplayName: "Noa Lin", Role: model.RoleAdmin, Bio: "Admin on weekdays and a quiet browser on weekends. Files the reports, then favorites the soft ones.", Location: "Library mezzanine", Website: "https://example.com/noa", Theme: "kura", Font: "system", Weight: 8, FavMin: 18, FavMax: 30, Avatar: true, ExtraRoleHistory: true},
	{Username: "kiri", DisplayName: "Kiri Mori", Role: model.RoleModerator, Bio: "Moderator with a silver-hair bias and very little patience for a mistagged artist.", Location: "North ward", Theme: "safebooru", Font: "rounded", Include: []string{"silver_hair", "long_hair"}, Exclude: []string{"comic"}, Weight: 8, FavMin: 25, FavMax: 40, Avatar: true},
	{Username: "mika", DisplayName: "Mika Sato", Role: model.RoleModerator, Bio: "The other moderator. Hides what does not belong on the board and leaves the reason in the history.", Location: "Harbor district", Theme: "rule34", Font: "mono", ShowNSFW: true, Weight: 8, FavMin: 30, FavMax: 50, Avatar: true},
	{Username: "hana", DisplayName: "Hana Iori", Role: model.RoleMember, Bio: "Uploads like the board might run out of pictures. Most of the shelf is hers, and she knows it.", Location: "Flower market", Website: "https://example.com/hana", Theme: "bernkastel", Font: "hack", Include: []string{"1girl", "smile"}, Exclude: []string{"text"}, Weight: 28, FavMin: 60, FavMax: 80, Avatar: true},
	{Username: "ren", DisplayName: "Ren Aoki", Role: model.RoleMember, Bio: "Second only to Hana on uploads. Keeps a private list for the frames that feel like late evening.", Location: "West rooftop", Theme: "lambdadelta", Font: "serif", Include: []string{"yuri", "2girls"}, Exclude: []string{"monochrome"}, ShowNSFW: true, Weight: 24, FavMin: 50, FavMax: 75, Avatar: true},
	{Username: "yuki", DisplayName: "Yuki Shiro", Role: model.RoleMember, Bio: "Posts when the light is good. Snow, school uniforms, and captions that stay out of the way.", Location: "Snow town", Website: "https://example.com/yuki", Theme: "beatrice", Font: "rounded", ShowNSFW: true, Weight: 12, FavMin: 35, FavMax: 55, Avatar: true},
	{Username: "sora", DisplayName: "Sora Minami", Role: model.RoleMember, Bio: "A handful of uploads and a long favorite habit. Here for the sky pictures more than the arguments.", Location: "South hill", Theme: "erika", Font: "system", Weight: 4, FavMin: 15, FavMax: 25, Avatar: false},
}

type demoListTemplate struct {
	Title string
	Desc  string
}

var demoListCatalog = []demoListTemplate{
	{"Silver hair shelf", "Long silver hair, saved for the days the board is too loud."},
	{"Quiet smiles", "Soft expressions and nothing that shouts."},
	{"Late evening", "The frames that feel like the lights just went down."},
	{"School rooftop", "Uniforms, sky, and the wind in the hair."},
	{"Yuri folder", "Two girls, one composition, no explanation required."},
	{"Staff picks", "Pictures the mods would actually pin."},
	{"Sky and weather", "Clouds, snow, and the occasional clear day."},
	{"Just 1girl", "A single subject, centered, and left alone."},
	{"Artist credits", "Saved because the signature was worth reading."},
	{"Night window", "Indoors, lamplight, and someone looking out."},
	{"Favorites I will not explain", "Private. The public lists are the polite ones."},
	{"Revisit later", "Not sorted yet. That is the point."},
}

var demoAddTags = []string{"blush", "outdoors", "looking_at_viewer", "closed_eyes", "open_mouth", "sky", "night", "portrait", "window", "from_side"}

var demoArtists = []string{"kasumi", "north_wind", "atelier_mori", "suzu", "komori"}

type demoEnv struct {
	log         *slog.Logger
	accounts    *accounts.Service
	board       *board.Service
	collections *collections.Service
	moderation  *moderation.Service
	roleRepo    *repository.Moderation
	store       *storage.Local
	rng         *rand.Rand
}

type demoStats struct {
	Imported        int
	Skipped         int
	Sidecars        int
	MissingSidecars int
	SidecarErrors   int
	NSFW            int
	TagEdits        int
	Favorites       int
	Lists           int
	Reordered       bool
	Hidden          int
	Deleted         int
	Rollbacks       int
	Avatars         int
	Contributions   map[string]int
}

type demoPost struct {
	Post model.Post
}

// Demo fills a fresh development database with accounts, posts and
// history for UI testing:
//
//	kura demo <dir> [dir...]
//
// It refuses unless the database has zero users and the public URL host
// is localhost or 127.0.0.1. Post created_at is stamped by the database;
// the create path does not accept a timestamp, so posts are not backdated.
func Demo(ctx context.Context, cfg config.Config, dirs []string, log *slog.Logger) error {
	if len(dirs) == 0 {
		return fmt.Errorf("kura demo: pass at least one directory")
	}
	hostOK := demoPublicHostAllowed(cfg.PublicURL)
	connectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	conn, err := db.Open(connectCtx, cfg.DatabaseURL)
	cancel()
	if err != nil {
		if !hostOK {
			return evaluateDemoGuard(0, cfg.PublicURL)
		}
		return err
	}
	defer conn.Close() //nolint:errcheck

	accountsDAO := dao.NewAccounts(conn)
	count, countErr := accountsDAO.CountUsers(ctx)
	if countErr != nil {
		if !hostOK {
			return evaluateDemoGuard(0, cfg.PublicURL)
		}
		// A fresh database may not have the users table yet. Migrate only
		// after the host check, then count again before any demo writes.
		if err := db.Migrate(ctx, conn); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
		count, countErr = accountsDAO.CountUsers(ctx)
		if countErr != nil {
			return fmt.Errorf("count users: %w", countErr)
		}
	}
	if err := evaluateDemoGuard(count, cfg.PublicURL); err != nil {
		return err
	}
	if err := db.Migrate(ctx, conn); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	store, err := storage.NewLocal(cfg.MediaDir)
	if err != nil {
		return err
	}
	defer store.Close() //nolint:errcheck
	accountsSvc, err := accounts.NewService(repository.NewAccounts(accountsDAO), cfg.PublicURL, store)
	if err != nil {
		return err
	}
	postsRepo := repository.NewPosts(dao.NewPosts(conn))
	env := &demoEnv{
		log: log, accounts: accountsSvc, board: board.NewService(postsRepo, store),
		collections: collections.NewService(repository.NewCollections(dao.NewCollections(conn))),
		roleRepo:    repository.NewModeration(accountsDAO, postsRepo, dao.NewAudit(conn)),
		store:       store, rng: demoRand(),
	}
	env.moderation = moderation.NewService(env.roleRepo)

	files, err := collectSeedFiles(dirs)
	if err != nil {
		return err
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })

	log.Info("demo creating accounts")
	users, err := createDemoUsers(ctx, env)
	if err != nil {
		return err
	}
	avatars, err := storeDemoAvatars(ctx, env, users, files)
	if err != nil {
		return err
	}
	stats := &demoStats{Avatars: avatars, Contributions: map[string]int{}}
	posts, err := importDemoPosts(ctx, env, users, files, stats)
	if err != nil {
		return err
	}
	if err := rateDemoPosts(ctx, env, users, posts, stats); err != nil {
		return err
	}
	rollbackIdx, err := editDemoPosts(ctx, env, users, posts, stats)
	if err != nil {
		return err
	}
	if err := favoriteDemoPosts(ctx, env, users, posts, stats); err != nil {
		return err
	}
	if err := listDemoPosts(ctx, env, users, posts, stats); err != nil {
		return err
	}
	if err := moderateDemoPosts(ctx, env, users, posts, rollbackIdx, stats); err != nil {
		return err
	}
	if len(posts) > 0 && stats.Rollbacks != 1 {
		return fmt.Errorf("demo: expected one rollback, got %d", stats.Rollbacks)
	}
	enough := demoHidePosts + demoDeletePosts
	if rollbackIdx >= 0 {
		enough++
	}
	if len(posts) >= enough && (stats.Hidden != demoHidePosts || stats.Deleted != demoDeletePosts) {
		return fmt.Errorf("demo: moderation counts hidden=%d deleted=%d", stats.Hidden, stats.Deleted)
	}

	log.Info("demo complete",
		"users", len(users),
		"imported", stats.Imported,
		"skipped", stats.Skipped,
		"sidecars", stats.Sidecars,
		"missing_sidecars", stats.MissingSidecars,
		"sidecar_errors", stats.SidecarErrors,
		"nsfw", stats.NSFW,
		"tag_edits", stats.TagEdits,
		"favorites", stats.Favorites,
		"lists", stats.Lists,
		"reordered", stats.Reordered,
		"hidden", stats.Hidden,
		"deleted", stats.Deleted,
		"rollbacks", stats.Rollbacks,
		"avatars", stats.Avatars,
		"contributions", contributionSummary(stats.Contributions),
	)
	if _, err := fmt.Fprintf(os.Stdout, "%s\n", demoCredentialsLine(users)); err != nil {
		return err
	}
	return nil
}

func demoRand() *rand.Rand { return rand.New(rand.NewPCG(demoSeedHi, demoSeedLo)) }

// evaluateDemoGuard refuses unless both safety conditions hold. Demo
// accounts share a known password, so a populated database or a
// non-loopback public URL is fatal on its own.
func evaluateDemoGuard(userCount int64, publicURL string) error {
	hostOK := demoPublicHostAllowed(publicURL)
	fresh := userCount == 0
	if hostOK && fresh {
		return nil
	}
	parts := make([]string, 0, 2)
	if !hostOK {
		parts = append(parts, fmt.Sprintf("public URL host must be localhost or 127.0.0.1 (got %q)", publicURL))
	}
	if !fresh {
		parts = append(parts, fmt.Sprintf("database must have zero users (found %d)", userCount))
	}
	return fmt.Errorf("kura demo refused: %s", strings.Join(parts, "; "))
}

func demoPublicHostAllowed(publicURL string) bool {
	parsed, err := url.Parse(publicURL)
	if err != nil {
		return false
	}
	switch strings.ToLower(parsed.Hostname()) {
	case "localhost", "127.0.0.1":
		return true
	default:
		return false
	}
}

func createDemoUsers(ctx context.Context, env *demoEnv) ([]model.User, error) {
	users := make([]model.User, 0, len(demoAccounts))
	for _, spec := range demoAccounts {
		user, _, _, err := env.accounts.RegisterWithPassword(ctx, spec.Username, demoPassword, spec.DisplayName)
		if err != nil {
			return nil, fmt.Errorf("demo: register %s: %w", spec.Username, err)
		}
		if _, err := env.accounts.UpdateProfile(ctx, user.ID, model.User{
			DisplayName: spec.DisplayName, Bio: spec.Bio, Location: spec.Location, Website: spec.Website,
		}); err != nil {
			return nil, fmt.Errorf("demo: profile %s: %w", spec.Username, err)
		}
		if err := env.accounts.SetAppearance(ctx, user.ID, spec.Theme, spec.Font); err != nil {
			return nil, fmt.Errorf("demo: appearance %s: %w", spec.Username, err)
		}
		if len(spec.Include) > 0 || len(spec.Exclude) > 0 {
			if err := env.accounts.SetTagPreferences(ctx, user.ID, spec.Include, spec.Exclude); err != nil {
				return nil, fmt.Errorf("demo: tag preferences %s: %w", spec.Username, err)
			}
		}
		if spec.ShowNSFW {
			if err := env.accounts.SetShowNSFW(ctx, user.ID, true, true); err != nil {
				return nil, fmt.Errorf("demo: nsfw preference %s: %w", spec.Username, err)
			}
		}
		users = append(users, user)
	}
	return applyDemoRoles(ctx, env, users)
}

func applyDemoRoles(ctx context.Context, env *demoEnv, users []model.User) ([]model.User, error) {
	astraIdx := -1
	for i, spec := range demoAccounts {
		if spec.Username == "astra" {
			astraIdx = i
			break
		}
	}
	if astraIdx < 0 || astraIdx >= len(users) {
		return nil, fmt.Errorf("demo: astra was not registered")
	}
	astra := users[astraIdx]
	entry := model.ChangeEntry{
		EntityType: "user", EntityID: astra.ID, ActorUsername: demoBootstrapActor, Action: "user.role_changed",
		Before: model.MarshalRoleSnapshot(model.RoleSnapshot{Role: astra.Role}),
		After:  model.MarshalRoleSnapshot(model.RoleSnapshot{Role: model.RoleSuperAdmin}),
	}
	updated, err := env.roleRepo.ApplyUserRole(ctx, astra.ID, model.RoleSuperAdmin, entry)
	if err != nil {
		return nil, fmt.Errorf("demo: bootstrap super admin: %w", err)
	}
	if updated.Role != model.RoleSuperAdmin {
		return nil, fmt.Errorf("demo: astra role is %s after bootstrap", updated.Role)
	}
	users[astraIdx] = updated
	astra = updated
	for i, spec := range demoAccounts {
		if spec.Username == "astra" {
			continue
		}
		if spec.Role != model.RoleMember {
			changed, err := env.moderation.SetRole(ctx, astra, users[i].ID, spec.Role)
			if err != nil {
				return nil, fmt.Errorf("demo: set role %s: %w", spec.Username, err)
			}
			users[i] = changed
		}
		if spec.ExtraRoleHistory {
			// Demote, then promote back, so history shows more than the final role.
			if _, err := env.moderation.SetRole(ctx, astra, users[i].ID, model.RoleModerator); err != nil {
				return nil, fmt.Errorf("demo: demote %s: %w", spec.Username, err)
			}
			changed, err := env.moderation.SetRole(ctx, astra, users[i].ID, spec.Role)
			if err != nil {
				return nil, fmt.Errorf("demo: promote %s: %w", spec.Username, err)
			}
			users[i] = changed
		}
	}
	return users, nil
}

func storeDemoAvatars(ctx context.Context, env *demoEnv, users []model.User, files []seedFile) (int, error) {
	var candidates []seedFile
	for _, file := range files {
		if isDemoAvatar(file.Name) {
			candidates = append(candidates, file)
		}
	}
	stored := 0
	for i, spec := range demoAccounts {
		if err := ctx.Err(); err != nil {
			return stored, err
		}
		if !spec.Avatar || len(candidates) == 0 {
			continue
		}
		var lastErr error
		for try := 0; try < 3 && try < len(candidates); try++ {
			file := candidates[(i+try)%len(candidates)]
			opened, err := os.Open(file.Path) //nolint:gosec // operator-supplied seed path
			if err != nil {
				lastErr = err
				continue
			}
			err = env.accounts.StoreAvatar(ctx, users[i].ID, opened, "")
			closeErr := opened.Close()
			if err == nil {
				err = closeErr
			}
			if err == nil {
				stored++
				lastErr = nil
				break
			}
			lastErr = err
		}
		if lastErr != nil {
			env.log.Warn("demo avatar skipped", "user", spec.Username, "error", lastErr)
		}
	}
	return stored, nil
}

func isDemoAvatar(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
		return true
	default:
		return false
	}
}

func importDemoPosts(ctx context.Context, env *demoEnv, users []model.User, files []seedFile, stats *demoStats) ([]demoPost, error) {
	for _, user := range users {
		stats.Contributions[user.Username] = 0
	}
	owners := assignOwners(len(files), demoAccountWeights(), env.rng)
	posts := make([]demoPost, 0, len(files))
	for i, file := range files {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		owner := users[owners[i]]
		post, skipped, err := importSeedFile(ctx, env.store, env.board, file.Path, file.Name, &owner.ID, func(contentType, sha string, meta board.SeedMeta) []string {
			return demoFileTags(env, stats, file, contentType, sha, meta)
		})
		if err != nil {
			return nil, err
		}
		if skipped {
			stats.Skipped++
			continue
		}
		env.board.RecordCreation(ctx, owner, post)
		stats.Imported++
		stats.Contributions[owner.Username]++
		posts = append(posts, demoPost{Post: post})
		if stats.Imported%50 == 0 {
			env.log.Info("demo import progress", "imported", stats.Imported, "skipped", stats.Skipped)
		}
	}
	return posts, nil
}

func demoFileTags(env *demoEnv, stats *demoStats, file seedFile, contentType, sha string, meta board.SeedMeta) []string {
	sidecar, ok, err := readSidecarTags(file.Path)
	if err != nil {
		env.log.Warn("demo sidecar unreadable", "path", file.Path, "error", err)
		stats.SidecarErrors++
		return tagsForSeedFile(file.Dir, contentType, sha, meta.Tags, nil, false)
	}
	if !ok {
		stats.MissingSidecars++
		return tagsForSeedFile(file.Dir, contentType, sha, meta.Tags, nil, false)
	}
	stats.Sidecars++
	return tagsForSeedFile(file.Dir, contentType, sha, meta.Tags, sidecar, true)
}

func rateDemoPosts(ctx context.Context, env *demoEnv, users []model.User, posts []demoPost, stats *demoStats) error {
	n := demoNSFWCount(len(posts))
	if n == 0 {
		return nil
	}
	fallback, err := demoUser(users, "kiri")
	if err != nil {
		return err
	}
	for _, idx := range pickIndexes(n, len(posts), env.rng) {
		post := posts[idx].Post
		actor, ok := userByID(users, deref(post.OwnerID))
		if !ok {
			actor = fallback
		}
		updated, err := env.board.SetMetaAs(ctx, &actor, post.ID, post.Artist, post.Tags, "nsfw")
		if err != nil {
			return fmt.Errorf("demo: rate post: %w", err)
		}
		posts[idx].Post = updated
		stats.NSFW++
	}
	if stats.NSFW != n {
		return fmt.Errorf("demo: marked %d posts nsfw, want %d", stats.NSFW, n)
	}
	return nil
}

func editDemoPosts(ctx context.Context, env *demoEnv, users []model.User, posts []demoPost, stats *demoStats) (int, error) {
	want := demoTagEdits
	if want > len(posts) {
		want = len(posts)
	}
	rollbackIdx := -1
	if want == 0 {
		return rollbackIdx, nil
	}
	for _, idx := range pickIndexes(len(posts), len(posts), env.rng) {
		if stats.TagEdits >= want {
			break
		}
		post := posts[idx].Post
		artist, tags, ok := demoMetaEdit(post, stats.TagEdits)
		if !ok {
			continue
		}
		actor := users[stats.TagEdits%len(users)]
		updated, err := env.board.SetMetaAs(ctx, &actor, post.ID, artist, tags, post.Rating)
		if err != nil {
			return -1, fmt.Errorf("demo: tag edit: %w", err)
		}
		posts[idx].Post = updated
		stats.TagEdits++
		rollbackIdx = idx
	}
	if stats.TagEdits != want {
		return -1, fmt.Errorf("demo: tag edits %d, want %d", stats.TagEdits, want)
	}
	if err := rollbackDemoEdit(ctx, env, users, posts[rollbackIdx].Post.ID); err != nil {
		return -1, err
	}
	stats.Rollbacks = 1
	return rollbackIdx, nil
}

func rollbackDemoEdit(ctx context.Context, env *demoEnv, users []model.User, postID string) error {
	actor, err := demoUser(users, "kiri")
	if err != nil {
		return err
	}
	entries, err := env.moderation.HistoryForPost(ctx, postID, 20)
	if err != nil {
		return fmt.Errorf("demo: history for rollback: %w", err)
	}
	var historyID string
	for _, entry := range entries {
		if entry.Action != "post.meta_updated" {
			continue
		}
		before, beforeErr := entry.PostBefore()
		after, afterErr := entry.PostAfter()
		if beforeErr != nil || afterErr != nil {
			continue
		}
		if before.Artist != after.Artist || !sameStrings(before.Tags, after.Tags) {
			historyID = entry.ID
			break
		}
	}
	if historyID == "" {
		return fmt.Errorf("demo: tag edit produced no history to roll back")
	}
	if _, err := env.moderation.Rollback(ctx, actor, historyID); err != nil {
		return fmt.Errorf("demo: rollback: %w", err)
	}
	return nil
}

func favoriteDemoPosts(ctx context.Context, env *demoEnv, users []model.User, posts []demoPost, stats *demoStats) error {
	if len(posts) == 0 {
		return nil
	}
	for i, spec := range demoAccounts {
		want := rangedInt(env.rng, spec.FavMin, spec.FavMax)
		if want > len(posts) {
			want = len(posts)
		}
		for _, idx := range pickIndexes(want, len(posts), env.rng) {
			if err := env.collections.Favorite(ctx, users[i].ID, posts[idx].Post.ID); err != nil {
				return fmt.Errorf("demo: favorite for %s: %w", spec.Username, err)
			}
			stats.Favorites++
		}
	}
	return nil
}

func listDemoPosts(ctx context.Context, env *demoEnv, users []model.User, posts []demoPost, stats *demoStats) error {
	if len(posts) == 0 {
		return nil
	}
	for i, spec := range demoAccounts {
		count := rangedInt(env.rng, 1, 4)
		for listIndex := 0; listIndex < count; listIndex++ {
			tmpl := demoListAt(i, listIndex)
			list, err := env.collections.CreateList(ctx, users[i].ID, tmpl.Title, tmpl.Desc, demoListVisibility(i, listIndex))
			if err != nil {
				return fmt.Errorf("demo: list for %s: %w", spec.Username, err)
			}
			stats.Lists++
			want := rangedInt(env.rng, 5, 25)
			if want > len(posts) {
				want = len(posts)
			}
			ids := make([]string, 0, want)
			for _, idx := range pickIndexes(want, len(posts), env.rng) {
				id := posts[idx].Post.ID
				if err := env.collections.AddItem(ctx, users[i].ID, list.ID, id); err != nil {
					return fmt.Errorf("demo: list item: %w", err)
				}
				ids = append(ids, id)
			}
			if stats.Reordered || len(ids) < 2 {
				continue
			}
			rotated := make([]string, len(ids))
			copy(rotated, ids[1:])
			rotated[len(rotated)-1] = ids[0]
			if err := env.collections.Reorder(ctx, users[i].ID, list.ID, rotated); err != nil {
				return fmt.Errorf("demo: reorder list: %w", err)
			}
			stats.Reordered = true
		}
	}
	if len(posts) >= 2 && !stats.Reordered {
		return fmt.Errorf("demo: no list was reordered")
	}
	return nil
}

func moderateDemoPosts(ctx context.Context, env *demoEnv, users []model.User, posts []demoPost, rollbackIdx int, stats *demoStats) error {
	kiri, err := demoUser(users, "kiri")
	if err != nil {
		return err
	}
	mika, err := demoUser(users, "mika")
	if err != nil {
		return err
	}
	noa, err := demoUser(users, "noa")
	if err != nil {
		return err
	}
	if kiri.Role != model.RoleModerator || mika.Role != model.RoleModerator {
		return fmt.Errorf("demo: moderators were not promoted")
	}
	if noa.Role != model.RoleAdmin {
		return fmt.Errorf("demo: noa is %s, want admin", noa.Role)
	}
	hide, del := pickModerationTargets(len(posts), demoHidePosts, demoDeletePosts, rollbackIdx, env.rng)
	for i, idx := range hide {
		actor := kiri
		if i%2 == 1 {
			actor = mika
		}
		if _, err := env.moderation.SetPostHidden(ctx, actor, posts[idx].Post.ID, true, false); err != nil {
			return fmt.Errorf("demo: hide post: %w", err)
		}
		stats.Hidden++
	}
	for _, idx := range del {
		if _, err := env.moderation.SetPostHidden(ctx, noa, posts[idx].Post.ID, true, true); err != nil {
			return fmt.Errorf("demo: delete post: %w", err)
		}
		stats.Deleted++
	}
	return nil
}

func demoUser(users []model.User, username string) (model.User, error) {
	for _, user := range users {
		if user.Username == username {
			return user, nil
		}
	}
	return model.User{}, fmt.Errorf("demo: missing user %s", username)
}

func userByID(users []model.User, id string) (model.User, bool) {
	if id == "" {
		return model.User{}, false
	}
	for _, user := range users {
		if user.ID == id {
			return user, true
		}
	}
	return model.User{}, false
}

func deref(id *string) string {
	if id == nil {
		return ""
	}
	return *id
}

func demoAccountWeights() []int {
	weights := make([]int, len(demoAccounts))
	for i, account := range demoAccounts {
		weights[i] = account.Weight
	}
	return weights
}

func demoNSFWCount(n int) int { return n * 8 / 100 }

func demoListAt(userIndex, listIndex int) demoListTemplate {
	return demoListCatalog[(userIndex*3+listIndex)%len(demoListCatalog)]
}

func demoListVisibility(userIndex, listIndex int) string {
	if (userIndex+listIndex)%2 == 0 {
		return "public"
	}
	return "private"
}

func demoCredentialsLine(users []model.User) string {
	parts := make([]string, len(users))
	for i, user := range users {
		parts[i] = user.Username + " " + string(user.Role)
	}
	return "demo accounts: " + strings.Join(parts, ", ") + "; password: " + demoPassword
}

func contributionSummary(counts map[string]int) string {
	parts := make([]string, len(demoAccounts))
	for i, account := range demoAccounts {
		parts[i] = fmt.Sprintf("%s=%d", account.Username, counts[account.Username])
	}
	return strings.Join(parts, " ")
}

// assignOwners picks an account index per file. Higher weights are chosen
// more often. The same rng seed and weights always yield the same slice.
func assignOwners(n int, weights []int, rng *rand.Rand) []int {
	total := 0
	for _, weight := range weights {
		total += weight
	}
	out := make([]int, n)
	if n == 0 || total <= 0 {
		return out
	}
	for i := range out {
		pick := rng.IntN(total)
		acc := 0
		for j, weight := range weights {
			acc += weight
			if pick < acc {
				out[i] = j
				break
			}
		}
	}
	return out
}

func rangedInt(rng *rand.Rand, lo, hi int) int {
	if hi <= lo {
		return lo
	}
	return lo + rng.IntN(hi-lo+1)
}

func pickIndexes(n, available int, rng *rand.Rand) []int {
	if n <= 0 || available <= 0 {
		return nil
	}
	if n > available {
		n = available
	}
	idx := make([]int, available)
	for i := range idx {
		idx[i] = i
	}
	rng.Shuffle(available, func(i, j int) { idx[i], idx[j] = idx[j], idx[i] })
	return append([]int(nil), idx[:n]...)
}

// pickModerationTargets chooses disjoint hide and delete indexes, skipping
// the post whose tag edit was rolled back (skip < 0 skips nothing).
func pickModerationTargets(n, hideN, deleteN, skip int, rng *rand.Rand) (hide, del []int) {
	pool := make([]int, 0, n)
	for i := 0; i < n; i++ {
		if i == skip {
			continue
		}
		pool = append(pool, i)
	}
	rng.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })
	if hideN > len(pool) {
		hideN = len(pool)
	}
	hide = append([]int(nil), pool[:hideN]...)
	rest := pool[hideN:]
	if deleteN > len(rest) {
		deleteN = len(rest)
	}
	del = append([]int(nil), rest[:deleteN]...)
	return hide, del
}

func demoMetaEdit(post model.Post, n int) (artist string, tags []string, ok bool) {
	for try := 0; try < 3; try++ {
		switch (n + try) % 3 {
		case 0:
			if tags, ok = demoAddTag(post.Tags, n+try); ok {
				return post.Artist, tags, true
			}
		case 1:
			if tags, ok = demoRemoveTag(post.Tags); ok {
				return post.Artist, tags, true
			}
		default:
			if artist, ok = demoSetArtist(post.Artist, n+try); ok {
				return artist, append([]string{}, post.Tags...), true
			}
		}
	}
	return post.Artist, append([]string{}, post.Tags...), false
}

func demoAddTag(tags []string, n int) ([]string, bool) {
	if len(demoAddTags) == 0 {
		return nil, false
	}
	tag := ""
	found := false
	for i := range demoAddTags {
		candidate := demoAddTags[(n+i)%len(demoAddTags)]
		if !demoHasTag(tags, candidate) {
			tag = candidate
			found = true
			break
		}
	}
	if !found {
		return nil, false
	}
	out := append([]string{}, tags...)
	if len(out) >= 30 {
		for i, existing := range out {
			if existing != "seed" && existing != tag {
				out[i] = tag
				return out, true
			}
		}
		return nil, false
	}
	return append(out, tag), true
}

func demoRemoveTag(tags []string) ([]string, bool) {
	drop := -1
	for i, tag := range tags {
		if tag != "seed" {
			drop = i
			break
		}
	}
	if drop < 0 {
		return nil, false
	}
	out := make([]string, 0, len(tags)-1)
	out = append(out, tags[:drop]...)
	out = append(out, tags[drop+1:]...)
	return out, true
}

func demoSetArtist(current string, n int) (string, bool) {
	if len(demoArtists) == 0 {
		return "", false
	}
	for i := range demoArtists {
		candidate := demoArtists[(n+i)%len(demoArtists)]
		if candidate != current {
			return candidate, true
		}
	}
	return "", false
}

func demoHasTag(tags []string, tag string) bool {
	for _, existing := range tags {
		if existing == tag {
			return true
		}
	}
	return false
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
