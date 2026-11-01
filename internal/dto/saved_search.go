package dto

// SavedSearch is a private board filter. Mirrored in frontend/src/types/savedSearch.ts.
type SavedSearch struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	IncludeTags []string `json:"include_tags"`
	ExcludeTags []string `json:"exclude_tags"`
	Q           string   `json:"q"`
	Sort        string   `json:"sort"`
	CreatedAt   string   `json:"created_at"`
}

// CreateSavedSearch is the body of POST /me/searches.
type CreateSavedSearch struct {
	Name        string   `json:"name"`
	IncludeTags []string `json:"include_tags"`
	ExcludeTags []string `json:"exclude_tags"`
	Q           string   `json:"q"`
	Sort        string   `json:"sort"`
}
