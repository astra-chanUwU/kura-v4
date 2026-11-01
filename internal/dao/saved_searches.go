package dao

import (
	"context"
	"database/sql"

	"github.com/astra-chanUwU/kura-v4/internal/dao/sqlcgen"
	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/google/uuid"
)

// SavedSearchesDAO is the sqlc wrapper for saved_searches.
// Query methods are generated from internal/dao/queries/saved_searches.sql
// and are not in sqlcgen until make generate runs.
type SavedSearchesDAO struct {
	db *sql.DB
	q  *sqlcgen.Queries
}

func NewSavedSearches(conn *sql.DB) *SavedSearchesDAO {
	return &SavedSearchesDAO{db: conn, q: sqlcgen.New(conn)}
}

func (d *SavedSearchesDAO) WithTx(tx *sql.Tx) *SavedSearchesDAO {
	return &SavedSearchesDAO{db: d.db, q: sqlcgen.New(tx)}
}

func (d *SavedSearchesDAO) DB() *sql.DB { return d.db }

func (d *SavedSearchesDAO) List(ctx context.Context, userID string) ([]model.SavedSearch, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}
	rows, err := d.q.ListSavedSearches(ctx, uid)
	if err != nil {
		return nil, err
	}
	out := make([]model.SavedSearch, 0, len(rows))
	for _, row := range rows {
		out = append(out, savedSearchFrom(row))
	}
	return out, nil
}

func (d *SavedSearchesDAO) Count(ctx context.Context, userID string) (int64, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return 0, err
	}
	return d.q.CountSavedSearches(ctx, uid)
}

func (d *SavedSearchesDAO) Create(ctx context.Context, userID string, in model.SavedSearch) (model.SavedSearch, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return model.SavedSearch{}, err
	}
	row, err := d.q.CreateSavedSearch(ctx, sqlcgen.CreateSavedSearchParams{
		UserID:      uid,
		Name:        in.Name,
		IncludeTags: tagsOrEmpty(in.IncludeTags),
		ExcludeTags: tagsOrEmpty(in.ExcludeTags),
		Query:       in.Query,
		Sort:        in.Sort,
	})
	if err != nil {
		return model.SavedSearch{}, err
	}
	return savedSearchFrom(row), nil
}

func (d *SavedSearchesDAO) Delete(ctx context.Context, userID, id string) (int64, error) {
	sid, err := uuid.Parse(id)
	if err != nil {
		return 0, nil
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return 0, nil
	}
	return d.q.DeleteSavedSearch(ctx, sqlcgen.DeleteSavedSearchParams{ID: sid, UserID: uid})
}

func savedSearchFrom(row sqlcgen.SavedSearch) model.SavedSearch {
	return model.SavedSearch{
		ID:          row.ID.String(),
		UserID:      row.UserID.String(),
		Name:        row.Name,
		IncludeTags: tagsOrEmpty(row.IncludeTags),
		ExcludeTags: tagsOrEmpty(row.ExcludeTags),
		Query:       row.Query,
		Sort:        row.Sort,
		CreatedAt:   row.CreatedAt,
	}
}

func tagsOrEmpty(tags []string) []string {
	if tags == nil {
		return []string{}
	}
	return tags
}
