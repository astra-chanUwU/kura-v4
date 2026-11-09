package model

import (
	"regexp"
	"sort"
	"strings"
)

// artistTokenPattern is the tag shape stored for an artist credit.
var artistTokenPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_]{0,39}$`)

// ArtistToken turns a display credit into the tag stored beside the post.
// "John Smith" becomes john_smith. Empty or invalid input returns "".
func ArtistToken(artist string) string {
	artist = strings.ToLower(strings.TrimSpace(artist))
	if artist == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(artist))
	underscore := false
	for _, r := range artist {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			underscore = false
		default:
			// Spaces, hyphens, and any other separator collapse to one underscore.
			if !underscore {
				b.WriteByte('_')
				underscore = true
			}
		}
	}
	token := strings.Trim(b.String(), "_")
	if len(token) > 40 {
		token = token[:40]
		token = strings.TrimRight(token, "_")
	}
	if !artistTokenPattern.MatchString(token) {
		return ""
	}
	return token
}

// ApplyArtistTag drops previousArtist's token when the new credit token
// differs, then adds the new token when it is missing. A blank artist only
// removes the old token. The result is sorted and deduped.
func ApplyArtistTag(previousArtist string, tags []string, artist string) []string {
	prev := ArtistToken(previousArtist)
	next := ArtistToken(artist)
	seen := map[string]bool{}
	out := make([]string, 0, len(tags)+1)
	for _, tag := range tags {
		if tag == "" || seen[tag] {
			continue
		}
		if prev != "" && tag == prev && prev != next {
			continue
		}
		seen[tag] = true
		out = append(out, tag)
	}
	if next != "" && !seen[next] {
		out = append(out, next)
	}
	sort.Strings(out)
	return out
}

// NormalizeRating keeps the two board ratings. Only an exact "nsfw" stays
// nsfw; every other value, including a blank, is sfw.
func NormalizeRating(rating string) string {
	if rating == "nsfw" {
		return "nsfw"
	}
	return "sfw"
}
