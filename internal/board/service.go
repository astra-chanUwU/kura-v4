package board

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"time"

	"github.com/astra-chanUwU/kura-v4/internal/model"
)

var ErrNotFound = errors.New("post not found")

// ValidationError is a rejected post field, safe to show to the member.
type ValidationError struct{ Message string }

func (e *ValidationError) Error() string { return e.Message }

type Repository interface {
	CreatePost(ctx context.Context, p model.Post, sha string) (model.Post, error)
	GetPost(ctx context.Context, id string) (model.Post, error)
	ListPosts(ctx context.Context, limit, offset int32, includeNSFW bool) ([]model.Post, error)
	ExistsBySHA(ctx context.Context, sha string) (bool, error)
	SetPostThumbnail(ctx context.Context, id, key string) error
	SetPostSample(ctx context.Context, id, key string) error
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
	// CountPostsByOwnerSince counts posts this owner created inside within.
	// Hidden and soft-deleted rows count; deleting a post does not free a slot.
	CountPostsByOwnerSince(ctx context.Context, ownerID string, within time.Duration) (int64, error)
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
	// CountPublicPostsByOwner counts posts this owner has that a visitor
	// can see: not hidden, not deleted, and NSFW only when includeNSFW.
	CountPublicPostsByOwner(ctx context.Context, ownerID string, includeNSFW bool) (int64, error)
	ListPublicPostsByOwner(ctx context.Context, ownerID string, includeNSFW bool, limit, offset int32) ([]model.Post, error)
	// CountMetaEditsByActor counts post.meta_updated history rows for this person.
	CountMetaEditsByActor(ctx context.Context, actorID string) (int64, error)
	// LastMetaEditor is the username on the latest post.meta_updated row.
	// An empty name means nobody has retagged the post.
	LastMetaEditor(ctx context.Context, postID string) (string, error)
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
	jobs  JobQueue
	log   *slog.Logger
}

func NewService(repo Repository, media Media) *Service { return &Service{repo: repo, media: media} }
