package agentcli

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/astra-chanUwU/kura-v4/internal/board"
	"github.com/astra-chanUwU/kura-v4/internal/model"
)

const postsSubcommands = "usage: kura agent posts <list|get|search|mine|upload|edit|hide|unhide|delete|restore|hidden>"

func (e *Engine) posts(ctx context.Context, args []string) int {
	if len(args) == 0 {
		return e.badUsage("posts", postsSubcommands)
	}
	rest := args[1:]
	switch args[0] {
	case "list":
		return e.postsList(ctx, rest)
	case "get":
		return e.postsGet(ctx, rest)
	case "search":
		return e.postsSearch(ctx, rest)
	case "mine":
		return e.postsMine(ctx, rest)
	case "upload":
		return e.postsUpload(ctx, rest)
	case "edit":
		return e.postsEdit(ctx, rest)
	case "hide":
		return e.postsHide(ctx, rest)
	case "unhide":
		return e.postsUnhide(ctx, rest)
	case "delete":
		return e.postsDelete(ctx, rest)
	case "restore":
		return e.postsRestore(ctx, rest)
	case "hidden":
		return e.postsHidden(ctx, rest)
	default:
		return e.badUsage("posts", postsSubcommands)
	}
}

func (e *Engine) postsList(ctx context.Context, args []string) int {
	const cmd = "posts.list"
	actor, err := e.actor(ctx)
	if err != nil {
		return e.failErr(cmd, err)
	}
	parsed, code, argsOK := e.postsArgs(args, cmd, 0, "limit", "offset")
	if !argsOK {
		return code
	}
	limit, err := parsed.Int("limit", 40, 1, 100)
	if err != nil {
		return e.badUsage(cmd, err.Error())
	}
	offset, err := parsed.Int("offset", 0, 0, 1000000)
	if err != nil {
		return e.badUsage(cmd, err.Error())
	}
	if e.deps.Board == nil {
		return e.fail(cmd, "internal", "Something went wrong. Please try again.")
	}
	posts, err := e.deps.Board.List(ctx, int32(limit), int32(offset), SeesNSFW(actor))
	if err != nil {
		return e.failErr(cmd, err)
	}
	if posts == nil {
		posts = []model.Post{}
	}
	return e.ok(cmd, presentPosts(posts, e.favs(ctx, e.actorID(actor))))
}

func (e *Engine) postsGet(ctx context.Context, args []string) int {
	const cmd = "posts.get"
	actor, err := e.actor(ctx)
	if err != nil {
		return e.failErr(cmd, err)
	}
	parsed, code, argsOK := e.postsArgs(args, cmd, 1)
	if !argsOK {
		return code
	}
	if e.deps.Board == nil {
		return e.fail(cmd, "internal", "Something went wrong. Please try again.")
	}
	post, err := e.deps.Board.Get(ctx, parsed.Pos[0])
	if err != nil {
		return e.failErr(cmd, err)
	}
	if !CanSee(actor, post) {
		return e.fail(cmd, "not_found", "Not found.")
	}
	return e.ok(cmd, presentPost(post, e.favs(ctx, e.actorID(actor))))
}

func (e *Engine) postsSearch(ctx context.Context, args []string) int {
	const cmd = "posts.search"
	actor, err := e.actor(ctx)
	if err != nil {
		return e.failErr(cmd, err)
	}
	parsed, code, argsOK := e.postsArgs(args, cmd, 0, "tags", "exclude", "q", "artist", "sort", "limit", "offset")
	if !argsOK {
		return code
	}
	if !parsed.Has("tags") && !parsed.Has("exclude") && !parsed.Has("q") && !parsed.Has("artist") {
		return e.badUsage(cmd, postsCommandUsage(cmd))
	}
	if parsed.Has("tags") && len(board.NormalizeTags(splitList(parsed.Get("tags")))) == 0 {
		return e.badUsage(cmd, "Give at least one tag.")
	}
	if parsed.Has("exclude") && len(board.NormalizeTags(splitList(parsed.Get("exclude")))) == 0 {
		return e.badUsage(cmd, "Give at least one tag to exclude.")
	}
	limit, err := parsed.Int("limit", 40, 1, 100)
	if err != nil {
		return e.badUsage(cmd, err.Error())
	}
	offset, err := parsed.Int("offset", 0, 0, 1_000_000)
	if err != nil {
		return e.badUsage(cmd, err.Error())
	}
	sort := parsed.Get("sort")
	if parsed.Has("sort") && sort != "newest" && sort != "oldest" {
		return e.badUsage(cmd, "Sort must be newest or oldest.")
	}
	if e.deps.Board == nil {
		return e.fail(cmd, "internal", "Something went wrong. Please try again.")
	}
	page, err := e.deps.Board.Search(ctx, model.PostQuery{
		Tags:        splitList(parsed.Get("tags")),
		Exclude:     splitList(parsed.Get("exclude")),
		Q:           parsed.Get("q"),
		Artist:      parsed.Get("artist"),
		Sort:        sort,
		Limit:       int32(limit),
		Offset:      int32(offset),
		IncludeNSFW: SeesNSFW(actor),
	})
	if err != nil {
		return e.failErr(cmd, err)
	}
	return e.ok(cmd, map[string]any{
		"total": page.Total,
		"posts": presentPosts(page.Posts, e.favs(ctx, e.actorID(actor))),
	})
}

