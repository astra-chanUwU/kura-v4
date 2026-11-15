package repository

import (
	"context"
	"testing"

	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/stretchr/testify/require"
)

func TestGetPostRejectsBadID(t *testing.T) {
	_, err := (&Posts{}).GetPost(context.Background(), "nope")
	require.ErrorIs(t, err, model.ErrNotFound)
}
