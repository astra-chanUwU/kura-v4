package dto

// Mirrored in frontend/src/types/api.ts.

type PostReport struct {
	ID               string `json:"id"`
	PostID           string `json:"post_id"`
	ThumbnailURL     string `json:"thumbnail_url"`
	ReporterUsername string `json:"reporter_username"`
	Reason           string `json:"reason"`
	Note             string `json:"note"`
	Status           string `json:"status"`
	CreatedAt        string `json:"created_at"`
	ResolvedAt       string `json:"resolved_at,omitempty"`
	ResolverID       string `json:"resolver_id,omitempty"`
}

type PostReportPage struct {
	Total   int64        `json:"total"`
	Reports []PostReport `json:"reports"`
}

type CreatePostReport struct {
	Reason string `json:"reason"`
	Note   string `json:"note"`
}
