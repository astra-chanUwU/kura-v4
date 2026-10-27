package repository

import (
	"context"

	"github.com/astra-chanUwU/kura-v4/internal/dao"
	"github.com/astra-chanUwU/kura-v4/internal/model"
)

type Posts struct {
	*dao.PostsDAO
	audit *dao.AuditDAO
}

func NewPosts(d *dao.PostsDAO) *Posts { return &Posts{PostsDAO: d, audit: dao.NewAudit(d.DB())} }

// ReplacePostMeta replaces a post's artist and full tag set atomically.
func (r *Posts) ReplacePostMeta(ctx context.Context, p model.Post) error {
	tx, err := r.DB().BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	d := r.WithTx(tx)
	if err := d.UpdatePostArtist(ctx, p.ID, p.Artist, p.Rating); err != nil {
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
	tx, err := r.DB().BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	d := r.WithTx(tx)
	if err := d.UpdatePostArtist(ctx, p.ID, p.Artist, p.Rating); err != nil {
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
func (r *Posts) SetPostHiddenAudited(ctx context.Context, postID string, hidden bool, actorID string, entry model.ChangeEntry) error {
	tx, err := r.DB().BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	if err := r.WithTx(tx).SetPostHidden(ctx, postID, hidden, actorID); err != nil {
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
	tx, err := r.DB().BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	d := r.WithTx(tx)
	if err := d.UpdatePostArtist(ctx, p.ID, p.Artist, p.Rating); err != nil {
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
	if err := d.SetPostHidden(ctx, p.ID, p.Hidden, actorID); err != nil {
		return err
	}
	if _, err := r.audit.WithTx(tx).InsertChangeHistory(ctx, entry); err != nil {
		return err
	}
	return tx.Commit()
}
