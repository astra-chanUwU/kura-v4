package model

import "time"

// CommentSafetyDraft is a comment the safety check blocked or flagged.
// An empty CommentID means the comment was blocked before insert.
type CommentSafetyDraft struct {
	CommentID  string // empty when the comment was blocked before insert
	PostID     string
	AuthorID   string
	Action     string // blocked | flagged
	Categories []string
	Body       string
}

// CommentSafetyFilter selects one page of the moderator queue.
// An empty Status, Action, Category, Query, or ResolverID skips that
// predicate. Sort is newest, oldest, or decided (latest decision first).
type CommentSafetyFilter struct {
	Status     string
	Action     string
	Category   string
	Query      string
	ResolverID string
	Sort       string
}

// CommentSafetyEvent is one moderator review row. Dismiss closes it
// and leaves a published comment. Remove closes an open or dismissed
// row and deletes a published comment. Reopen puts a closed row back
// on the queue and republishes a comment that remove deleted.
type CommentSafetyEvent struct {
	ID               string
	CommentID        string // empty if null
	PostID           string
	AuthorID         string
	AuthorUsername   string
	Action           string
	Categories       []string
	Body             string
	Status           string
	CreatedAt        time.Time
	ResolvedAt       time.Time // zero if null
	ResolverID       string    // empty if null
	ResolverUsername string    // empty if null
	Decisions        []CommentSafetyDecision
}

// CommentSafetyDecision is one step in a review's chain. The last step
// is who holds the review now. Earlier steps stay so a later delete
// does not hide who restored it.
type CommentSafetyDecision struct {
	ActorID       string
	ActorUsername string
	Action        string // dismissed | removed | reopened | restored
	CreatedAt     time.Time
}

// CommentSafetyResolver is a moderator who closed at least one review,
// with that person's latest decision.
type CommentSafetyResolver struct {
	ID             string
	Username       string
	Decisions      int64
	LastStatus     string
	LastAction     string
	LastCategories []string
	LastBody       string
	LastAt         time.Time
}

// CommentSafetyPage is one page of safety events.
type CommentSafetyPage struct {
	Total  int64
	Events []CommentSafetyEvent
}
