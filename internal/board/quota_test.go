package board

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func repeatOwned(owner string, at time.Time, n int) []ownedPost {
	posts := make([]ownedPost, n)
	for i := range posts {
		posts[i] = ownedPost{ownerID: owner, createdAt: at}
	}
	return posts
}

func TestCheckUploadQuota(t *testing.T) {
	now := time.Now()
	recent := now.Add(-time.Minute)
	older := now.Add(-2 * time.Hour)
	stale := now.Add(-48 * time.Hour)
	check := func(posts []ownedPost) error {
		return NewService(&fakeRepo{owned: posts}, nil).CheckUploadQuota(context.Background(), "ada")
	}

	t.Run("under the hour cap", func(t *testing.T) {
		posts := append(repeatOwned("ada", recent, 19), repeatOwned("bea", recent, 30)...)
		require.NoError(t, check(posts))
	})

	t.Run("twenty-first in an hour", func(t *testing.T) {
		// Hidden and deleted rows count, so 18 plus one of each is already 20.
		posts := append(repeatOwned("ada", recent, 18),
			ownedPost{ownerID: "ada", createdAt: recent, hidden: true},
			ownedPost{ownerID: "ada", createdAt: recent, deleted: true},
		)
		err := check(posts)
		require.ErrorIs(t, err, ErrUploadLimit)
		require.EqualError(t, err, "Upload limit reached. Try again later.")
	})

	t.Run("day cap", func(t *testing.T) {
		under := append(repeatOwned("ada", older, 99), repeatOwned("ada", stale, 50)...)
		require.NoError(t, check(under))
		full := append(repeatOwned("ada", older, 100), repeatOwned("ada", stale, 50)...)
		require.ErrorIs(t, check(full), ErrUploadLimit)
	})

	t.Run("older than an hour", func(t *testing.T) {
		// Nineteen in the last hour plus one from two hours ago is still under the hour cap.
		posts := append(repeatOwned("ada", recent, 19), repeatOwned("ada", older, 1)...)
		require.NoError(t, check(posts))
	})
}
