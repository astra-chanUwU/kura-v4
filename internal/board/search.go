package board

import (
	"context"
	"errors"
	"math"
	"strings"
	"unicode/utf8"

	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/google/uuid"
)

// ErrBadSort is returned when a search sort is not newest, oldest, or favorites.
var ErrBadSort = errString("sort must be newest, oldest, or favorites")

// ErrBadCursor is returned when a search page continues from a post
// that is not in this library.
var ErrBadCursor = errString("after must be a post in this library")

type errString string

func (e errString) Error() string { return string(e) }

// PrepareQuery normalizes a board search the same way the board does:
// tags are booru tokens, a tag in both lists is excluded, free text
// shorter than two characters is ignored, and the page is bounded.
func PrepareQuery(q model.PostQuery) (model.PostQuery, error) {
	q.Tags = NormalizeTags(q.Tags)
	q.Exclude = NormalizeTags(q.Exclude)
	if len(q.Exclude) > 0 {
		skip := map[string]bool{}
		for _, tag := range q.Exclude {
			skip[tag] = true
		}
		kept := q.Tags[:0]
		for _, tag := range q.Tags {
			if !skip[tag] {
				kept = append(kept, tag)
			}
		}
		q.Tags = append([]string{}, kept...)
	}
	q.Q = clipText(strings.ToLower(strings.TrimSpace(q.Q)), 80)
	if utf8.RuneCountInString(q.Q) < 2 {
		q.Q = ""
	}
	q.QLike = likeContains(q.Q)
	q.Artist = clipText(strings.TrimSpace(q.Artist), 80)
	q.ArtistLike = likeContains(strings.ToLower(q.Artist))
	if q.Sort == "" {
		q.Sort = "newest"
	}
	if q.Sort != "newest" && q.Sort != "oldest" && q.Sort != "favorites" {
		return model.PostQuery{}, ErrBadSort
	}
	if q.Limit <= 0 {
		q.Limit = 40
	}
	if q.Limit > 100 {
		q.Limit = 100
	}
	if q.Offset < 0 {
		q.Offset = 0
	}
	if q.Tags == nil {
		q.Tags = []string{}
	}
	if q.Exclude == nil {
		q.Exclude = []string{}
	}
	return q, nil
}

func clipText(s string, maxRunes int) string {
	if utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	return string([]rune(s)[:maxRunes])
}

// likeContains builds a case-insensitive substring pattern. % and _
// in the text stay literal.
func likeContains(s string) string {
	if s == "" {
		return ""
	}
	return "%" + escapeLike(s) + "%"
}

func escapeLike(s string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return replacer.Replace(s)
}

// Contributions counts how many posts a user has added to the board.
func (s *Service) Contributions(ctx context.Context, userID string) (int64, error) {
	return s.repo.CountPostsByOwner(ctx, userID)
}

// EditCount is how many tag, artist, rating, or source saves this person made.
func (s *Service) EditCount(ctx context.Context, userID string) (int64, error) {
	return s.repo.CountMetaEditsByActor(ctx, userID)
}

// PublicByOwner is the first page of posts a profile visitor can see.
// Hidden and deleted posts stay off the public profile.
func (s *Service) PublicByOwner(ctx context.Context, userID string, includeNSFW bool, limit int) ([]model.Post, int64, error) {
	lim, _ := model.PageBounds(limit, 0)
	total, err := s.repo.CountPublicPostsByOwner(ctx, userID, includeNSFW)
	if err != nil {
		return nil, 0, err
	}
	posts, err := s.repo.ListPublicPostsByOwner(ctx, userID, includeNSFW, lim, 0)
	if err != nil {
		return nil, 0, err
	}
	posts, removed := filterPosts(posts, func(p model.Post) bool {
		return publicPost(p, includeNSFW)
	})
	return posts, shrinkTotal(total, removed), nil
}

