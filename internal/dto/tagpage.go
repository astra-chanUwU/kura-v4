package dto

// Mirrored in frontend/src/types/tagPage.ts.

// TagPage is a tag's wiki, the names that point at it, and a strip of posts.
// AliasOf is set when this name is an alternate for another tag.
type TagPage struct {
	Tag               string   `json:"tag"`
	AliasOf           string   `json:"alias_of"`
	Aliases           []string `json:"aliases"`
	Implies           []string `json:"implies"`
	ImpliedBy         []string `json:"implied_by"`
	Category          string   `json:"category"`
	Description       string   `json:"description"`
	PostCount         int64    `json:"post_count"`
	UpdatedByUsername string   `json:"updated_by_username"`
	UpdatedAt         string   `json:"updated_at"`
	Posts             []Post   `json:"posts"`
}

// UpdateTagPage replaces the category and description of a tag.
type UpdateTagPage struct {
	Category    string `json:"category"`
	Description string `json:"description"`
}

// CreateTagAlias points an alternate name at this tag.
type CreateTagAlias struct {
	Alias string `json:"alias"`
}

// CreateTagImplication names the tag that should be added alongside this one.
type CreateTagImplication struct {
	Consequent string `json:"consequent"`
}
