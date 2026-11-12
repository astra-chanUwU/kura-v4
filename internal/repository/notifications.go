package repository

import (
	"context"

	"github.com/astra-chanUwU/kura-v4/internal/dao"
)

type Notifications struct{ *dao.NotificationsDAO }

func NewNotifications(d *dao.NotificationsDAO) *Notifications {
	return &Notifications{NotificationsDAO: d}
}

// MarkRead stamps read_at on the given rows for one recipient, together.
func (r *Notifications) MarkRead(ctx context.Context, recipientID string, ids []string) error {
	if len(ids) <= 1 {
		return r.NotificationsDAO.MarkRead(ctx, recipientID, ids)
	}
	tx, err := r.DB().BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	if err := r.WithTx(tx).MarkRead(ctx, recipientID, ids); err != nil {
		return err
	}
	return tx.Commit()
}
