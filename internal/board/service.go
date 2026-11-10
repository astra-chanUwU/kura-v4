package board

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"net/url"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/astra-chanUwU/kura-v4/internal/tags"
	"github.com/google/uuid"
)

var ErrNotFound = errors.New("post not found")

const sourceURLMessage = "Source link must be an http or https URL."

// ValidationError is a rejected post field, safe to show to the member.
type ValidationError struct{ Message string }

func (e *ValidationError) Error() string { return e.Message }

func invalid(msg string) error { return &ValidationError{Message: msg} }

type Repository interface {
	CreatePost(ctx context.Context, p model.Post, sha string) (model.Post, error)
	GetPost(ctx context.Context, id string) (model.Post, error)
	ListPosts(ctx context.Context, limit, offset int32, includeNSFW bool) ([]model.Post, error)
	ExistsBySHA(ctx context.Context, sha string) (bool, error)
	SetPostThumbnail(ctx context.Context, id, key string) error
	SetPostDimensions(ctx context.Context, id string, width, height int) error
	ReplacePostMeta(ctx context.Context, p model.Post) error
	ReplacePostMetaAudited(ctx context.Context, p model.Post, entry model.ChangeEntry) error
	InsertHistory(ctx context.Context, entry model.ChangeEntry) (model.ChangeEntry, error)
	SearchPosts(ctx context.Context, q model.PostQuery) ([]model.Post, error)
	CountBoardMatches(ctx context.Context, q model.PostQuery) (int64, error)
	SuggestTags(ctx context.Context, prefix string, filter model.PostQuery, limit int32) ([]model.TagCount, error)
	PostNeighbors(ctx context.Context, id string, q model.PostQuery) (model.Neighbors, error)
	GetPostBySHA(ctx context.Context, sha string) (model.Post, error)
	CountPostsByOwner(ctx context.Context, ownerID string) (int64, error)
	CountVisiblePostsByOwner(ctx context.Context, ownerID string, includeNSFW bool) (int64, error)
	ListPostsByOwner(ctx context.Context, ownerID string, includeNSFW bool, limit, offset int32) ([]model.Post, error)
	RelatedPosts(ctx context.Context, id string, includeNSFW bool, exclude []string, limit int32) ([]model.Post, error)
	RandomPost(ctx context.Context, q model.PostQuery) (model.Post, error)
	// EnsureArtistPage records the artist wiki row for a tag already stored
	// on the post. The service resolves moderator aliases before save.
	EnsureArtistPage(ctx context.Context, tag string) error
	// TagAliasTargets maps an alternate tag name to the canonical tag.
	// Missing names are omitted. An empty input returns an empty map.
	TagAliasTargets(ctx context.Context, tags []string) (map[string]string, error)
	ListImplications(ctx context.Context) ([]model.TagImplication, error)
}

// Media is the original-file store the board reads and writes through.
type Media interface {
	Put(context.Context, io.Reader) (string, error)
	PutHashed(context.Context, io.Reader) (string, string, error)
	Open(context.Context, string) (io.ReadCloser, error)
	Delete(context.Context, string) error
}

type Service struct {
	repo  Repository
	media Media
}

func NewService(repo Repository, media Media) *Service { return &Service{repo: repo, media: media} }

// Upload stores an original under its content hash (identical bytes
// are stored once) and returns the storage key and SHA-256.
func (s *Service) Upload(ctx context.Context, r io.Reader) (string, string, error) {
	return s.media.PutHashed(ctx, r)
}

// Discard removes a stored original, e.g. after a failed post create.
func (s *Service) Discard(ctx context.Context, key string) error {
	return s.media.Delete(ctx, key)
}

// OpenMedia opens the original bytes for a post.
func (s *Service) OpenMedia(ctx context.Context, p model.Post) (io.ReadCloser, error) {
	return s.media.Open(ctx, p.StorageKey)
}

