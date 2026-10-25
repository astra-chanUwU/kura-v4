package dao

import (
	"context"
	"database/sql"

	"github.com/astra-chanUwU/kura-v4/internal/dao/sqlcgen"
	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/google/uuid"
	"time"
)

type CollectionsDAO struct {
	db *sql.DB
	q  *sqlcgen.Queries
}

func NewCollections(conn *sql.DB) *CollectionsDAO {
	return &CollectionsDAO{db: conn, q: sqlcgen.New(conn)}
}
func (d *CollectionsDAO) WithTx(tx *sql.Tx) *CollectionsDAO {
	return &CollectionsDAO{db: d.db, q: sqlcgen.New(tx)}
}
func (d *CollectionsDAO) DB() *sql.DB { return d.db }

func (d *CollectionsDAO) AddFavorite(ctx context.Context, userID, postID string) error {
	u, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	p, err := uuid.Parse(postID)
	if err != nil {
		return err
	}
	return d.q.AddFavorite(ctx, sqlcgen.AddFavoriteParams{UserID: u, PostID: p})
}
func (d *CollectionsDAO) RemoveFavorite(ctx context.Context, userID, postID string) error {
	u, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	p, err := uuid.Parse(postID)
	if err != nil {
		return err
	}
	return d.q.RemoveFavorite(ctx, sqlcgen.RemoveFavoriteParams{UserID: u, PostID: p})
}
func (d *CollectionsDAO) FavoriteIDs(ctx context.Context, userID string) ([]string, error) {
	u, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}
	ids, err := d.q.ListFavoritePostIDs(ctx, u)
	if err != nil {
		return nil, err
	}
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = id.String()
	}
	return out, nil
}
func (d *CollectionsDAO) ListFavorites(ctx context.Context, userID string) ([]model.Post, error) {
	u, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}
	rows, err := d.q.ListFavorites(ctx, u)
	if err != nil {
		return nil, err
	}
	out := make([]model.Post, 0, len(rows))
	for _, r := range rows {
		out = append(out, postFrom(r.ID, r.OwnerID, r.OriginalName, r.StorageKey, r.ContentType, r.SizeBytes, r.CreatedAt, r.OwnerUsername, "", "", "", "", "", 0, 0, r.Rating))
	}
	return out, nil
}

func listFrom(id, owner uuid.UUID, title, desc, vis string, created, updated time.Time, ownerName, ownerDisplay string, count int64) model.List {
	return model.List{ID: id.String(), OwnerID: owner.String(), OwnerUsername: ownerName, OwnerDisplayName: ownerDisplay, Title: title, Description: desc, Visibility: vis, ItemCount: count, CreatedAt: created, UpdatedAt: updated}
}

