package app

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSeedTagsAreDeterministicAndFolderScoped(t *testing.T) {
	sha := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	a := seedTags("/pics/Anime", "image/png", sha, []string{"twitter"})
	b := seedTags("/pics/Anime", "image/png", sha, []string{"twitter"})
	require.Equal(t, a, b)
	require.Contains(t, a, "anime")
	require.Contains(t, a, "image")
	require.Contains(t, a, "seed")
	require.Contains(t, a, "twitter")
	// folder + medium + seed + twitter + three pool picks, deduped
	require.GreaterOrEqual(t, len(a), 5)
	require.LessOrEqual(t, len(a), 7)
}

func TestSeedTagsMediumAndLegacyPool(t *testing.T) {
	sha := "ff00ff00ff00ff00ff00ff00ff00ff00ff00ff00ff00ff00ff00ff00ff00ff00"
	tags := seedTags("/pics/Legacy Mac", "video/mp4", sha, nil)
	require.Contains(t, tags, "legacy")
	require.Contains(t, tags, "mac")
	require.Contains(t, tags, "video")
	require.NotContains(t, tags, "image")
}
