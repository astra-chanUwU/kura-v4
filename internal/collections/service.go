package collections

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/astra-chanUwU/kura-v4/internal/model"
)

var (
	ErrNotFound  = errors.New("not found")
	ErrForbidden = errors.New("not yours")
)

type ValidationError struct{ Message string }

func (e *ValidationError) Error() string { return e.Message }

type Repository interface {
	AddFavorite(ctx context.Context, userID, postID string) error
	RemoveFavorite(ctx context.Context, userID, postID string) error
	FavoriteIDs(ctx context.Context, userID string) ([]string, error)
	ListFavorites(ctx context.Context, userID string, includeNSFW bool, limit, offset int32) ([]model.Post, error)
	CountFavorites(ctx context.Context, userID string, includeNSFW bool) (int64, error)
	CreateList(ctx context.Context, ownerID, title, desc, vis string) (model.List, error)
	GetList(ctx context.Context, id string) (model.List, error)
	ListsByOwner(ctx context.Context, ownerID string, publicOnly bool) ([]model.List, error)
	UpdateList(ctx context.Context, id, ownerID, title, desc, vis string) (model.List, error)
	DeleteList(ctx context.Context, id, ownerID string) (int64, error)
	ListItems(ctx context.Context, listID string, includeNSFW bool, limit, offset int32) ([]model.Post, error)
	CountListItems(ctx context.Context, listID string, includeNSFW bool) (int64, error)
	ListItemIDs(ctx context.Context, listID string) ([]string, error)
	NextPosition(ctx context.Context, listID string) (int32, error)
	AddItem(ctx context.Context, listID, postID string, pos int32) error
	RemoveItem(ctx context.Context, listID, postID string) error
	TouchList(ctx context.Context, listID string) error
	Reorder(ctx context.Context, listID string, postIDs []string) error
}

type Service struct{ repo Repository }

func NewService(repo Repository) *Service { return &Service{repo: repo} }

// Favorites are private: every method here is scoped to the signed-in owner.

func (s *Service) Favorite(ctx context.Context, userID, postID string) error {
	return s.repo.AddFavorite(ctx, userID, postID)
}
func (s *Service) Unfavorite(ctx context.Context, userID, postID string) error {
	return s.repo.RemoveFavorite(ctx, userID, postID)
}
func (s *Service) Favorites(ctx context.Context, userID string, includeNSFW bool, limit, offset int) (model.SearchPage, error) {
	lim, off := model.PageBounds(limit, offset)
	total, err := s.repo.CountFavorites(ctx, userID, includeNSFW)
	if err != nil {
		return model.SearchPage{}, err
	}
	posts, err := s.repo.ListFavorites(ctx, userID, includeNSFW, lim, off)
	if err != nil {
		return model.SearchPage{}, err
	}
	if posts == nil {
		posts = []model.Post{}
	}
	return model.SearchPage{Total: total, Posts: posts}, nil
}
func (s *Service) FavoriteIDs(ctx context.Context, userID string) (map[string]bool, error) {
	ids, err := s.repo.FavoriteIDs(ctx, userID)
	out := make(map[string]bool, len(ids))
	for _, id := range ids {
		out[id] = true
	}
	return out, err
}

func validateList(title, desc, vis string) (string, string, string, error) {
	title = strings.TrimSpace(title)
	desc = strings.TrimSpace(desc)
	if title == "" || len([]rune(title)) > 80 {
		return "", "", "", &ValidationError{Message: "Give the list a title (up to 80 characters)."}
	}
	if len([]rune(desc)) > 300 {
		return "", "", "", &ValidationError{Message: "List descriptions are capped at 300 characters."}
	}
	if vis != "public" {
		vis = "private"
	}
	return title, desc, vis, nil
}

func (s *Service) CreateList(ctx context.Context, ownerID, title, desc, vis string) (model.List, error) {
	title, desc, vis, err := validateList(title, desc, vis)
	if err != nil {
		return model.List{}, err
	}
	return s.repo.CreateList(ctx, ownerID, title, desc, vis)
}

func (s *Service) MyLists(ctx context.Context, ownerID string) ([]model.List, error) {
	return s.repo.ListsByOwner(ctx, ownerID, false)
}

func (s *Service) PublicLists(ctx context.Context, ownerID string) ([]model.List, error) {
	return s.repo.ListsByOwner(ctx, ownerID, true)
}

