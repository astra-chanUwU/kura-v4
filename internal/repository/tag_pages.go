package repository

import (
	"context"

	"github.com/astra-chanUwU/kura-v4/internal/dao"
	"github.com/astra-chanUwU/kura-v4/internal/model"
)

// TagPages stores wiki rows and aliases. Creating an alias also retags posts.
type TagPages struct{ *dao.TagPagesDAO }

func NewTagPages(d *dao.TagPagesDAO) *TagPages { return &TagPages{TagPagesDAO: d} }

// CreateImplication stores one edge and adds each backfill tag to posts that
// already have the antecedent. The caller chooses those tags. The insert and
// the post updates commit together.
func (r *TagPages) CreateImplication(ctx context.Context, antecedent, consequent, userID string, backfill []string) error {
	tx, err := r.DB().BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	d := r.WithTx(tx)
	if err := d.InsertImplication(ctx, antecedent, consequent, userID); err != nil {
		return err
	}
	for _, tag := range backfill {
		if err := d.BackfillImplication(ctx, antecedent, tag); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *TagPages) CreateAlias(ctx context.Context, alias, target, userID string) error {
	tx, err := r.DB().BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	d := r.WithTx(tx)
	if err := d.MovePostTag(ctx, alias, target); err != nil {
		return err
	}
	if err := d.InsertAlias(ctx, alias, target, userID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *TagPages) GetPage(ctx context.Context, tag string) (model.TagWiki, bool, error) {
	return r.TagPagesDAO.GetPage(ctx, tag)
}
