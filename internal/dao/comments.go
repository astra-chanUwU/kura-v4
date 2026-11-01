package dao

import (
	"context"
	"database/sql"
	"time"

	"github.com/astra-chanUwU/kura-v4/internal/dao/sqlcgen"
	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/google/uuid"
)

type CommentsDAO struct {
	db *sql.DB
	q  *sqlcgen.Queries
}

func NewComments(conn *sql.DB) *CommentsDAO {
	return &CommentsDAO{db: conn, q: sqlcgen.New(conn)}
}

func (d *CommentsDAO) WithTx(tx *sql.Tx) *CommentsDAO {
	return &CommentsDAO{db: d.db, q: sqlcgen.New(tx)}
}

func (d *CommentsDAO) DB() *sql.DB { return d.db }

func commentFrom(id, postID, authorID uuid.UUID, username, avatarKey, body string, created, updated time.Time) model.Comment {
	return model.Comment{
		ID: id.String(), PostID: postID.String(), AuthorID: authorID.String(),
		AuthorUsername: username, AuthorAvatarKey: avatarKey, Body: body,
		CreatedAt: created, UpdatedAt: updated,
	}
}

func (d *CommentsDAO) CreateComment(ctx context.Context, postID, authorID, body string) (model.Comment, error) {
	post, err := uuid.Parse(postID)
	if err != nil {
		return model.Comment{}, err
	}
	author, err := uuid.Parse(authorID)
	if err != nil {
		return model.Comment{}, err
	}
	created, err := d.q.CreateComment(ctx, sqlcgen.CreateCommentParams{PostID: post, AuthorID: author, Body: body})
	if err != nil {
		return model.Comment{}, err
	}
	return d.GetComment(ctx, created.ID.String())
}

func (d *CommentsDAO) GetComment(ctx context.Context, id string) (model.Comment, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return model.Comment{}, err
	}
	r, err := d.q.GetComment(ctx, uid)
	if err != nil {
		return model.Comment{}, err
	}
	return commentFrom(r.ID, r.PostID, r.AuthorID, r.AuthorUsername, r.AuthorAvatarKey, r.Body, r.CreatedAt, r.UpdatedAt), nil
}

func (d *CommentsDAO) ListCommentsByPost(ctx context.Context, postID string, limit, offset int32) ([]model.Comment, error) {
	uid, err := uuid.Parse(postID)
	if err != nil {
		return nil, err
	}
	rows, err := d.q.ListCommentsByPost(ctx, sqlcgen.ListCommentsByPostParams{PostID: uid, RowLimit: limit, RowOffset: offset})
	if err != nil {
		return nil, err
	}
	out := make([]model.Comment, 0, len(rows))
	for _, r := range rows {
		out = append(out, commentFrom(r.ID, r.PostID, r.AuthorID, r.AuthorUsername, r.AuthorAvatarKey, r.Body, r.CreatedAt, r.UpdatedAt))
	}
	return out, nil
}

func (d *CommentsDAO) CountCommentsByPost(ctx context.Context, postID string) (int64, error) {
	uid, err := uuid.Parse(postID)
	if err != nil {
		return 0, err
	}
	return d.q.CountCommentsByPost(ctx, uid)
}

func (d *CommentsDAO) UpdateCommentBody(ctx context.Context, id, body string) (model.Comment, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return model.Comment{}, err
	}
	r, err := d.q.UpdateCommentBody(ctx, sqlcgen.UpdateCommentBodyParams{Body: body, ID: uid})
	if err != nil {
		return model.Comment{}, err
	}
	return commentFrom(r.ID, r.PostID, r.AuthorID, r.AuthorUsername, r.AuthorAvatarKey, r.Body, r.CreatedAt, r.UpdatedAt), nil
}

func (d *CommentsDAO) DeleteComment(ctx context.Context, id string) (int64, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return 0, err
	}
	return d.q.DeleteComment(ctx, uid)
}

func (d *CommentsDAO) CommentPostAccess(ctx context.Context, postID string) (model.CommentPostAccess, error) {
	uid, err := uuid.Parse(postID)
	if err != nil {
		return model.CommentPostAccess{}, err
	}
	r, err := d.q.CommentPostAccess(ctx, uid)
	if err != nil {
		return model.CommentPostAccess{}, err
	}
	access := model.CommentPostAccess{Rating: r.Rating, Hidden: r.Hidden, Deleted: r.Deleted}
	if r.OwnerID.Valid {
		access.OwnerID = r.OwnerID.UUID.String()
	}
	return access, nil
}
