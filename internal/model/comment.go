package model

import "time"

// Comment is one post comment. CanEdit and CanDelete are not stored;
// the HTTP layer derives them for the signed-in viewer.
type Comment struct {
	ID              string
	PostID          string
	AuthorID        string
	AuthorUsername  string
	AuthorAvatarKey string
	Body            string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// CommentPage is one page of comments on a post, oldest first.
type CommentPage struct {
	Total    int64
	Comments []Comment
}

// CommentPostAccess is the slice of a post used to decide whether a
// viewer may know that post exists.
type CommentPostAccess struct {
	OwnerID string
	Rating  string
	Hidden  bool
	Deleted bool
}