func splitList(value string) []string {
	if strings.TrimSpace(value) == "" {
		return []string{}
	}
	return strings.Split(value, ",")
}

func (e *Engine) postsMine(ctx context.Context, args []string) int {
	const cmd = "posts.mine"
	user, ok := e.requireActor(ctx, cmd)
	if !ok {
		return 1
	}
	_, code, argsOK := e.postsArgs(args, cmd, 0)
	if !argsOK {
		return code
	}
	if e.deps.Board == nil {
		return e.fail(cmd, "internal", "Something went wrong. Please try again.")
	}
	posts, err := e.deps.Board.ListByOwner(ctx, user.ID)
	if err != nil {
		return e.failErr(cmd, err)
	}
	// Hidden posts stay. This is the owner's contribution list.
	posts = filterNSFW(&user, posts)
	if posts == nil {
		posts = []model.Post{}
	}
	return e.ok(cmd, presentPosts(posts, e.favs(ctx, user.ID)))
}

func (e *Engine) postsUpload(ctx context.Context, args []string) int {
	const cmd = "posts.upload"
	user, ok := e.requireActor(ctx, cmd)
	if !ok {
		return 1
	}
	parsed, code, argsOK := e.postsArgs(args, cmd, 1, "artist", "tags", "rating")
	if !argsOK {
		return code
	}
	path := parsed.Pos[0]
	f, err := os.Open(path)
	if err != nil {
		return e.fail(cmd, "validation", "Choose a file to upload.")
	}
	defer f.Close()
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return e.fail(cmd, "validation", "Choose a file to upload.")
	}
	head := make([]byte, 512)
	n, err := f.Read(head)
	if err != nil && !errors.Is(err, io.EOF) {
		return e.failErr(cmd, err)
	}
	name := filepath.Base(path)
	contentType := board.SniffContentType(head[:n], name)
	if !board.IsAllowedMedia(contentType) {
		return e.fail(cmd, "validation", "That file type isn't supported yet. Try an image, GIF or MP4/WebM video.")
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return e.failErr(cmd, err)
	}
	if e.deps.Board == nil {
		return e.fail(cmd, "internal", "Something went wrong. Please try again.")
	}
	key, sha, err := e.deps.Board.Upload(ctx, f)
	if err != nil {
		return e.failErr(cmd, err)
	}
	existing, err := e.deps.Board.GetBySHA(ctx, sha)
	if err == nil {
		// The blob is content-addressed and may already be this post's original.
		return e.failData(cmd, "duplicate_post", "Already on the board.", map[string]any{
			"post": presentPost(existing, e.favs(ctx, user.ID)),
		})
	}
	if !errors.Is(err, board.ErrNotFound) {
		return e.failErr(cmd, err)
	}
	rating := ""
	if parsed.Has("rating") {
		rating = parsed.Get("rating")
		if rating != "sfw" && rating != "nsfw" {
			return e.fail(cmd, "validation", "Rating must be sfw or nsfw.")
		}
	}
	meta := board.ParseSeedName(name)
	artist := meta.Artist
	if parsed.Has("artist") {
		artist = parsed.Get("artist")
	}
	tags := meta.Tags
	if parsed.Has("tags") {
		tags = append(tags, parsed.Get("tags"))
	}
	post, err := e.deps.Board.Create(ctx, model.Post{
		OwnerID:      &user.ID,
		OriginalName: name,
		StorageKey:   key,
		ContentType:  contentType,
		SizeBytes:    info.Size(),
		Artist:       artist,
		Source:       meta.Source,
		SourceID:     meta.SourceID,
		Tags:         tags,
		Rating:       rating,
	}, sha)
	if err != nil {
		// Content-addressed bytes may be shared; do not delete the blob.
		return e.failErr(cmd, err)
	}
	post.OwnerUsername = user.Username
	e.deps.Board.RecordCreation(ctx, user, post)
	return e.ok(cmd, presentPost(post, e.favs(ctx, user.ID)))
}