func (s *Service) Create(ctx context.Context, p model.Post, sha string) (model.Post, error) {
	sourceURL, err := normalizeSourceURL(p.SourceURL)
	if err != nil {
		return model.Post{}, err
	}
	p.SourceURL = sourceURL
	p.Tags, err = s.applyAliases(ctx, NormalizeTags(p.Tags))
	if err != nil {
		return model.Post{}, err
	}
	p.Rating = model.NormalizeRating(p.Rating)
	var artistTag string
	p.Tags, artistTag, err = s.attachArtist(ctx, "", p.Tags, p.Artist)
	if err != nil {
		return model.Post{}, err
	}
	p.Tags, err = s.applyImplications(ctx, p.Tags)
	if err != nil {
		return model.Post{}, err
	}
	if (p.Width <= 0 || p.Height <= 0) && s.media != nil && p.StorageKey != "" {
		if w, h, err := s.probeDimensions(ctx, p); err == nil {
			p.Width, p.Height = w, h
		}
	}
	// CreatePost stores the row and its tags in one transaction.
	created, err := s.repo.CreatePost(ctx, p, sha)
	if err != nil {
		return model.Post{}, err
	}
	if err := s.ensureArtistPage(ctx, artistTag); err != nil {
		return model.Post{}, err
	}
	return created, nil
}

// GetBySHA finds the existing post for identical bytes, if any.
func (s *Service) GetBySHA(ctx context.Context, sha string) (model.Post, error) {
	p, err := s.repo.GetPostBySHA(ctx, sha)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Post{}, ErrNotFound
	}
	return p, err
}

// Contributions counts how many posts a user has added to the board.
func (s *Service) Contributions(ctx context.Context, userID string) (int64, error) {
	return s.repo.CountPostsByOwner(ctx, userID)
}

// ListByOwner returns one page of a user's own posts, newest first.
// Hidden posts stay. NSFW is included only when includeNSFW is set,
// so the page size matches what the viewer is allowed to see.
func (s *Service) ListByOwner(ctx context.Context, userID string, includeNSFW bool, limit, offset int) (model.SearchPage, error) {
	lim, off := model.PageBounds(limit, offset)
	total, err := s.repo.CountVisiblePostsByOwner(ctx, userID, includeNSFW)
	if err != nil {
		return model.SearchPage{}, err
	}
	posts, err := s.repo.ListPostsByOwner(ctx, userID, includeNSFW, lim, off)
	if err != nil {
		return model.SearchPage{}, err
	}
	if posts == nil {
		posts = []model.Post{}
	}
	return model.SearchPage{Total: total, Posts: posts}, nil
}

// Search returns one page of public posts matching q, plus the full
// match count. The board pages through this instead of downloading
// every post.
func (s *Service) Search(ctx context.Context, q model.PostQuery) (model.SearchPage, error) {
	prepared, err := s.prepareSearch(ctx, q)
	if err != nil {
		return model.SearchPage{}, err
	}
	if strings.TrimSpace(q.AfterID) != "" {
		if _, err := uuid.Parse(q.AfterID); err != nil {
			return model.SearchPage{}, ErrBadCursor
		}
		post, err := s.repo.GetPost(ctx, q.AfterID)
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, ErrNotFound) {
			return model.SearchPage{}, ErrBadCursor
		}
		if err != nil {
			return model.SearchPage{}, err
		}
		prepared.AfterID = post.ID
		prepared.AfterAt = post.CreatedAt
		prepared.AfterFavorites = int32(post.FavoriteCount)
		prepared.Offset = 0
	}
	total, err := s.repo.CountBoardMatches(ctx, prepared)
	if err != nil {
		return model.SearchPage{}, err
	}
	posts, err := s.repo.SearchPosts(ctx, prepared)
	if err != nil {
		return model.SearchPage{}, err
	}
	if posts == nil {
		posts = []model.Post{}
	}
	return model.SearchPage{Total: total, Posts: posts}, nil
}