// GetList enforces visibility: private lists are only visible to their owner.
func (s *Service) GetList(ctx context.Context, viewerID *string, id string, includeNSFW bool, limit, offset int) (model.ListDetail, error) {
	list, err := s.repo.GetList(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return model.ListDetail{}, ErrNotFound
	}
	if err != nil {
		return model.ListDetail{}, err
	}
	if list.Visibility != "public" && (viewerID == nil || *viewerID != list.OwnerID) {
		return model.ListDetail{}, ErrNotFound // do not leak that a private list exists
	}
	lim, off := model.PageBounds(limit, offset)
	total, err := s.repo.CountListItems(ctx, id, includeNSFW)
	if err != nil {
		return model.ListDetail{}, err
	}
	items, err := s.repo.ListItems(ctx, id, includeNSFW, lim, off)
	if items == nil {
		items = []model.Post{}
	}
	return model.ListDetail{List: list, Items: items, Total: total}, err
}

func (s *Service) ownedList(ctx context.Context, ownerID, id string) (model.List, error) {
	list, err := s.repo.GetList(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return model.List{}, ErrNotFound
	}
	if err != nil {
		return model.List{}, err
	}
	if list.OwnerID != ownerID {
		return model.List{}, ErrForbidden
	}
	return list, nil
}

func (s *Service) UpdateList(ctx context.Context, ownerID, id, title, desc, vis string) (model.List, error) {
	if _, err := s.ownedList(ctx, ownerID, id); err != nil {
		return model.List{}, err
	}
	title, desc, vis, err := validateList(title, desc, vis)
	if err != nil {
		return model.List{}, err
	}
	updated, err := s.repo.UpdateList(ctx, id, ownerID, title, desc, vis)
	if errors.Is(err, sql.ErrNoRows) {
		return model.List{}, ErrNotFound
	}
	return updated, err
}

func (s *Service) DeleteList(ctx context.Context, ownerID, id string) error {
	n, err := s.repo.DeleteList(ctx, id, ownerID)
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Service) AddItem(ctx context.Context, ownerID, listID, postID string) error {
	if _, err := s.ownedList(ctx, ownerID, listID); err != nil {
		return err
	}
	pos, err := s.repo.NextPosition(ctx, listID)
	if err != nil {
		return err
	}
	if err := s.repo.AddItem(ctx, listID, postID, pos); err != nil {
		return err
	}
	return s.repo.TouchList(ctx, listID)
}

func (s *Service) RemoveItem(ctx context.Context, ownerID, listID, postID string) error {
	if _, err := s.ownedList(ctx, ownerID, listID); err != nil {
		return err
	}
	if err := s.repo.RemoveItem(ctx, listID, postID); err != nil {
		return err
	}
	return s.repo.TouchList(ctx, listID)
}

func (s *Service) Reorder(ctx context.Context, ownerID, listID string, postIDs []string) error {
	if _, err := s.ownedList(ctx, ownerID, listID); err != nil {
		return err
	}
	ids, err := s.repo.ListItemIDs(ctx, listID)
	if err != nil {
		return err
	}
	if len(postIDs) != len(ids) {
		return &ValidationError{Message: "Reorder must include every item in the list."}
	}
	seen := make(map[string]bool, len(postIDs))
	for _, id := range postIDs {
		if seen[id] {
			return &ValidationError{Message: "Reorder has a duplicate item."}
		}
		seen[id] = true
	}
	for _, id := range ids {
		if !seen[id] {
			return &ValidationError{Message: "Reorder must include every item in the list."}
		}
	}
	return s.repo.Reorder(ctx, listID, postIDs)
}

// MoveItem places postID immediately after afterID. An empty afterID
// moves it to the start. Items the caller has not loaded stay put.
func (s *Service) MoveItem(ctx context.Context, ownerID, listID, postID, afterID string) error {
	if _, err := s.ownedList(ctx, ownerID, listID); err != nil {
		return err
	}
	if postID == "" {
		return &ValidationError{Message: "Choose a post to move."}
	}
	if postID == afterID {
		return nil
	}
	ids, err := s.repo.ListItemIDs(ctx, listID)
	if err != nil {
		return err
	}
	from := -1
	for i, id := range ids {
		if id == postID {
			from = i
			break
		}
	}
	if from < 0 {
		return &ValidationError{Message: "That post is not in this list."}
	}
	rest := append(append([]string{}, ids[:from]...), ids[from+1:]...)
	insert := 0
	if afterID != "" {
		insert = -1
		for i, id := range rest {
			if id == afterID {
				insert = i + 1
				break
			}
		}
		if insert < 0 {
			return &ValidationError{Message: "Move relative to a post in this list."}
		}
	}
	next := make([]string, 0, len(rest)+1)
	next = append(next, rest[:insert]...)
	next = append(next, postID)
	next = append(next, rest[insert:]...)
	return s.repo.Reorder(ctx, listID, next)
}
