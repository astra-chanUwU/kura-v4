package repository

import (
	"context"

	"github.com/astra-chanUwU/kura-v4/internal/dao"
	"github.com/astra-chanUwU/kura-v4/internal/model"
)

type Collections struct {
	dao *dao.CollectionsDAO
}

func NewCollections(d *dao.CollectionsDAO) *Collections { return &Collections{dao: d} }

func (r *Collections) AddFavorite(ctx context.Context, userID, postID string) error {
	return r.dao.AddFavorite(ctx, userID, postID)
}

func (r *Collections) RemoveFavorite(ctx context.Context, userID, postID string) error {
	return r.dao.RemoveFavorite(ctx, userID, postID)
}

func (r *Collections) FavoriteIDs(ctx context.Context, userID string) ([]string, error) {
	return r.dao.FavoriteIDs(ctx, userID)
}

func (r *Collections) ListFavorites(ctx context.Context, userID string, includeNSFW bool, limit, offset int32) ([]model.Post, error) {
	return r.dao.ListFavorites(ctx, userID, includeNSFW, limit, offset)
}

func (r *Collections) CountFavorites(ctx context.Context, userID string, includeNSFW bool) (int64, error) {
	return r.dao.CountFavorites(ctx, userID, includeNSFW)
}

func (r *Collections) CreateList(ctx context.Context, ownerID, title, desc, vis string) (model.List, error) {
	return r.dao.CreateList(ctx, ownerID, title, desc, vis)
}

func (r *Collections) GetList(ctx context.Context, id string) (model.List, error) {
	list, err := r.dao.GetList(ctx, id)
	return list, NotFound(err)
}

func (r *Collections) ListsByOwner(ctx context.Context, ownerID string, publicOnly bool) ([]model.List, error) {
	return r.dao.ListsByOwner(ctx, ownerID, publicOnly)
}

func (r *Collections) UpdateList(ctx context.Context, id, ownerID, title, desc, vis string) (model.List, error) {
	list, err := r.dao.UpdateList(ctx, id, ownerID, title, desc, vis)
	return list, NotFound(err)
}

func (r *Collections) DeleteList(ctx context.Context, id, ownerID string) (int64, error) {
	return r.dao.DeleteList(ctx, id, ownerID)
}

func (r *Collections) CountListItems(ctx context.Context, listID string, includeNSFW, includeHidden bool, exclude []string) (int64, error) {
	return r.dao.CountListItems(ctx, listID, includeNSFW, includeHidden, exclude)
}

func (r *Collections) ListItemIDs(ctx context.Context, listID string) ([]string, error) {
	return r.dao.ListItemIDs(ctx, listID)
}

func (r *Collections) NextPosition(ctx context.Context, listID string) (int32, error) {
	return r.dao.NextPosition(ctx, listID)
}

// AddItem inserts a list row and bumps the list's updated time together.
func (r *Collections) AddItem(ctx context.Context, listID, postID string, pos int32) error {
	tx, err := r.dao.DB().BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	d := r.dao.WithTx(tx)
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
	tx, err := r.dao.DB().BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	d := r.dao.WithTx(tx)
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
	tx, err := r.dao.DB().BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	d := r.dao.WithTx(tx)
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
	return r.dao.ListItems(ctx, listID, includeNSFW, includeHidden, limit, offset, exclude)
}
