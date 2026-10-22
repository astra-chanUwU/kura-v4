package dto

// These wire types are mirrored deliberately in frontend/src/types/api.ts.
type Site struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type Error struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
}
