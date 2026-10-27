package dao

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/astra-chanUwU/kura-v4/internal/dao/sqlcgen"
	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/google/uuid"
)

type AuditDAO struct {
	db *sql.DB
	q  *sqlcgen.Queries
}

func NewAudit(conn *sql.DB) *AuditDAO { return &AuditDAO{db: conn, q: sqlcgen.New(conn)} }
func (d *AuditDAO) WithTx(tx *sql.Tx) *AuditDAO {
	return &AuditDAO{db: d.db, q: sqlcgen.New(tx)}
}
func (d *AuditDAO) DB() *sql.DB { return d.db }

func entryFrom(r sqlcgen.ChangeHistory) model.ChangeEntry {
	e := model.ChangeEntry{ID: r.ID.String(), EntityType: r.EntityType, EntityID: r.EntityID.String(),
		ActorUsername: r.ActorUsername, Action: r.Action, Before: json.RawMessage(r.BeforeState),
		After: json.RawMessage(r.AfterState), CreatedAt: r.CreatedAt}
	if r.ActorID.Valid {
		s := r.ActorID.UUID.String()
		e.ActorID = &s
	}
	if r.RollbackOf.Valid {
		s := r.RollbackOf.UUID.String()
		e.RollbackOf = &s
	}
	return e
}

func (d *AuditDAO) InsertChangeHistory(ctx context.Context, e model.ChangeEntry) (model.ChangeEntry, error) {
	entityID, err := uuid.Parse(e.EntityID)
	if err != nil {
		return model.ChangeEntry{}, err
	}
	params := sqlcgen.InsertChangeHistoryParams{EntityType: e.EntityType, EntityID: entityID,
		ActorUsername: e.ActorUsername, Action: e.Action, BeforeState: e.Before, AfterState: e.After}
	if e.ActorID != nil {
		if id, err := uuid.Parse(*e.ActorID); err == nil {
			params.ActorID = uuid.NullUUID{UUID: id, Valid: true}
		}
	}
	if e.RollbackOf != nil {
		if id, err := uuid.Parse(*e.RollbackOf); err == nil {
			params.RollbackOf = uuid.NullUUID{UUID: id, Valid: true}
		}
	}
	r, err := d.q.InsertChangeHistory(ctx, params)
	if err != nil {
		return model.ChangeEntry{}, err
	}
	return entryFrom(r), nil
}

func (d *AuditDAO) GetChangeHistory(ctx context.Context, id string) (model.ChangeEntry, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return model.ChangeEntry{}, err
	}
	r, err := d.q.GetChangeHistory(ctx, uid)
	if err != nil {
		return model.ChangeEntry{}, err
	}
	return entryFrom(r), nil
}

func (d *AuditDAO) ListChangeHistoryForEntity(ctx context.Context, entityType, entityID string, limit int32) ([]model.ChangeEntry, error) {
	eid, err := uuid.Parse(entityID)
	if err != nil {
		return nil, err
	}
	rows, err := d.q.ListChangeHistoryForEntity(ctx, sqlcgen.ListChangeHistoryForEntityParams{EntityType: entityType, EntityID: eid, Limit: limit})
	if err != nil {
		return nil, err
	}
	out := make([]model.ChangeEntry, 0, len(rows))
	for _, r := range rows {
		out = append(out, entryFrom(r))
	}
	return out, nil
}

func (d *AuditDAO) ListRecentChangeHistory(ctx context.Context, limit int32) ([]model.ChangeEntry, error) {
	rows, err := d.q.ListRecentChangeHistory(ctx, limit)
	if err != nil {
		return nil, err
	}
	out := make([]model.ChangeEntry, 0, len(rows))
	for _, r := range rows {
		out = append(out, entryFrom(r))
	}
	return out, nil
}

func (d *AuditDAO) ListChangeHistoryByActor(ctx context.Context, actorID string, limit int32) ([]model.ChangeEntry, error) {
	uid, err := uuid.Parse(actorID)
	if err != nil {
		return nil, err
	}
	rows, err := d.q.ListChangeHistoryByActor(ctx, sqlcgen.ListChangeHistoryByActorParams{ActorID: uuid.NullUUID{UUID: uid, Valid: true}, Limit: limit})
	if err != nil {
		return nil, err
	}
	out := make([]model.ChangeEntry, 0, len(rows))
	for _, r := range rows {
		out = append(out, entryFrom(r))
	}
	return out, nil
}
