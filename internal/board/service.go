package board

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"strings"

	"github.com/astra-chanUwU/kura-v4/internal/model"
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
	ListPostIndex(ctx context.Context, includeNSFW bool) ([]model.Post, error)
	GetPostBySHA(ctx context.Context, sha string) (model.Post, error)
	CountPostsByOwner(ctx context.Context, ownerID string) (int64, error)
	ListPostsByOwner(ctx context.Context, ownerID string) ([]model.Post, error)
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

// ListByOwner returns a user's own posts, newest first — their
// contributions panel. Callers apply NSFW visibility for the viewer.
func (s *Service) ListByOwner(ctx context.Context, userID string) ([]model.Post, error) {
	posts, err := s.repo.ListPostsByOwner(ctx, userID)
	if posts == nil {
		posts = []model.Post{}
	}
	return posts, err
}

// SearchIndex returns every post's searchable fields for client-side
// fuzzy search. Capped so a huge board cannot produce an unbounded reply.
func (s *Service) SearchIndex(ctx context.Context, includeNSFW bool) ([]model.Post, error) {
	posts, err := s.repo.ListPostIndex(ctx, includeNSFW)
	if err != nil {
		return nil, err
	}
	if posts == nil {
		posts = []model.Post{}
	}
	if len(posts) > 10000 {
		posts = posts[:10000]
	}
	return posts, nil
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
