package repository

import (
	"context"

	"github.com/astra-chanUwU/kura-v4/internal/dao"
	"github.com/astra-chanUwU/kura-v4/internal/model"
)

// TagPages stores wiki rows and aliases. Creating an alias also retags posts.
type TagPages struct {
	pages *dao.TagPagesDAO
}

func NewTagPages(d *dao.TagPagesDAO) *TagPages { return &TagPages{pages: d} }

// CreateImplication stores one edge and adds each backfill tag to posts that
// already have the antecedent. The caller chooses those tags. The insert and
// the post updates commit together.
func (r *TagPages) CreateImplication(ctx context.Context, antecedent, consequent, userID string, backfill []string) error {
	tx, err := r.pages.DB().BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	d := r.pages.WithTx(tx)
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
	tx, err := r.pages.DB().BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	d := r.pages.WithTx(tx)
	if err := d.MovePostTag(ctx, alias, target); err != nil {
		return err
	}
	if err := d.InsertAlias(ctx, alias, target, userID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *TagPages) GetPage(ctx context.Context, tag string) (model.TagWiki, bool, error) {
	return r.pages.GetPage(ctx, tag)
}

func (r *TagPages) UpsertPage(ctx context.Context, tag, category, description, userID string) error {
	return r.pages.UpsertPage(ctx, tag, category, description, userID)
}

func (r *TagPages) CountPosts(ctx context.Context, tag string, includeNSFW bool, exclude []string) (int64, error) {
	return r.pages.CountPosts(ctx, tag, includeNSFW, exclude)
}

func (r *TagPages) RecentPosts(ctx context.Context, tag string, includeNSFW bool, exclude []string) ([]model.Post, error) {
	return r.pages.RecentPosts(ctx, tag, includeNSFW, exclude)
}

func (r *TagPages) AliasTarget(ctx context.Context, alias string) (string, bool, error) {
	return r.pages.AliasTarget(ctx, alias)
}

func (r *TagPages) AliasesOf(ctx context.Context, target string) ([]string, error) {
	return r.pages.AliasesOf(ctx, target)
}

func (r *TagPages) IsAliasTarget(ctx context.Context, tag string) (bool, error) {
	return r.pages.IsAliasTarget(ctx, tag)
}

func (r *TagPages) DeleteAlias(ctx context.Context, alias, target string) (int64, error) {
	return r.pages.DeleteAlias(ctx, alias, target)
}

func (r *TagPages) ListImplications(ctx context.Context) ([]model.TagImplication, error) {
	return r.pages.ListImplications(ctx)
}

func (r *TagPages) DeleteImplication(ctx context.Context, antecedent, consequent string) (int64, error) {
	return r.pages.DeleteImplication(ctx, antecedent, consequent)
}

func (r *TagPages) CreateSuggestion(ctx context.Context, kind, sourceTag, targetTag, reason, authorID string) (model.TagSuggestion, error) {
	return r.pages.CreateSuggestion(ctx, kind, sourceTag, targetTag, reason, authorID)
}

func (r *TagPages) GetSuggestion(ctx context.Context, id string) (model.TagSuggestion, error) {
	suggestion, err := r.pages.GetSuggestion(ctx, id)
	return suggestion, NotFound(err)
}

func (r *TagPages) ListSuggestions(ctx context.Context, status string) ([]model.TagSuggestion, error) {
	return r.pages.ListSuggestions(ctx, status)
}

func (r *TagPages) ResolveSuggestion(ctx context.Context, id, resolverID, status, fromStatus string) (model.TagSuggestion, error) {
	suggestion, err := r.pages.ResolveSuggestion(ctx, id, resolverID, status, fromStatus)
	return suggestion, NotFound(err)
}