// SuggestTags lists tags inside filter, most common first.
// prefix prefers a prefix and then a close spelling. An empty filter
// counts the whole visible board.
func (s *Service) SuggestTags(ctx context.Context, prefix string, filter model.PostQuery, limit int32) ([]model.TagCount, error) {
	prepared, err := s.prepareSearch(ctx, filter)
	if err != nil {
		return nil, err
	}
	prefix = clipText(strings.ToLower(strings.TrimSpace(prefix)), 40)
	if limit <= 0 {
		limit = 8
	}
	if limit > 24 {
		limit = 24
	}
	tags, err := s.repo.SuggestTags(ctx, prefix, prepared, limit)
	if tags == nil {
		tags = []model.TagCount{}
	}
	return tags, err
}

// Neighbors reports where id sits in the same sequence Search would
// return, plus the previous and next post ids.
func (s *Service) Neighbors(ctx context.Context, id string, q model.PostQuery) (model.Neighbors, error) {
	prepared, err := s.prepareSearch(ctx, q)
	if err != nil {
		return model.Neighbors{}, err
	}
	return s.repo.PostNeighbors(ctx, id, prepared)
}

// SetMeta replaces a post's artist, tags, rating, and source link
// (communal tagging: any signed-in user may edit; the controller enforces sign-in).
func (s *Service) SetMeta(ctx context.Context, id, artist string, tags []string, rating, sourceURL string) (model.Post, error) {
	return s.setMeta(ctx, nil, id, artist, tags, rating, sourceURL)
}

// SetMetaAs is SetMeta with an actor: a changed edit is written to the
// post's change history in the same transaction, so it can be rolled back.
func (s *Service) SetMetaAs(ctx context.Context, actor *model.User, id, artist string, tags []string, rating, sourceURL string) (model.Post, error) {
	return s.setMeta(ctx, actor, id, artist, tags, rating, sourceURL)
}

func (s *Service) setMeta(ctx context.Context, actor *model.User, id, artist string, tags []string, rating, sourceURL string) (model.Post, error) {
	sourceURL, err := normalizeSourceURL(sourceURL)
	if err != nil {
		return model.Post{}, err
	}
	post, err := s.Get(ctx, id)
	if err != nil {
		return model.Post{}, err
	}
	before := model.SnapshotOf(post)
	if rating == "sfw" || rating == "nsfw" {
		post.Rating = model.NormalizeRating(rating)
	}
	artist = strings.TrimSpace(artist)
	if len([]rune(artist)) > 80 {
		artist = string([]rune(artist)[:80])
	}
	previousArtist := post.Artist
	post.Artist = artist
	post.SourceURL = sourceURL
	post.Tags, err = s.applyAliases(ctx, NormalizeTags(tags))
	if err != nil {
		return model.Post{}, err
	}
	var artistTag string
	post.Tags, artistTag, err = s.attachArtist(ctx, previousArtist, post.Tags, post.Artist)
	if err != nil {
		return model.Post{}, err
	}
	post.Tags, err = s.applyImplications(ctx, post.Tags)
	if err != nil {
		return model.Post{}, err
	}
	after := model.SnapshotOf(post)
	changed := before.Artist != after.Artist || before.Rating != after.Rating || before.SourceURL != after.SourceURL || !equalTags(before.Tags, after.Tags)
	if actor != nil && changed {
		entry := model.ChangeEntry{EntityType: "post", EntityID: post.ID, ActorID: &actor.ID, ActorUsername: actor.Username,
			Action: "post.meta_updated", Before: model.MarshalPostSnapshot(before), After: model.MarshalPostSnapshot(after)}
		if err := s.repo.ReplacePostMetaAudited(ctx, post, entry); err != nil {
			return model.Post{}, err
		}
		if err := s.ensureArtistPage(ctx, artistTag); err != nil {
			return model.Post{}, err
		}
		return post, nil
	}
	if err := s.repo.ReplacePostMeta(ctx, post); err != nil {
		return model.Post{}, err
	}
	if err := s.ensureArtistPage(ctx, artistTag); err != nil {
		return model.Post{}, err
	}
	return post, nil
}

