package dto

// Mirrored in frontend/src/types/suggestion.ts.

type TagSuggestion struct {
	ID             string `json:"id"`
	Kind           string `json:"kind"`
	SourceTag      string `json:"source_tag"`
	TargetTag      string `json:"target_tag"`
	Reason         string `json:"reason"`
	Status         string `json:"status"`
	AuthorUsername string `json:"author_username"`
	CreatedAt      string `json:"created_at"`
}

type CreateTagSuggestion struct {
	Kind   string `json:"kind"`
	Other  string `json:"other"`
	Reason string `json:"reason"`
}
