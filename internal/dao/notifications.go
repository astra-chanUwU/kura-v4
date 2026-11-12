package dao

import (
	"context"
	"database/sql"

	"github.com/astra-chanUwU/kura-v4/internal/dao/sqlcgen"
	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/google/uuid"
)

type NotificationsDAO struct {
	db *sql.DB
	q  *sqlcgen.Queries
}

func NewNotifications(conn *sql.DB) *NotificationsDAO {
	return &NotificationsDAO{db: conn, q: sqlcgen.New(conn)}
}

func (d *NotificationsDAO) WithTx(tx *sql.Tx) *NotificationsDAO {
	return &NotificationsDAO{db: d.db, q: sqlcgen.New(tx)}
}

func (d *NotificationsDAO) DB() *sql.DB { return d.db }

func nullID(id string) (uuid.NullUUID, error) {
	if id == "" {
		return uuid.NullUUID{}, nil
	}
	parsed, err := uuid.Parse(id)
	if err != nil {
		return uuid.NullUUID{}, err
	}
	return uuid.NullUUID{UUID: parsed, Valid: true}, nil
}

func (d *NotificationsDAO) Create(ctx context.Context, recipientID, actorID, kind, postID, commentID string) error {
	recipient, err := uuid.Parse(recipientID)
	if err != nil {
		return err
	}
	post, err := uuid.Parse(postID)
	if err != nil {
		return err
	}
	actor, err := nullID(actorID)
	if err != nil {
		return err
	}
	comment, err := nullID(commentID)
	if err != nil {
		return err
	}
	return d.q.CreateNotification(ctx, sqlcgen.CreateNotificationParams{
		RecipientID: recipient,
		ActorID:     actor,
		Kind:        kind,
		PostID:      post,
		CommentID:   comment,
	})
}

func notificationFrom(row sqlcgen.ListNotificationsRow) model.Notification {
	item := model.Notification{
		ID:            row.ID.String(),
		Kind:          row.Kind,
		PostID:        row.PostID.String(),
		ActorUsername: row.ActorUsername,
		Read:          row.IsRead,
		CreatedAt:     row.CreatedAt,
	}
	if row.CommentID.Valid {
		item.CommentID = row.CommentID.UUID.String()
	}
	return item
}

func (d *NotificationsDAO) List(ctx context.Context, recipientID string, limit, offset int32) ([]model.Notification, error) {
	recipient, err := uuid.Parse(recipientID)
	if err != nil {
		return nil, err
	}
	rows, err := d.q.ListNotifications(ctx, sqlcgen.ListNotificationsParams{
		RecipientID: recipient,
		RowLimit:    limit,
		RowOffset:   offset,
	})
	if err != nil {
		return nil, err
	}
	out := make([]model.Notification, 0, len(rows))
	for _, row := range rows {
		out = append(out, notificationFrom(row))
	}
	return out, nil
}

func (d *NotificationsDAO) Count(ctx context.Context, recipientID string) (int64, error) {
	recipient, err := uuid.Parse(recipientID)
	if err != nil {
		return 0, err
	}
	return d.q.CountNotifications(ctx, recipient)
}

func (d *NotificationsDAO) CountUnread(ctx context.Context, recipientID string) (int64, error) {
	recipient, err := uuid.Parse(recipientID)
	if err != nil {
		return 0, err
	}
	return d.q.CountUnreadNotifications(ctx, recipient)
}

func (d *NotificationsDAO) MarkRead(ctx context.Context, recipientID string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	recipient, err := uuid.Parse(recipientID)
	if err != nil {
		return err
	}
	for _, id := range ids {
		uid, err := uuid.Parse(id)
		if err != nil {
			return err
		}
		if err := d.q.MarkNotificationRead(ctx, sqlcgen.MarkNotificationReadParams{
			ID:          uid,
			RecipientID: recipient,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (d *NotificationsDAO) MarkAllRead(ctx context.Context, recipientID string) error {
	recipient, err := uuid.Parse(recipientID)
	if err != nil {
		return err
	}
	return d.q.MarkAllNotificationsRead(ctx, recipient)
}

func (d *NotificationsDAO) PostOwner(ctx context.Context, postID string) (string, error) {
	uid, err := uuid.Parse(postID)
	if err != nil {
		return "", sql.ErrNoRows
	}
	return d.q.NotificationPostOwner(ctx, uid)
}
