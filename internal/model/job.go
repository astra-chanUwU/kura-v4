package model

import (
	"encoding/json"
	"time"
)

type Job struct {
	ID      int64
	Kind    string
	Payload json.RawMessage
	Token   string
}

// FailedJob is one exhausted queue row a moderator can inspect or retry.
type FailedJob struct {
	ID          int64
	Kind        string
	Attempts    int32
	MaxAttempts int32
	LastError   string
	UpdatedAt   time.Time
}

// FailedJobPage is one page of failed jobs plus the full failed count.
type FailedJobPage struct {
	Total int64
	Jobs  []FailedJob
}
