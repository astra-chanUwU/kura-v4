package repository

import (
	"context"

	"github.com/astra-chanUwU/kura-v4/internal/dao"
	"github.com/astra-chanUwU/kura-v4/internal/model"
)

type Posts struct{ *dao.PostsDAO }

func NewPosts(d *dao.PostsDAO) *Posts { return &Posts{PostsDAO: d} }

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
