package dto

// Mirrored in frontend/src/types/comments.ts.

type Comment struct {
	ID              string `json:"id"`
	PostID          string `json:"post_id"`
	AuthorUsername  string `json:"author_username"`
	AuthorAvatarURL string `json:"author_avatar_url"`
	Body            string `json:"body"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
	CanEdit         bool   `json:"can_edit"`
	CanDelete       bool   `json:"can_delete"`
}

type CommentPage struct {
	Total    int64     `json:"total"`
	Comments []Comment `json:"comments"`
}

type CreateComment struct {
	Body string `json:"body"`
}

type UpdateComment struct {
	Body string `json:"body"`
}
