package repository

import (
	"context"

	"github.com/astra-chanUwU/kura-v4/internal/dao"
)

type Collections struct{ *dao.CollectionsDAO }

func NewCollections(d *dao.CollectionsDAO) *Collections { return &Collections{CollectionsDAO: d} }

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
