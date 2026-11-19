package dao

import (
	"context"
	"database/sql"
	"sort"
	"time"

	"github.com/astra-chanUwU/kura-v4/internal/dao/sqlcgen"
	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/google/uuid"
)

type CommentSafetyDAO struct {
	db *sql.DB
	q  *sqlcgen.Queries
}

func NewCommentSafety(conn *sql.DB) *CommentSafetyDAO {
	return &CommentSafetyDAO{db: conn, q: sqlcgen.New(conn)}
}

func (d *CommentSafetyDAO) WithTx(tx *sql.Tx) *CommentSafetyDAO {
	return &CommentSafetyDAO{db: d.db, q: sqlcgen.New(tx)}
}

func (d *CommentSafetyDAO) DB() *sql.DB { return d.db }

func optionalUUID(id string) (uuid.NullUUID, error) {
	if id == "" {
		return uuid.NullUUID{}, nil
	}
	parsed, err := uuid.Parse(id)
	if err != nil {
		return uuid.NullUUID{}, err
	}
	return uuid.NullUUID{UUID: parsed, Valid: true}, nil
}

func categoriesOrEmpty(categories []string) []string {
	if categories == nil {
		return []string{}
	}
	return categories
}

func commentSafetyFrom(id uuid.UUID, commentID uuid.NullUUID, postID, authorID uuid.UUID, username, action string, categories []string, body, status string, created time.Time, resolved sql.NullTime, resolver uuid.NullUUID) model.CommentSafetyEvent {
	out := model.CommentSafetyEvent{
		ID: id.String(), PostID: postID.String(), AuthorID: authorID.String(),
		AuthorUsername: username, Action: action, Categories: categoriesOrEmpty(categories),
		Body: body, Status: status, CreatedAt: created, Decisions: []model.CommentSafetyDecision{},
	}
	if commentID.Valid {
		out.CommentID = commentID.UUID.String()
	}
	if resolved.Valid {
		out.ResolvedAt = resolved.Time
	}
	if resolver.Valid {
		out.ResolverID = resolver.UUID.String()
	}
	return out
}

func (d *CommentSafetyDAO) CreateCommentSafetyEvent(ctx context.Context, draft model.CommentSafetyDraft) error {
	postID, err := uuid.Parse(draft.PostID)
	if err != nil {
		return err
	}
	authorID, err := uuid.Parse(draft.AuthorID)
	if err != nil {
		return err
	}
	commentID, err := optionalUUID(draft.CommentID)
	if err != nil {
		return err
	}
	return d.q.CreateCommentSafetyEvent(ctx, sqlcgen.CreateCommentSafetyEventParams{
		CommentID:  commentID,
		PostID:     postID,
		AuthorID:   authorID,
		Action:     draft.Action,
		Categories: categoriesOrEmpty(draft.Categories),
		Body:       draft.Body,
	})
}

func (d *CommentSafetyDAO) ListCommentSafetyEvents(ctx context.Context, filter model.CommentSafetyFilter, limit, offset int32) ([]model.CommentSafetyEvent, error) {
	rows, err := d.q.ListFilteredCommentSafetyEvents(ctx, sqlcgen.ListFilteredCommentSafetyEventsParams{
		Status: filter.Status, Action: filter.Action, Category: filter.Category,
		Query: substringLike(filter.Query), ResolverID: filter.ResolverID, Sort: filter.Sort,
		RowLimit: limit, RowOffset: offset,
	})
	if err != nil {
		return nil, err
	}
	out := make([]model.CommentSafetyEvent, 0, len(rows))
	for _, r := range rows {
		event := commentSafetyFrom(r.ID, r.CommentID, r.PostID, r.AuthorID, r.AuthorUsername, r.Action, r.Categories, r.Body, r.Status, r.CreatedAt, r.ResolvedAt, r.ResolverID)
		event.ResolverUsername = r.ResolverUsername
		out = append(out, event)
	}
	if err := d.attachDecisions(ctx, out); err != nil {
		return nil, err
	}
	return out, nil
}

func (d *CommentSafetyDAO) CountCommentSafetyEvents(ctx context.Context, filter model.CommentSafetyFilter) (int64, error) {
	return d.q.CountFilteredCommentSafetyEvents(ctx, sqlcgen.CountFilteredCommentSafetyEventsParams{
		Status: filter.Status, Action: filter.Action, Category: filter.Category,
		Query: substringLike(filter.Query), ResolverID: filter.ResolverID,
	})
}

func (d *CommentSafetyDAO) ListCommentSafetyResolvers(ctx context.Context) ([]model.CommentSafetyResolver, error) {
	rows, err := d.q.ListCommentSafetyResolvers(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]model.CommentSafetyResolver, 0, len(rows))
	for _, r := range rows {
		if !r.ResolverID.Valid {
			continue
		}
		item := model.CommentSafetyResolver{
			ID: r.ResolverID.UUID.String(), Username: r.Username, Decisions: r.Decisions,
			LastStatus: r.LastStatus, LastAction: r.LastAction,
			LastCategories: categoriesOrEmpty(r.LastCategories), LastBody: r.LastBody,
			LastAt: r.LastAt,
		}
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].LastAt.Equal(out[j].LastAt) {
			return out[i].Username < out[j].Username
		}
		return out[i].LastAt.After(out[j].LastAt)
	})
	return out, nil
}