func (e *Engine) postsEdit(ctx context.Context, args []string) int {
	const cmd = "posts.edit"
	user, ok := e.requireActor(ctx, cmd)
	if !ok {
		return 1
	}
	parsed, code, argsOK := e.postsArgs(args, cmd, 1, "artist", "tags", "rating")
	if !argsOK {
		return code
	}
	if !parsed.Has("artist") && !parsed.Has("tags") && !parsed.Has("rating") {
		return e.badUsage(cmd, postsCommandUsage(cmd))
	}
	if e.deps.Board == nil {
		return e.fail(cmd, "internal", "Something went wrong. Please try again.")
	}
	post, err := e.deps.Board.Get(ctx, parsed.Pos[0])
	if err != nil {
		return e.failErr(cmd, err)
	}
	if !CanSee(&user, post) {
		return e.fail(cmd, "not_found", "Not found.")
	}
	artist, tags, rating := post.Artist, post.Tags, post.Rating
	if parsed.Has("artist") {
		artist = parsed.Get("artist")
	}
	if parsed.Has("tags") {
		tags = board.NormalizeTags([]string{parsed.Get("tags")})
	}
	if parsed.Has("rating") {
		rating = parsed.Get("rating")
		if rating != "sfw" && rating != "nsfw" {
			return e.fail(cmd, "validation", "Rating must be sfw or nsfw.")
		}
	}
	updated, err := e.deps.Board.SetMetaAs(ctx, &user, parsed.Pos[0], artist, tags, rating)
	if err != nil {
		return e.failErr(cmd, err)
	}
	return e.ok(cmd, presentPost(updated, e.favs(ctx, user.ID)))
}

func (e *Engine) postsHide(ctx context.Context, args []string) int {
	return e.postsSetHidden(ctx, args, "posts.hide", "moderator", true, false)
}

func (e *Engine) postsUnhide(ctx context.Context, args []string) int {
	return e.postsSetHidden(ctx, args, "posts.unhide", "moderator", false, false)
}

func (e *Engine) postsRestore(ctx context.Context, args []string) int {
	return e.postsSetHidden(ctx, args, "posts.restore", "moderator", false, false)
}

func (e *Engine) postsDelete(ctx context.Context, args []string) int {
	return e.postsSetHidden(ctx, args, "posts.delete", "admin", true, true)
}

func (e *Engine) postsHidden(ctx context.Context, args []string) int {
	const cmd = "posts.hidden"
	user, ok := e.requireActor(ctx, cmd)
	if !ok {
		return 1
	}
	_, code, argsOK := e.postsArgs(args, cmd, 0)
	if !argsOK {
		return code
	}
	if !e.requireMin(cmd, user, "moderator") {
		return 1
	}
	if e.deps.Moderation == nil {
		return e.fail(cmd, "internal", "Something went wrong. Please try again.")
	}
	posts, err := e.deps.Moderation.ListHiddenPosts(ctx, user)
	if err != nil {
		return e.failErr(cmd, err)
	}
	if posts == nil {
		posts = []model.Post{}
	}
	return e.ok(cmd, presentPosts(posts, map[string]bool{}))
}

func (e *Engine) postsSetHidden(ctx context.Context, args []string, command, minRole string, hidden, deleted bool) int {
	user, ok := e.requireActor(ctx, command)
	if !ok {
		return 1
	}
	parsed, code, argsOK := e.postsArgs(args, command, 1)
	if !argsOK {
		return code
	}
	if !e.requireMin(command, user, minRole) {
		return 1
	}
	if e.deps.Moderation == nil {
		return e.fail(command, "internal", "Something went wrong. Please try again.")
	}
	post, err := e.deps.Moderation.SetPostHidden(ctx, user, parsed.Pos[0], hidden, deleted)
	if err != nil {
		return e.failErr(command, err)
	}
	return e.ok(command, presentPost(post, map[string]bool{}))
}

func (e *Engine) postsArgs(args []string, command string, positionals int, allowed ...string) (Args, int, bool) {
	parsed, err := parseArgs(args)
	if err != nil {
		return Args{}, e.badUsage(command, err.Error()), false
	}
	if err := parsed.Only(allowed...); err != nil || len(parsed.Pos) != positionals {
		return Args{}, e.badUsage(command, postsCommandUsage(command)), false
	}
	return parsed, 0, true
}

func postsCommandUsage(name string) string {
	for _, cmd := range Catalog() {
		if cmd.Name == name {
			return cmd.Usage
		}
	}
	return "kura agent " + name
}
