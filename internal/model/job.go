package model

import "encoding/json"

type Job struct {
	ID      int64
	Kind    string
	Payload json.RawMessage
	Token   string
}
