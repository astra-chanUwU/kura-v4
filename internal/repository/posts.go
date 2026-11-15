package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/astra-chanUwU/kura-v4/internal/dao"
	"github.com/astra-chanUwU/kura-v4/internal/model"
)

type Posts struct {
	dao   *dao.PostsDAO
	audit *dao.AuditDAO
}

func NewPosts(d *dao.PostsDAO) *Posts { return &Posts{dao: d, audit: dao.NewAudit(d.DB())} }

// CreatePost stores a post and its tags together. A failed tag write rolls
// the post back, so a successful create never lands on the board without the
// tags the member sent.
func (r *Posts) CreatePost(ctx context.Context, p model.Post, sha string) (model.Post, error) {
	tx, err := r.dao.DB().BeginTx(ctx, nil)
	if err != nil {
		return model.Post{}, err
	}
	defer tx.Rollback() //nolint:errcheck
	d := r.dao.WithTx(tx)
	created, err := d.CreatePost(ctx, p, sha)
	if err != nil {
		return model.Post{}, err
	}
	for _, tag := range p.Tags {
		if err := d.InsertPostTag(ctx, created.ID, tag); err != nil {
			return model.Post{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return model.Post{}, err
	}
	return created, nil
}

// ReplacePostMeta replaces a post's artist, rating, source link, and full tag set atomically.
func (r *Posts) ReplacePostMeta(ctx context.Context, p model.Post) error {
	tx, err := r.dao.DB().BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	d := r.dao.WithTx(tx)
	if err := d.UpdatePostMeta(ctx, p.ID, p.Artist, p.Rating, p.SourceURL); err != nil {
		return err
	}
	if err := d.DeletePostTags(ctx, p.ID); err != nil {
		return err
	}
	for _, tag := range p.Tags {
		if err := d.InsertPostTag(ctx, p.ID, tag); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// ReplacePostMetaAudited is ReplacePostMeta plus a history row, atomically:
// a tag edit and its audit record can never drift apart.
func (r *Posts) ReplacePostMetaAudited(ctx context.Context, p model.Post, entry model.ChangeEntry) error {
	tx, err := r.dao.DB().BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	d := r.dao.WithTx(tx)
	if err := d.UpdatePostMeta(ctx, p.ID, p.Artist, p.Rating, p.SourceURL); err != nil {
		return err
	}
	if err := d.DeletePostTags(ctx, p.ID); err != nil {
		return err
	}
	for _, tag := range p.Tags {
		if err := d.InsertPostTag(ctx, p.ID, tag); err != nil {
			return err
		}
	}
	if _, err := r.audit.WithTx(tx).InsertChangeHistory(ctx, entry); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *Posts) InsertHistory(ctx context.Context, entry model.ChangeEntry) (model.ChangeEntry, error) {
	return r.audit.InsertChangeHistory(ctx, entry)
}

// SetPostHiddenAudited flips visibility and records it atomically.
func (r *Posts) SetPostHiddenAudited(ctx context.Context, postID string, hidden, deleted bool, actorID string, entry model.ChangeEntry) error {
	tx, err := r.dao.DB().BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	if err := r.dao.WithTx(tx).SetPostHidden(ctx, postID, hidden, deleted, actorID); err != nil {
		return err
	}
	if _, err := r.audit.WithTx(tx).InsertChangeHistory(ctx, entry); err != nil {
		return err
	}
	return tx.Commit()
}

// ApplyPostSnapshotAudited restores a snapshot (rollback) and records the
// rollback itself as a new history row, atomically.
func (r *Posts) ApplyPostSnapshotAudited(ctx context.Context, p model.Post, entry model.ChangeEntry) error {
	tx, err := r.dao.DB().BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	d := r.dao.WithTx(tx)
	if err := d.UpdatePostMeta(ctx, p.ID, p.Artist, p.Rating, p.SourceURL); err != nil {
		return err
	}
	if err := d.DeletePostTags(ctx, p.ID); err != nil {
		return err
	}
	for _, tag := range p.Tags {
		if err := d.InsertPostTag(ctx, p.ID, tag); err != nil {
			return err
		}
	}
	actorID := ""
	if entry.ActorID != nil {
		actorID = *entry.ActorID
	}
	if err := d.SetPostHidden(ctx, p.ID, p.Hidden, p.Deleted, actorID); err != nil {
		return err
	}
	if _, err := r.audit.WithTx(tx).InsertChangeHistory(ctx, entry); err != nil {
		return err
	}
	return tx.Commit()
}

// EnsureArtistPage writes the artist wiki row for the tag stored on the post.
// A moderator alias is applied by the service before save, and
// EnsureArtistCategory is called with that stored tag. This does not guess
// a different name when the credit token was rewritten.
func (r *Posts) EnsureArtistPage(ctx context.Context, tag string) error {
	return dao.NewTagPages(r.dao.DB()).EnsureArtistCategory(ctx, tag)
}

func (r *Posts) CountMetaEditsByActor(ctx context.Context, actorID string) (int64, error) {
	return r.audit.CountMetaEditsByActor(ctx, actorID)
}

// LastMetaEditor is the username on the latest post.meta_updated row.
// A missing history row means nobody has retagged the post.
func (r *Posts) LastMetaEditor(ctx context.Context, postID string) (string, error) {
	name, err := r.audit.LastMetaEditor(ctx, postID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return name, err
}

func (r *Posts) GetPost(ctx context.Context, id string) (model.Post, error) {
	post, err := r.dao.GetPost(ctx, id)
	return post, NotFound(err)
}

func (r *Posts) GetPostBySHA(ctx context.Context, sha string) (model.Post, error) {
	post, err := r.dao.GetPostBySHA(ctx, sha)
	return post, NotFound(err)
}

func (r *Posts) RandomPost(ctx context.Context, q model.PostQuery) (model.Post, error) {
	post, err := r.dao.RandomPost(ctx, q)
	return post, NotFound(err)
}

func (r *Posts) ListPosts(ctx context.Context, limit, offset int32, includeNSFW bool) ([]model.Post, error) {
	return r.dao.ListPosts(ctx, limit, offset, includeNSFW)
}

func (r *Posts) ExistsBySHA(ctx context.Context, sha string) (bool, error) {
	return r.dao.ExistsBySHA(ctx, sha)
}

func (r *Posts) SetPostThumbnail(ctx context.Context, id, key string) error {
	return r.dao.SetPostThumbnail(ctx, id, key)
}

func (r *Posts) SetPostSample(ctx context.Context, id, key string) error {
	return r.dao.SetPostSample(ctx, id, key)
}

func (r *Posts) SetPostDimensions(ctx context.Context, id string, width, height int) error {
	return r.dao.SetPostDimensions(ctx, id, width, height)
}

func (r *Posts) SearchPosts(ctx context.Context, q model.PostQuery) ([]model.Post, error) {
	return r.dao.SearchPosts(ctx, q)
}

func (r *Posts) CountBoardMatches(ctx context.Context, q model.PostQuery) (int64, error) {
	return r.dao.CountBoardMatches(ctx, q)
}

func (r *Posts) SuggestTags(ctx context.Context, prefix string, filter model.PostQuery, limit int32) ([]model.TagCount, error) {
	return r.dao.SuggestTags(ctx, prefix, filter, limit)
}

func (r *Posts) PostNeighbors(ctx context.Context, id string, q model.PostQuery) (model.Neighbors, error) {
	return r.dao.PostNeighbors(ctx, id, q)
}

func (r *Posts) CountPostsByOwner(ctx context.Context, ownerID string) (int64, error) {
	return r.dao.CountPostsByOwner(ctx, ownerID)
}

func (r *Posts) CountPostsByOwnerSince(ctx context.Context, ownerID string, within time.Duration) (int64, error) {
	return r.dao.CountPostsByOwnerSince(ctx, ownerID, within)
}

func (r *Posts) CountVisiblePostsByOwner(ctx context.Context, ownerID string, includeNSFW bool) (int64, error) {
	return r.dao.CountVisiblePostsByOwner(ctx, ownerID, includeNSFW)
}

func (r *Posts) ListPostsByOwner(ctx context.Context, ownerID string, includeNSFW bool, limit, offset int32) ([]model.Post, error) {
	return r.dao.ListPostsByOwner(ctx, ownerID, includeNSFW, limit, offset)
}

func (r *Posts) RelatedPosts(ctx context.Context, id string, includeNSFW bool, exclude []string, limit int32) ([]model.Post, error) {
	return r.dao.RelatedPosts(ctx, id, includeNSFW, exclude, limit)
}

func (r *Posts) TagAliasTargets(ctx context.Context, tags []string) (map[string]string, error) {
	return r.dao.TagAliasTargets(ctx, tags)
}

func (r *Posts) ListImplications(ctx context.Context) ([]model.TagImplication, error) {
	return r.dao.ListImplications(ctx)
}

func (r *Posts) CountPublicPostsByOwner(ctx context.Context, ownerID string, includeNSFW bool) (int64, error) {
	return r.dao.CountPublicPostsByOwner(ctx, ownerID, includeNSFW)
}

func (r *Posts) ListPublicPostsByOwner(ctx context.Context, ownerID string, includeNSFW bool, limit, offset int32) ([]model.Post, error) {
	return r.dao.ListPublicPostsByOwner(ctx, ownerID, includeNSFW, limit, offset)
}

func (r *Posts) CountHiddenPosts(ctx context.Context, kind, query, rating string) (int64, error) {
	return r.dao.CountHiddenPosts(ctx, kind, query, rating)
}

func (r *Posts) ListHiddenPosts(ctx context.Context, kind, query, rating, sort string, limit, offset int32) ([]model.Post, error) {
	return r.dao.ListHiddenPosts(ctx, kind, query, rating, sort, limit, offset)
}

func (r *Posts) CountRemovedPosts(ctx context.Context) (int64, int64, error) {
	return r.dao.CountRemovedPosts(ctx)
}
