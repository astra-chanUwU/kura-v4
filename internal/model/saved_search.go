package model

import "time"

// SavedSearch is one account's private board filter.
// IncludeTags and ExcludeTags are empty slices, never nil.
type SavedSearch struct {
	ID          string
	UserID      string
	Name        string
	IncludeTags []string
	ExcludeTags []string
	Query       string
	Sort        string
	CreatedAt   time.Time
}