// Related lists visible posts that share a tag with id, most overlap first.
// No shared tags is an empty slice. The caller has already checked that
// the viewer may see the anchor.
func (s *Service) Related(ctx context.Context, id string, includeNSFW bool, exclude []string, limit int) ([]model.Post, error) {
	if strings.TrimSpace(id) == "" {
		return nil, ErrNotFound
	}
	if exclude == nil {
		exclude = []string{}
	}
	posts, err := s.repo.RelatedPosts(ctx, id, includeNSFW, exclude, clampRelated(limit))
	if posts == nil {
		posts = []model.Post{}
	}
	return posts, err
}

func clampRelated(limit int) int32 {
	if limit <= 0 {
		limit = 12
	}
	if limit > 24 {
		limit = 24
	}
	return int32(limit)
}

// Random returns one visible post matching q. No match is ErrNotFound.
func (s *Service) Random(ctx context.Context, q model.PostQuery) (model.Post, error) {
	prepared, err := s.prepareSearch(ctx, q)
	if err != nil {
		return model.Post{}, err
	}
	post, err := s.repo.RandomPost(ctx, prepared)
	if errors.Is(err, sql.ErrNoRows) || errors.Is(err, ErrNotFound) {
		return model.Post{}, ErrNotFound
	}
	if err != nil {
		return model.Post{}, err
	}
	return post, nil
}

func normalizeSourceURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	if utf8.RuneCountInString(raw) > 300 {
		return "", invalid(sourceURLMessage)
	}
	parsed, err := url.Parse(raw)
	if err != nil || !httpURL(parsed) {
		return "", invalid(sourceURLMessage)
	}
	return raw, nil
}

func httpURL(u *url.URL) bool {
	if u == nil || u.Host == "" || u.Opaque != "" {
		return false
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		return true
	default:
		return false
	}
}

// prepareSearch normalizes a board query and rewrites alias tags to the
// canonical name, so grey_hair and gray_hair search the same posts.
func (s *Service) prepareSearch(ctx context.Context, q model.PostQuery) (model.PostQuery, error) {
	prepared, err := PrepareQuery(q)
	if err != nil {
		return model.PostQuery{}, err
	}
	prepared.Tags, err = s.applyAliases(ctx, prepared.Tags)
	if err != nil {
		return model.PostQuery{}, err
	}
	prepared.Exclude, err = s.applyAliases(ctx, prepared.Exclude)
	if err != nil {
		return model.PostQuery{}, err
	}
	return prepared, nil
}

func (s *Service) applyImplications(ctx context.Context, tags []string) ([]string, error) {
	if len(tags) == 0 {
		return tags, nil
	}
	edges, err := s.repo.ListImplications(ctx)
	if err != nil || len(edges) == 0 {
		if tags == nil {
			tags = []string{}
		}
		return tags, err
	}
	return closeImplications(tags, edges), nil
}

// closeImplications adds every tag implied by names. The walk lives in the tags package.
func closeImplications(names []string, edges []model.TagImplication) []string {
	return tags.CloseImplications(names, edges)
}

// applyAliases replaces alternate names with the canonical tag and drops
// duplicates. Tags that are not aliases stay as they are.
func (s *Service) applyAliases(ctx context.Context, tags []string) ([]string, error) {
	if len(tags) == 0 {
		return tags, nil
	}
	mapped, err := s.repo.TagAliasTargets(ctx, tags)
	if err != nil || len(mapped) == 0 {
		if tags == nil {
			tags = []string{}
		}
		return tags, err
	}
	seen := map[string]bool{}
	out := make([]string, 0, len(tags))
	for _, tag := range tags {
		if next, ok := mapped[tag]; ok && next != "" {
			tag = next
		}
		if tag == "" || seen[tag] {
			continue
		}
		seen[tag] = true
		out = append(out, tag)
	}
	sort.Strings(out)
	return out, nil
}

