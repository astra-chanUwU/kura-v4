package site

import (
	"context"
	"fmt"
	"strings"

	"github.com/astra-chanUwU/kura-v4/internal/model"
)

type Repository interface {
	Get(context.Context) (model.Site, error)
}
type Service struct{ repo Repository }

func NewService(repo Repository) *Service { return &Service{repo: repo} }
func (s *Service) Get(ctx context.Context) (model.Site, error) {
	info, err := s.repo.Get(ctx)
	if err != nil {
		return model.Site{}, fmt.Errorf("load site settings: %w", err)
	}
	if strings.TrimSpace(info.Name) == "" {
		info.Name = "Kura"
	}
	return info, nil
}
