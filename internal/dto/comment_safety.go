package dto

// Mirrored in frontend/src/types/commentSafety.ts.

type CommentSafetyEvent struct {
	ID               string                  `json:"id"`
	CommentID        string                  `json:"comment_id"`
	PostID           string                  `json:"post_id"`
	AuthorUsername   string                  `json:"author_username"`
	Action           string                  `json:"action"`
	Categories       []string                `json:"categories"`
	Body             string                  `json:"body"`
	Status           string                  `json:"status"`
	CreatedAt        string                  `json:"created_at"`
	ResolvedAt       string                  `json:"resolved_at"`
	ResolverID       string                  `json:"resolver_id"`
	ResolverUsername string                  `json:"resolver_username"`
	Decisions        []CommentSafetyDecision `json:"decisions"`
}

type CommentSafetyDecision struct {
	ActorID       string `json:"actor_id"`
	ActorUsername string `json:"actor_username"`
	Action        string `json:"action"`
	CreatedAt     string `json:"created_at"`
}

type CommentSafetyPage struct {
	Total  int64                `json:"total"`
	Events []CommentSafetyEvent `json:"events"`
}

// CommentSafetyResolver is a moderator who closed at least one review,
// with that person's latest decision. Mirrored in frontend/src/types/commentSafety.ts.
type CommentSafetyResolver struct {
	ID             string   `json:"id"`
	Username       string   `json:"username"`
	Decisions      int64    `json:"decisions"`
	LastStatus     string   `json:"last_status"`
	LastAction     string   `json:"last_action"`
	LastCategories []string `json:"last_categories"`
	LastBody       string   `json:"last_body"`
	LastAt         string   `json:"last_at"`
}

type CommentSafetyResolvers struct {
	Resolvers []CommentSafetyResolver `json:"resolvers"`
}
