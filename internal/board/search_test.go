package board

import (
	"testing"

	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/stretchr/testify/require"
)

func TestPrepareQuery(t *testing.T) {
	q, err := PrepareQuery(model.PostQuery{
		Tags:    []string{"Yuri", "gif"},
		Exclude: []string{"GIF"},
		Q:       "  S ",
		Artist:  "Serraphyl",
		Sort:    "",
		Limit:   0,
		Offset:  -3,
	})
	require.NoError(t, err)
	require.Equal(t, []string{"yuri"}, q.Tags)
	require.Equal(t, []string{"gif"}, q.Exclude)
	require.Equal(t, "", q.Q, "a one-character query is not a search")
	require.Equal(t, "", q.QLike)
	require.Equal(t, "%serraphyl%", q.ArtistLike)
	require.Equal(t, "newest", q.Sort)
	require.Equal(t, int32(40), q.Limit)
	require.Equal(t, int32(0), q.Offset)

	q, err = PrepareQuery(model.PostQuery{Q: "100%_sure", Sort: "oldest", Limit: 500})
	require.NoError(t, err)
	require.Equal(t, "oldest", q.Sort)
	require.Equal(t, int32(100), q.Limit)
	require.Equal(t, `%100\%\_sure%`, q.QLike)

	_, err = PrepareQuery(model.PostQuery{Sort: "score"})
	require.ErrorIs(t, err, ErrBadSort)
}