// attachArtist stores the artist credit as a tag after alias resolution.
// A moderator alias is rewritten to its canonical tag here, before save.
// previousArtist's token, and that token's canonical alias when it differs,
// leave the set. The returned tag is the one stored, or "" when the credit
// has no token. A blank artist only removes the previous tag.
func (s *Service) attachArtist(ctx context.Context, previousArtist string, tags []string, artist string) ([]string, string, error) {
	prevToken := model.ArtistToken(previousArtist)
	nextToken := model.ArtistToken(artist)
	mapped, err := s.artistAliases(ctx, prevToken, nextToken)
	if err != nil {
		return nil, "", err
	}
	prevCanonical := aliasOrToken(prevToken, mapped)
	nextCanonical := aliasOrToken(nextToken, mapped)
	out := model.ApplyArtistTag(previousArtist, tags, artist)
	if nextCanonical != "" && nextCanonical != nextToken {
		out = exchangeTag(out, nextToken, nextCanonical)
	}
	if prevCanonical != "" && prevCanonical != prevToken && prevCanonical != nextCanonical {
		out = dropTag(out, prevCanonical)
	}
	if nextCanonical != "" && !tagPresent(out, nextCanonical) {
		out = append(out, nextCanonical)
		sort.Strings(out)
	}
	if out == nil {
		out = []string{}
	}
	return out, nextCanonical, nil
}

func (s *Service) artistAliases(ctx context.Context, tokens ...string) (map[string]string, error) {
	lookup := make([]string, 0, len(tokens))
	seen := map[string]bool{}
	for _, token := range tokens {
		if token == "" || seen[token] {
			continue
		}
		seen[token] = true
		lookup = append(lookup, token)
	}
	if len(lookup) == 0 {
		return map[string]string{}, nil
	}
	return s.repo.TagAliasTargets(ctx, lookup)
}

func aliasOrToken(token string, mapped map[string]string) string {
	if token == "" {
		return ""
	}
	if next, ok := mapped[token]; ok && next != "" {
		return next
	}
	return token
}

func exchangeTag(tags []string, raw, canonical string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(tags)+1)
	for _, tag := range tags {
		if tag == "" || tag == raw || seen[tag] {
			continue
		}
		seen[tag] = true
		out = append(out, tag)
	}
	if canonical != "" && !seen[canonical] {
		out = append(out, canonical)
	}
	sort.Strings(out)
	return out
}

func dropTag(tags []string, drop string) []string {
	if drop == "" {
		if tags == nil {
			return []string{}
		}
		return tags
	}
	out := make([]string, 0, len(tags))
	seen := map[string]bool{}
	for _, tag := range tags {
		if tag == "" || tag == drop || seen[tag] {
			continue
		}
		seen[tag] = true
		out = append(out, tag)
	}
	sort.Strings(out)
	return out
}

func tagPresent(tags []string, want string) bool {
	for _, tag := range tags {
		if tag == want {
			return true
		}
	}
	return false
}

func (s *Service) ensureArtistPage(ctx context.Context, tag string) error {
	if tag == "" {
		return nil
	}
	return s.repo.EnsureArtistPage(ctx, tag)
}

func equalTags(a, b []string) bool {
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

// RecordCreation writes the post.created history row for a new post.
func (s *Service) RecordCreation(ctx context.Context, actor model.User, post model.Post) {
	empty := model.PostSnapshot{Artist: "", Tags: []string{}, Rating: "sfw", Hidden: false}
	_, _ = s.repo.InsertHistory(ctx, model.ChangeEntry{EntityType: "post", EntityID: post.ID, ActorID: &actor.ID,
		ActorUsername: actor.Username, Action: "post.created", Before: model.MarshalPostSnapshot(empty), After: model.MarshalPostSnapshot(model.SnapshotOf(post))})
}

func (s *Service) Get(ctx context.Context, id string) (model.Post, error) {
	p, err := s.repo.GetPost(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Post{}, ErrNotFound
	}
	if err != nil {
		return p, err
	}
	return s.ensureDimensions(ctx, p), nil
}

func (s *Service) List(ctx context.Context, limit, offset int32, includeNSFW bool) ([]model.Post, error) {
	posts, err := s.repo.ListPosts(ctx, limit, offset, includeNSFW)
	if posts == nil {
		posts = []model.Post{}
	}
	return posts, err
}

func (s *Service) ExistsBySHA(ctx context.Context, sha string) (bool, error) {
	return s.repo.ExistsBySHA(ctx, sha)
}