// ListByOwner returns one page of a user's own posts, newest first.
// Hidden posts stay. NSFW is included only when includeNSFW is set,
// so the page size matches what the viewer is allowed to see.
func (s *Service) ListByOwner(ctx context.Context, userID string, includeNSFW bool, limit, offset int) (model.SearchPage, error) {
	// PageBounds casts to int32. A page past MaxInt32 must stay past the
	// end; wrapping it produces a negative OFFSET and the first page.
	if offset > math.MaxInt32 {
		offset = math.MaxInt32
	}
	lim, off := model.PageBounds(limit, offset)
	total, err := s.repo.CountVisiblePostsByOwner(ctx, userID, includeNSFW)
	if err != nil {
		return model.SearchPage{}, err
	}
	posts, err := s.repo.ListPostsByOwner(ctx, userID, includeNSFW, lim, off)
	if err != nil {
		return model.SearchPage{}, err
	}
	// The owner still sees hidden and deleted uploads. NSFW stays off
	// unless this viewer opted in, and the count drops with the page.
	posts, removed := filterPosts(posts, func(p model.Post) bool {
		return includeNSFW || p.Rating != "nsfw"
	})
	return model.SearchPage{Total: shrinkTotal(total, removed), Posts: posts}, nil
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
		if errors.Is(err, model.ErrNotFound) {
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
	posts, removed := filterPosts(posts, func(p model.Post) bool {
		return publicPost(p, prepared.IncludeNSFW)
	})
	return model.SearchPage{Total: shrinkTotal(total, removed), Posts: posts}, nil
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
	n, err := s.repo.PostNeighbors(ctx, id, prepared)
	if err != nil {
		return model.Neighbors{}, err
	}
	// The sequence is the prepared sort. A neighbor the viewer cannot
	// open is not a link, even if the query returned its id.
	prev, err := s.visibleNeighbor(ctx, n.PrevID, prepared.IncludeNSFW)
	if err != nil {
		return model.Neighbors{}, err
	}
	next, err := s.visibleNeighbor(ctx, n.NextID, prepared.IncludeNSFW)
	if err != nil {
		return model.Neighbors{}, err
	}
	n.PrevID = prev
	n.NextID = next
	return n, nil
}

// Related lists visible posts that share a tag with id, most overlap first.
// No shared tags is an empty slice. The caller has already checked that
// the viewer may see the anchor.
func (s *Service) Related(ctx context.Context, id string, includeNSFW bool, exclude []string, limit int) ([]model.Post, error) {
	if strings.TrimSpace(id) == "" {
		return nil, model.ErrNotFound
	}
	if exclude == nil {
		exclude = []string{}
	}
	posts, err := s.repo.RelatedPosts(ctx, id, includeNSFW, exclude, clampRelated(limit))
	if err != nil {
		if posts == nil {
			posts = []model.Post{}
		}
		return posts, err
	}
	filtered, _ := filterPosts(posts, func(p model.Post) bool {
		return publicPost(p, includeNSFW)
	})
	return filtered, nil
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

// Random returns one visible post matching q. No match is model.ErrNotFound.
func (s *Service) Random(ctx context.Context, q model.PostQuery) (model.Post, error) {
	prepared, err := s.prepareSearch(ctx, q)
	if err != nil {
		return model.Post{}, err
	}
	post, err := s.repo.RandomPost(ctx, prepared)
	if err != nil {
		return model.Post{}, err
	}
	if !publicPost(post, prepared.IncludeNSFW) {
		return model.Post{}, model.ErrNotFound
	}
	return post, nil
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
	// Alias resolution can put one tag on both lists. The excluded name wins,
	// matching the rule PrepareQuery applies before aliases are rewritten.
	prepared.Tags = withoutTags(prepared.Tags, prepared.Exclude)
	return prepared, nil
}

func (s *Service) Get(ctx context.Context, id string) (model.Post, error) {
	p, err := s.repo.GetPost(ctx, id)
	if err != nil {
		// A non-UUID never names a row. Callers that surface the raw
		// parse error would turn a missing post into a server error.
		if _, parseErr := uuid.Parse(id); parseErr != nil {
			return model.Post{}, model.ErrNotFound
		}
		return model.Post{}, err
	}
	return s.ensureDimensions(ctx, p), nil
}

func (s *Service) List(ctx context.Context, limit, offset int32, includeNSFW bool) ([]model.Post, error) {
	// A negative offset is a wrapped page number. The library audit walks
	// this method in full pages and stops when a page is short, so rows
	// are not dropped here. The public list route filters them.
	if offset < 0 {
		offset = 0
	}
	posts, err := s.repo.ListPosts(ctx, limit, offset, includeNSFW)
	if posts == nil {
		posts = []model.Post{}
	}
	return posts, err
}

// publicPost is a row a signed-out viewer, or a member who has not opted
// into NSFW, is allowed to see on the board, related posts, random, and
// a public profile. Hidden and deleted stay off those surfaces.
func publicPost(p model.Post, includeNSFW bool) bool {
	if p.Hidden || p.Deleted {
		return false
	}
	return includeNSFW || p.Rating != "nsfw"
}

func (s *Service) visibleNeighbor(ctx context.Context, id string, includeNSFW bool) (string, error) {
	if id == "" {
		return "", nil
	}
	p, err := s.repo.GetPost(ctx, id)
	if errors.Is(err, model.ErrNotFound) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if !publicPost(p, includeNSFW) {
		return "", nil
	}
	return id, nil
}

func filterPosts(posts []model.Post, keep func(model.Post) bool) ([]model.Post, int) {
	out := make([]model.Post, 0, len(posts))
	removed := 0
	for _, p := range posts {
		if keep(p) {
			out = append(out, p)
		} else {
			removed++
		}
	}
	return out, removed
}

func shrinkTotal(total int64, removed int) int64 {
	if removed <= 0 {
		return total
	}
	n := int64(removed)
	if total < n {
		return 0
	}
	return total - n
}

func withoutTags(tags, skip []string) []string {
	if len(skip) == 0 {
		if tags == nil {
			return []string{}
		}
		return tags
	}
	drop := map[string]bool{}
	for _, tag := range skip {
		drop[tag] = true
	}
	out := make([]string, 0, len(tags))
	for _, tag := range tags {
		if tag == "" || drop[tag] {
			continue
		}
		out = append(out, tag)
	}
	return out
}

func (s *Service) ExistsBySHA(ctx context.Context, sha string) (bool, error) {
	return s.repo.ExistsBySHA(ctx, sha)
}
