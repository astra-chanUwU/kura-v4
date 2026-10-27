package model

import (
	"encoding/json"
	"time"
)

// ChangeEntry is one append-only history row. Before/After hold a
// PostSnapshot (entity_type=post) or RoleSnapshot (entity_type=user).
type ChangeEntry struct {
	ID            string
	EntityType    string // post | user
	EntityID      string
	ActorID       *string
	ActorUsername string
	Action        string
	Before        json.RawMessage
	After         json.RawMessage
	RollbackOf    *string
	CreatedAt     time.Time
}

func (e ChangeEntry) PostBefore() (PostSnapshot, error) { return decodePost(e.Before) }
func (e ChangeEntry) PostAfter() (PostSnapshot, error)  { return decodePost(e.After) }
func (e ChangeEntry) RoleBefore() (RoleSnapshot, error) { return decodeRole(e.Before) }
func (e ChangeEntry) RoleAfter() (RoleSnapshot, error)  { return decodeRole(e.After) }

func decodePost(raw json.RawMessage) (PostSnapshot, error) {
	var s PostSnapshot
	if len(raw) == 0 {
		return s, nil
	}
	err := json.Unmarshal(raw, &s)
	if s.Tags == nil {
		s.Tags = []string{}
	}
	return s, err
}

func decodeRole(raw json.RawMessage) (RoleSnapshot, error) {
	var s RoleSnapshot
	if len(raw) == 0 {
		return s, nil
	}
	err := json.Unmarshal(raw, &s)
	return s, err
}

func MarshalPostSnapshot(s PostSnapshot) json.RawMessage {
	if s.Tags == nil {
		s.Tags = []string{}
	}
	b, _ := json.Marshal(s)
	return b
}

func MarshalRoleSnapshot(s RoleSnapshot) json.RawMessage {
	b, _ := json.Marshal(s)
	return b
}

// SnapshotOf captures a post's rollback-able state.
func SnapshotOf(p Post) PostSnapshot {
	tags := p.Tags
	if tags == nil {
		tags = []string{}
	}
	return PostSnapshot{Artist: p.Artist, Tags: tags, Rating: p.Rating, Hidden: p.Hidden}
}
