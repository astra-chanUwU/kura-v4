package board

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"strings"

	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/google/uuid"
)

var ErrNotFound = errors.New("post not found")

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
	p.Tags = NormalizeTags(p.Tags)
	if p.Rating != "nsfw" {
		p.Rating = "sfw"
	}
	if (p.Width <= 0 || p.Height <= 0) && s.media != nil && p.StorageKey != "" {
		if w, h, err := s.probeDimensions(ctx, p); err == nil {
			p.Width, p.Height = w, h
		}
	}
	created, err := s.repo.CreatePost(ctx, p, sha)
	if err != nil {
		return model.Post{}, err
	}
	if len(p.Tags) > 0 || p.Artist != "" {
		meta := created
		meta.Tags = p.Tags
		if err := s.repo.ReplacePostMeta(ctx, meta); err == nil {
			created.Tags = p.Tags
		}
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
	prepared, err := PrepareQuery(q)
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
	prepared, err := PrepareQuery(filter)
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
	prepared, err := PrepareQuery(q)
	if err != nil {
		return model.Neighbors{}, err
	}
	return s.repo.PostNeighbors(ctx, id, prepared)
}

// SetMeta replaces a post's artist and tags (communal tagging: any
// signed-in user may tag; the controller enforces sign-in).
func (s *Service) SetMeta(ctx context.Context, id, artist string, tags []string, rating string) (model.Post, error) {
	return s.setMeta(ctx, nil, id, artist, tags, rating)
}

// SetMetaAs is SetMeta with an actor: a changed edit is written to the
// post's change history in the same transaction, so it can be rolled back.
func (s *Service) SetMetaAs(ctx context.Context, actor *model.User, id, artist string, tags []string, rating string) (model.Post, error) {
	return s.setMeta(ctx, actor, id, artist, tags, rating)
}

func (s *Service) setMeta(ctx context.Context, actor *model.User, id, artist string, tags []string, rating string) (model.Post, error) {
	post, err := s.Get(ctx, id)
	if err != nil {
		return model.Post{}, err
	}
	before := model.SnapshotOf(post)
	if rating == "sfw" || rating == "nsfw" {
		post.Rating = rating
	}
	artist = strings.TrimSpace(artist)
	if len([]rune(artist)) > 80 {
		artist = string([]rune(artist)[:80])
	}
	post.Artist = artist
	post.Tags = NormalizeTags(tags)
	after := model.SnapshotOf(post)
	if actor != nil && (before.Artist != after.Artist || before.Rating != after.Rating || !equalTags(before.Tags, after.Tags)) {
		entry := model.ChangeEntry{EntityType: "post", EntityID: post.ID, ActorID: &actor.ID, ActorUsername: actor.Username,
			Action: "post.meta_updated", Before: model.MarshalPostSnapshot(before), After: model.MarshalPostSnapshot(after)}
		if err := s.repo.ReplacePostMetaAudited(ctx, post, entry); err != nil {
			return model.Post{}, err
		}
		return post, nil
	}
	if err := s.repo.ReplacePostMeta(ctx, post); err != nil {
		return model.Post{}, err
	}
	return post, nil
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
