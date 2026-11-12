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
	Implies           []string
	ImpliedBy         []string
	Category          string
	Description       string
	PostCount         int64
	UpdatedByUsername string
	UpdatedAt         time.Time
	Posts             []Post
}

// TagImplication says posts tagged with Antecedent also receive Consequent.
type TagImplication struct {
	Antecedent string
	Consequent string
}

// TagSuggestion is a member's proposal for an alias or an implication.
// Alias: SourceTag is the alternate name, TargetTag is the canonical tag.
// Implication: SourceTag is the antecedent, TargetTag is the consequent.
type TagSuggestion struct {
	ID             string
	Kind           string
	SourceTag      string
	TargetTag      string
	Reason         string
	Status         string
	AuthorUsername string
	CreatedAt      time.Time
}
