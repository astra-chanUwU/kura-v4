package dto

// Mirrored in frontend/src/types/notification.ts.

type Notification struct {
	ID            string `json:"id"`
	Kind          string `json:"kind"`
	PostID        string `json:"post_id"`
	CommentID     string `json:"comment_id,omitempty"`
	ActorUsername string `json:"actor_username"`
	Read          bool   `json:"read"`
	CreatedAt     string `json:"created_at"`
}

type NotificationPage struct {
	Total         int64          `json:"total"`
	Unread        int64          `json:"unread"`
	Notifications []Notification `json:"notifications"`
}

// MarkNotificationsRead is the optional body of POST /notifications/read.
// An empty or omitted id list marks every unread notice for the caller.
type MarkNotificationsRead struct {
	IDs []string `json:"ids"`
}
