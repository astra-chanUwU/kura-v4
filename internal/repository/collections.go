package repository

import (
	"context"

	"github.com/astra-chanUwU/kura-v4/internal/dao"
	"github.com/astra-chanUwU/kura-v4/internal/model"
)

type Collections struct{ *dao.CollectionsDAO }

func NewCollections(d *dao.CollectionsDAO) *Collections { return &Collections{CollectionsDAO: d} }

// AddItem inserts a list row and bumps the list's updated time together.
func (r *Collections) AddItem(ctx context.Context, listID, postID string, pos int32) error {
	tx, err := r.DB().BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	d := r.WithTx(tx)
	if err := d.AddItem(ctx, listID, postID, pos); err != nil {
		return err
	}
	if err := d.TouchList(ctx, listID); err != nil {
		return err
	}
	return tx.Commit()
}

// RemoveItem drops a list row and bumps the list's updated time together.
func (r *Collections) RemoveItem(ctx context.Context, listID, postID string) error {
	tx, err := r.DB().BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	d := r.WithTx(tx)
	if err := d.RemoveItem(ctx, listID, postID); err != nil {
		return err
	}
	if err := d.TouchList(ctx, listID); err != nil {
		return err
	}
	return tx.Commit()
}

// Reorder rewrites positions in one transaction; the (list, position) unique
// constraint is deferrable so intermediate states are allowed.
func (r *Collections) Reorder(ctx context.Context, listID string, postIDs []string) error {
	tx, err := r.DB().BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	d := r.WithTx(tx)
	for i, postID := range postIDs {
		if err := d.SetItemPosition(ctx, listID, postID, int32(i)); err != nil {
			return err
		}
	}
	if err := d.TouchList(ctx, listID); err != nil {
		return err
	}
	return tx.Commit()
}

// ListItems matches the collections service. The DAO keeps exclude last.
func (r *Collections) ListItems(ctx context.Context, listID string, includeNSFW, includeHidden bool, exclude []string, limit, offset int32) ([]model.Post, error) {
	return r.CollectionsDAO.ListItems(ctx, listID, includeNSFW, includeHidden, limit, offset, exclude)
}
