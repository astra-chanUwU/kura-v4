package site

import (
	"context"
	"errors"
	"testing"

	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/stretchr/testify/require"
)

func TestGetPreservesConfiguredName(t *testing.T) {
	repo := NewMockRepository(t)
	repo.EXPECT().Get(context.Background()).Return(model.Site{Name: "My library"}, nil)
	got, err := NewService(repo).Get(context.Background())
	require.NoError(t, err)
	require.Equal(t, "My library", got.Name)
}
func TestGetFallsBackForBlankName(t *testing.T) {
	repo := NewMockRepository(t)
	repo.EXPECT().Get(context.Background()).Return(model.Site{Name: "  "}, nil)
	got, err := NewService(repo).Get(context.Background())
	require.NoError(t, err)
	require.Equal(t, "Kura", got.Name)
}
func TestGetPreservesDatabaseFailure(t *testing.T) {
	repo := NewMockRepository(t)
	failure := errors.New("database offline")
	repo.EXPECT().Get(context.Background()).Return(model.Site{}, failure)
	_, err := NewService(repo).Get(context.Background())
	require.ErrorIs(t, err, failure)
}
