package repository

import (
	"context"

	"github.com/astra-chanUwU/kura-v4/internal/dao"
	"github.com/astra-chanUwU/kura-v4/internal/model"
)

type CommentSafety struct {
	dao *dao.CommentSafetyDAO
}

func NewCommentSafety(d *dao.CommentSafetyDAO) *CommentSafety {
	return &CommentSafety{dao: d}
}

func (r *CommentSafety) Record(ctx context.Context, draft model.CommentSafetyDraft) error {
	return r.dao.CreateCommentSafetyEvent(ctx, draft)
}

func (r *CommentSafety) List(ctx context.Context, filter model.CommentSafetyFilter, limit, offset int32) ([]model.CommentSafetyEvent, error) {
	return r.dao.ListCommentSafetyEvents(ctx, filter, limit, offset)
}

func (r *CommentSafety) Count(ctx context.Context, filter model.CommentSafetyFilter) (int64, error) {
	return r.dao.CountCommentSafetyEvents(ctx, filter)
}

func (r *CommentSafety) Resolvers(ctx context.Context) ([]model.CommentSafetyResolver, error) {
	return r.dao.ListCommentSafetyResolvers(ctx)
}

func (r *CommentSafety) Dismiss(ctx context.Context, id, resolverID string) (model.CommentSafetyEvent, error) {
	event, err := r.dao.DismissCommentSafetyEvent(ctx, id, resolverID)
	return event, NotFound(err)
}

// Remove marks an open or dismissed event removed and, when it still
// points at a comment, deletes that comment in the same transaction.
// A comment that is already gone still lets the event be marked removed.
func (r *CommentSafety) Remove(ctx context.Context, id, resolverID string) (model.CommentSafetyEvent, error) {
	tx, err := r.dao.DB().BeginTx(ctx, nil)
	if err != nil {
		return model.CommentSafetyEvent{}, err
	}
	defer tx.Rollback() //nolint:errcheck
	d := r.dao.WithTx(tx)
	event, err := d.RemoveCommentSafetyEvent(ctx, id, resolverID)
	if err != nil {
		return model.CommentSafetyEvent{}, NotFound(err)
	}
	if event.CommentID != "" {
		if _, err := d.DeleteSafetyComment(ctx, event.CommentID); err != nil {
			return model.CommentSafetyEvent{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return model.CommentSafetyEvent{}, err
	}
	return event, nil
}

// Reopen puts a dismissed or removed event back on the queue. When
// remove deleted a published comment, the stored body is published
// again and linked to the event in the same transaction.
func (r *CommentSafety) Reopen(ctx context.Context, id, actorID string) (model.CommentSafetyEvent, error) {
	tx, err := r.dao.DB().BeginTx(ctx, nil)
	if err != nil {
		return model.CommentSafetyEvent{}, err
	}
	defer tx.Rollback() //nolint:errcheck
	d := r.dao.WithTx(tx)
	event, err := d.ReopenCommentSafetyEvent(ctx, id, actorID)
	if err != nil {
		return model.CommentSafetyEvent{}, NotFound(err)
	}
	if event.Action == "flagged" && event.CommentID == "" {
		commentID, err := d.RestoreSafetyComment(ctx, event.PostID, event.AuthorID, event.Body)
		if err != nil {
			return model.CommentSafetyEvent{}, err
		}
		if err := d.AttachSafetyComment(ctx, event.ID, commentID); err != nil {
			return model.CommentSafetyEvent{}, err
		}
		event.CommentID = commentID
	}
	if err := tx.Commit(); err != nil {
		return model.CommentSafetyEvent{}, err
	}
	return event, nil
}
