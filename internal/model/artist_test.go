package model

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestArtistToken(t *testing.T) {
	require.Equal(t, "john_smith", ArtistToken("John Smith"))
	require.Equal(t, "ada_lovelace", ArtistToken("  Ada Lovelace  "))
	require.Equal(t, "mary_jane", ArtistToken("Mary-Jane"))
	require.Equal(t, "john_smith", ArtistToken("John--Smith"))
	require.Equal(t, "hello_world", ArtistToken("hello.world"))
	require.Equal(t, "a_b", ArtistToken("a  b"))
	require.Equal(t, "a_b", ArtistToken("a__b"))
	require.Equal(t, "john", ArtistToken("___john___"))
	require.Equal(t, "bea", ArtistToken("Bea"))
	require.Equal(t, "9lives", ArtistToken("9lives"))
	require.Equal(t, "jos", ArtistToken("José"))
	require.Equal(t, strings.Repeat("a", 40), ArtistToken(strings.Repeat("a", 50)))
	require.Equal(t, "", ArtistToken(""))
	require.Equal(t, "", ArtistToken("   "))
	require.Equal(t, "", ArtistToken("---"))
	require.Equal(t, "", ArtistToken("___"))
}

func TestApplyArtistTag(t *testing.T) {
	require.Equal(t, []string{"cat", "john_smith"}, ApplyArtistTag("", []string{"cat"}, "John Smith"))
	require.Equal(t, []string{"bea", "cat"}, ApplyArtistTag("Ada Lovelace", []string{"ada_lovelace", "cat"}, "Bea"))
	require.Equal(t, []string{"cat"}, ApplyArtistTag("Ada Lovelace", []string{"ada_lovelace", "cat"}, "  "))
	require.Equal(t, []string{"ada_lovelace", "cat"}, ApplyArtistTag("Ada Lovelace", []string{"cat"}, "Ada Lovelace"))
	require.Equal(t, []string{"ada_lovelace", "cat"}, ApplyArtistTag("", []string{"cat", "cat", "ada_lovelace"}, "Ada Lovelace"))
	require.Equal(t, []string{"aaa", "bea", "cat"}, ApplyArtistTag("", []string{"cat", "aaa"}, "Bea"))
	require.Equal(t, []string{}, ApplyArtistTag("", nil, ""))
}

func TestNormalizeRating(t *testing.T) {
	require.Equal(t, "nsfw", NormalizeRating("nsfw"))
	require.Equal(t, "sfw", NormalizeRating("sfw"))
	require.Equal(t, "sfw", NormalizeRating(""))
	require.Equal(t, "sfw", NormalizeRating("NSFW"))
	require.Equal(t, "sfw", NormalizeRating("questionable"))
}
