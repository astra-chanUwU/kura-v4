package repository

import (
	"context"

	"github.com/astra-chanUwU/kura-v4/internal/dao"
	"github.com/astra-chanUwU/kura-v4/internal/model"
)

// SavedSearches is the concrete store behind searches.Repository.
type SavedSearches struct {
	dao *dao.SavedSearchesDAO
}

func NewSavedSearches(d *dao.SavedSearchesDAO) *SavedSearches {
	return &SavedSearches{dao: d}
}

func (r *SavedSearches) List(ctx context.Context, userID string) ([]model.SavedSearch, error) {
	return r.dao.List(ctx, userID)
}

func (r *SavedSearches) Count(ctx context.Context, userID string) (int64, error) {
	return r.dao.Count(ctx, userID)
}

func (r *SavedSearches) Create(ctx context.Context, userID string, in model.SavedSearch) (model.SavedSearch, error) {
	return r.dao.Create(ctx, userID, in)
}

func (r *SavedSearches) Delete(ctx context.Context, userID, id string) (int64, error) {
	return r.dao.Delete(ctx, userID, id)
}