func resolveEventIDs(id, resolverID string) (uuid.UUID, uuid.NullUUID, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return uuid.UUID{}, uuid.NullUUID{}, sql.ErrNoRows
	}
	resolver, err := uuid.Parse(resolverID)
	if err != nil {
		return uuid.UUID{}, uuid.NullUUID{}, err
	}
	return uid, uuid.NullUUID{UUID: resolver, Valid: true}, nil
}

func (d *CommentSafetyDAO) attachDecisions(ctx context.Context, events []model.CommentSafetyEvent) error {
	if len(events) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, 0, len(events))
	index := make(map[string]int, len(events))
	for i := range events {
		id, err := uuid.Parse(events[i].ID)
		if err != nil {
			continue
		}
		ids = append(ids, id)
		index[events[i].ID] = i
		events[i].Decisions = []model.CommentSafetyDecision{}
	}
	if len(ids) == 0 {
		return nil
	}
	rows, err := d.q.ListCommentSafetyDecisions(ctx, ids)
	if err != nil {
		return err
	}
	for _, row := range rows {
		at, ok := index[row.EventID.String()]
		if !ok {
			continue
		}
		step := model.CommentSafetyDecision{
			ActorUsername: row.ActorUsername, Action: row.Action, CreatedAt: row.CreatedAt,
		}
		if row.ActorID.Valid {
			step.ActorID = row.ActorID.UUID.String()
		}
		events[at].Decisions = append(events[at].Decisions, step)
	}
	return nil
}

func (d *CommentSafetyDAO) withDecisions(ctx context.Context, event model.CommentSafetyEvent) (model.CommentSafetyEvent, error) {
	events := []model.CommentSafetyEvent{event}
	if err := d.attachDecisions(ctx, events); err != nil {
		return model.CommentSafetyEvent{}, err
	}
	return events[0], nil
}

func (d *CommentSafetyDAO) DismissCommentSafetyEvent(ctx context.Context, id, resolverID string) (model.CommentSafetyEvent, error) {
	uid, resolver, err := resolveEventIDs(id, resolverID)
	if err != nil {
		return model.CommentSafetyEvent{}, err
	}
	r, err := d.q.DismissCommentSafetyEvent(ctx, sqlcgen.DismissCommentSafetyEventParams{
		ID: uid, ResolverID: resolver,
	})
	if err != nil {
		return model.CommentSafetyEvent{}, err
	}
	return d.withDecisions(ctx, commentSafetyFrom(r.ID, r.CommentID, r.PostID, r.AuthorID, r.AuthorUsername, r.Action, r.Categories, r.Body, r.Status, r.CreatedAt, r.ResolvedAt, r.ResolverID))
}

func (d *CommentSafetyDAO) RemoveCommentSafetyEvent(ctx context.Context, id, resolverID string) (model.CommentSafetyEvent, error) {
	uid, resolver, err := resolveEventIDs(id, resolverID)
	if err != nil {
		return model.CommentSafetyEvent{}, err
	}
	r, err := d.q.RemoveCommentSafetyEvent(ctx, sqlcgen.RemoveCommentSafetyEventParams{
		ID: uid, ResolverID: resolver,
	})
	if err != nil {
		return model.CommentSafetyEvent{}, err
	}
	return d.withDecisions(ctx, commentSafetyFrom(r.ID, r.CommentID, r.PostID, r.AuthorID, r.AuthorUsername, r.Action, r.Categories, r.Body, r.Status, r.CreatedAt, r.ResolvedAt, r.ResolverID))
}

func (d *CommentSafetyDAO) ReopenCommentSafetyEvent(ctx context.Context, id, actorID string) (model.CommentSafetyEvent, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return model.CommentSafetyEvent{}, sql.ErrNoRows
	}
	actor, err := uuid.Parse(actorID)
	if err != nil {
		return model.CommentSafetyEvent{}, err
	}
	r, err := d.q.ReopenCommentSafetyEvent(ctx, sqlcgen.ReopenCommentSafetyEventParams{
		ID: uid, ActorID: uuid.NullUUID{UUID: actor, Valid: true},
	})
	if err != nil {
		return model.CommentSafetyEvent{}, err
	}
	return d.withDecisions(ctx, commentSafetyFrom(r.ID, r.CommentID, r.PostID, r.AuthorID, r.AuthorUsername, r.Action, r.Categories, r.Body, r.Status, r.CreatedAt, r.ResolvedAt, r.ResolverID))
}

func (d *CommentSafetyDAO) RestoreSafetyComment(ctx context.Context, postID, authorID, body string) (string, error) {
	post, err := uuid.Parse(postID)
	if err != nil {
		return "", err
	}
	author, err := uuid.Parse(authorID)
	if err != nil {
		return "", err
	}
	id, err := d.q.RestoreSafetyComment(ctx, sqlcgen.RestoreSafetyCommentParams{
		PostID: post, AuthorID: author, Body: body,
	})
	if err != nil {
		return "", err
	}
	return id.String(), nil
}

func (d *CommentSafetyDAO) AttachSafetyComment(ctx context.Context, eventID, commentID string) error {
	event, err := uuid.Parse(eventID)
	if err != nil {
		return err
	}
	comment, err := uuid.Parse(commentID)
	if err != nil {
		return err
	}
	return d.q.AttachSafetyComment(ctx, sqlcgen.AttachSafetyCommentParams{
		ID: event, CommentID: uuid.NullUUID{UUID: comment, Valid: true},
	})
}

func (d *CommentSafetyDAO) DeleteSafetyComment(ctx context.Context, id string) (int64, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return 0, err
	}
	return d.q.DeleteSafetyComment(ctx, uid)
}