func (d *CollectionsDAO) CreateList(ctx context.Context, ownerID, title, desc, vis string) (model.List, error) {
	o, err := uuid.Parse(ownerID)
	if err != nil {
		return model.List{}, err
	}
	r, err := d.q.CreateList(ctx, sqlcgen.CreateListParams{OwnerID: o, Title: title, Description: desc, Visibility: vis})
	if err != nil {
		return model.List{}, err
	}
	return listFrom(r.ID, r.OwnerID, r.Title, r.Description, r.Visibility, r.CreatedAt, r.UpdatedAt, "", "", 0), nil
}
func (d *CollectionsDAO) GetList(ctx context.Context, id string) (model.List, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return model.List{}, err
	}
	r, err := d.q.GetList(ctx, uid)
	if err != nil {
		return model.List{}, err
	}
	return listFrom(r.ID, r.OwnerID, r.Title, r.Description, r.Visibility, r.CreatedAt, r.UpdatedAt, r.OwnerUsername, r.OwnerDisplayName, r.ItemCount), nil
}
func (d *CollectionsDAO) ListsByOwner(ctx context.Context, ownerID string, publicOnly bool) ([]model.List, error) {
	o, err := uuid.Parse(ownerID)
	if err != nil {
		return nil, err
	}
	var out []model.List
	if publicOnly {
		rows, err := d.q.ListPublicListsByOwner(ctx, o)
		if err != nil {
			return nil, err
		}
		for _, r := range rows {
			out = append(out, listFrom(r.ID, r.OwnerID, r.Title, r.Description, r.Visibility, r.CreatedAt, r.UpdatedAt, r.OwnerUsername, r.OwnerDisplayName, r.ItemCount))
		}
	} else {
		rows, err := d.q.ListListsByOwner(ctx, o)
		if err != nil {
			return nil, err
		}
		for _, r := range rows {
			out = append(out, listFrom(r.ID, r.OwnerID, r.Title, r.Description, r.Visibility, r.CreatedAt, r.UpdatedAt, r.OwnerUsername, r.OwnerDisplayName, r.ItemCount))
		}
	}
	if out == nil {
		out = []model.List{}
	}
	return out, nil
}
func (d *CollectionsDAO) UpdateList(ctx context.Context, id, ownerID, title, desc, vis string) (model.List, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return model.List{}, err
	}
	o, err := uuid.Parse(ownerID)
	if err != nil {
		return model.List{}, err
	}
	r, err := d.q.UpdateList(ctx, sqlcgen.UpdateListParams{ID: uid, Title: title, Description: desc, Visibility: vis, OwnerID: o})
	if err != nil {
		return model.List{}, err
	}
	return listFrom(r.ID, r.OwnerID, r.Title, r.Description, r.Visibility, r.CreatedAt, r.UpdatedAt, "", "", 0), nil
}
func (d *CollectionsDAO) DeleteList(ctx context.Context, id, ownerID string) (int64, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return 0, err
	}
	o, err := uuid.Parse(ownerID)
	if err != nil {
		return 0, err
	}
	return d.q.DeleteList(ctx, sqlcgen.DeleteListParams{ID: uid, OwnerID: o})
}
func (d *CollectionsDAO) ListItems(ctx context.Context, listID string) ([]model.Post, error) {
	uid, err := uuid.Parse(listID)
	if err != nil {
		return nil, err
	}
	rows, err := d.q.ListItems(ctx, uid)
	if err != nil {
		return nil, err
	}
	out := make([]model.Post, 0, len(rows))
	for _, r := range rows {
		out = append(out, postFrom(r.ID, r.OwnerID, r.OriginalName, r.StorageKey, r.ContentType, r.SizeBytes, r.CreatedAt, r.OwnerUsername, "", "", "", "", "", 0, 0, r.Rating))
	}
	return out, nil
}
func (d *CollectionsDAO) NextPosition(ctx context.Context, listID string) (int32, error) {
	uid, err := uuid.Parse(listID)
	if err != nil {
		return 0, err
	}
	return d.q.NextListPosition(ctx, uid)
}
func (d *CollectionsDAO) AddItem(ctx context.Context, listID, postID string, pos int32) error {
	l, err := uuid.Parse(listID)
	if err != nil {
		return err
	}
	p, err := uuid.Parse(postID)
	if err != nil {
		return err
	}
	return d.q.AddListItem(ctx, sqlcgen.AddListItemParams{ListID: l, PostID: p, Position: pos})
}
func (d *CollectionsDAO) RemoveItem(ctx context.Context, listID, postID string) error {
	l, err := uuid.Parse(listID)
	if err != nil {
		return err
	}
	p, err := uuid.Parse(postID)
	if err != nil {
		return err
	}
	return d.q.RemoveListItem(ctx, sqlcgen.RemoveListItemParams{ListID: l, PostID: p})
}
func (d *CollectionsDAO) SetItemPosition(ctx context.Context, listID, postID string, pos int32) error {
	l, err := uuid.Parse(listID)
	if err != nil {
		return err
	}
	p, err := uuid.Parse(postID)
	if err != nil {
		return err
	}
	return d.q.SetListItemPosition(ctx, sqlcgen.SetListItemPositionParams{ListID: l, PostID: p, Position: pos})
}
func (d *CollectionsDAO) TouchList(ctx context.Context, listID string) error {
	uid, err := uuid.Parse(listID)
	if err != nil {
		return err
	}
	return d.q.TouchList(ctx, uid)
}
