package repository

import "github.com/astra-chanUwU/kura-v4/internal/dao"

// SavedSearches is the concrete store behind searches.Repository.
type SavedSearches struct{ *dao.SavedSearchesDAO }

func NewSavedSearches(d *dao.SavedSearchesDAO) *SavedSearches {
	return &SavedSearches{SavedSearchesDAO: d}
}
