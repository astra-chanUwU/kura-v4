package model

import "time"

const (
	// NotificationFavorite is a new favorite on the recipient's post.
	NotificationFavorite = "favorite"
	// NotificationComment is a new comment on the recipient's post.
	NotificationComment = "comment"
)

// Notification is one private notice for the uploader of a post.
// The public favorite list stays private. The row may name the actor,
// because the recipient is allowed to know who favored or commented.
type Notification struct {
	ID            string
	Kind          string
	PostID        string
	CommentID     string
	ActorUsername string
	Read          bool
	CreatedAt     time.Time
}

// NotificationPage is one page of a recipient's notices, newest first,
// plus how many of that recipient's notices are still unread.
type NotificationPage struct {
	Total         int64
	Unread        int64
	Notifications []Notification
}
