package model

import "time"

// TagWiki is the stored description of a canonical tag.
type TagWiki struct {
	Tag               string
	Category          string
	Description       string
	UpdatedByUsername string
	UpdatedAt         time.Time
}

// TagPage is what a visitor sees for a tag or for one of its aliases.
// When AliasOf is set, the description and posts belong to that canonical tag.
type TagPage struct {
	Tag               string
	AliasOf           string
	Aliases           []string
	Category          string
	Description       string
	PostCount         int64
	UpdatedByUsername string
	UpdatedAt         time.Time
	Posts             []Post
}
