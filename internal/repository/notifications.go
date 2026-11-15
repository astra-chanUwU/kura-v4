package repository

import (
	"context"

	"github.com/astra-chanUwU/kura-v4/internal/dao"
	"github.com/astra-chanUwU/kura-v4/internal/model"
)

type Notifications struct {
	dao *dao.NotificationsDAO
}

func NewNotifications(d *dao.NotificationsDAO) *Notifications {
	return &Notifications{dao: d}
}

func (r *Notifications) Create(ctx context.Context, recipientID, actorID, kind, postID, commentID string) error {
	return r.dao.Create(ctx, recipientID, actorID, kind, postID, commentID)
}

func (r *Notifications) List(ctx context.Context, recipientID string, limit, offset int32) ([]model.Notification, error) {
	return r.dao.List(ctx, recipientID, limit, offset)
}

func (r *Notifications) Count(ctx context.Context, recipientID string) (int64, error) {
	return r.dao.Count(ctx, recipientID)
}

func (r *Notifications) CountUnread(ctx context.Context, recipientID string) (int64, error) {
	return r.dao.CountUnread(ctx, recipientID)
}

// MarkRead stamps read_at on the given rows for one recipient, together.
func (r *Notifications) MarkRead(ctx context.Context, recipientID string, ids []string) error {
	if len(ids) <= 1 {
		return r.dao.MarkRead(ctx, recipientID, ids)
	}
	tx, err := r.dao.DB().BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	if err := r.dao.WithTx(tx).MarkRead(ctx, recipientID, ids); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *Notifications) MarkAllRead(ctx context.Context, recipientID string) error {
	return r.dao.MarkAllRead(ctx, recipientID)
}

func (r *Notifications) PostOwner(ctx context.Context, postID string) (string, error) {
	owner, err := r.dao.PostOwner(ctx, postID)
	return owner, NotFound(err)
}
