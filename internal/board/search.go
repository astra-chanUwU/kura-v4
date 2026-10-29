package board

import (
	"strings"
	"unicode/utf8"

	"github.com/astra-chanUwU/kura-v4/internal/model"
)

// ErrBadSort is returned when a search sort is not newest or oldest.
var ErrBadSort = errString("sort must be newest or oldest")

// ErrBadCursor is returned when a search page continues from a post
// that is not in this library.
var ErrBadCursor = errString("after must be a post in this library")

type errString string

func (e errString) Error() string { return string(e) }

// PrepareQuery normalizes a board search the same way the board does:
// tags are booru tokens, a tag in both lists is excluded, free text
// shorter than two characters is ignored, and the page is bounded.
func PrepareQuery(q model.PostQuery) (model.PostQuery, error) {
	q.Tags = NormalizeTags(q.Tags)
	q.Exclude = NormalizeTags(q.Exclude)
	if len(q.Exclude) > 0 {
		skip := map[string]bool{}
		for _, tag := range q.Exclude {
			skip[tag] = true
		}
		kept := q.Tags[:0]
		for _, tag := range q.Tags {
			if !skip[tag] {
				kept = append(kept, tag)
			}
		}
		q.Tags = append([]string{}, kept...)
	}
	q.Q = clipText(strings.ToLower(strings.TrimSpace(q.Q)), 80)
	if utf8.RuneCountInString(q.Q) < 2 {
		q.Q = ""
	}
	q.QLike = likeContains(q.Q)
	q.Artist = clipText(strings.TrimSpace(q.Artist), 80)
	q.ArtistLike = likeContains(strings.ToLower(q.Artist))
	if q.Sort == "" {
		q.Sort = "newest"
	}
	if q.Sort != "newest" && q.Sort != "oldest" {
		return model.PostQuery{}, ErrBadSort
	}
	if q.Limit <= 0 {
		q.Limit = 40
	}
	if q.Limit > 100 {
		q.Limit = 100
	}
	if q.Offset < 0 {
		q.Offset = 0
	}
	if q.Tags == nil {
		q.Tags = []string{}
	}
	if q.Exclude == nil {
		q.Exclude = []string{}
	}
	return q, nil
}

func clipText(s string, maxRunes int) string {
	if utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	return string([]rune(s)[:maxRunes])
}

// likeContains builds a case-insensitive substring pattern. % and _
// in the text stay literal.
func likeContains(s string) string {
	if s == "" {
		return ""
	}
	return "%" + escapeLike(s) + "%"
}

func escapeLike(s string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return replacer.Replace(s)
}
