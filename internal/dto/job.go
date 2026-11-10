package dto

// FailedJob is one exhausted background job. Mirrored in frontend/src/types/api.ts.
type FailedJob struct {
	ID          int64  `json:"id"`
	Kind        string `json:"kind"`
	Attempts    int32  `json:"attempts"`
	MaxAttempts int32  `json:"max_attempts"`
	LastError   string `json:"last_error"`
	UpdatedAt   string `json:"updated_at"`
}

// FailedJobPage is one page of failed jobs plus the full failed count.
type FailedJobPage struct {
	Total int64       `json:"total"`
	Jobs  []FailedJob `json:"jobs"`
}
