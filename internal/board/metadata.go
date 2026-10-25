package board

import (
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// SeedMeta is the metadata parsed out of a seed/upload filename.
type SeedMeta struct {
	Artist   string
	Source   string
	SourceID string
	Tags     []string
}

var mediumWords = map[string]string{
	"gif": "gif", "img": "image", "image": "image", "pic": "image",
	"photo": "image", "video": "video", "mp4": "video", "webm": "video",
}

func isNumericID(s string) bool {
	if len(s) < 6 {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// ParseSeedName extracts artist/source/tags from downloader filenames,
// e.g. "twitter gif Serraphyll 2108932354891030955.mp4" ->
// artist "Serraphyll", source "twitter", id 2108932354891030955, tags [gif twitter].
// Anything unrecognized just yields empty metadata.
func ParseSeedName(filename string) SeedMeta {
	name := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
	tokens := strings.FieldsFunc(strings.ToLower(name), func(r rune) bool {
		return r == ' ' || r == '-' || r == '_' || r == '.'
	})
	// Keep original-cased tokens for the artist name.
	orig := strings.FieldsFunc(name, func(r rune) bool {
		return r == ' ' || r == '-' || r == '_' || r == '.'
	})
	var out SeedMeta
	tagSet := map[string]bool{}
	addTag := func(t string) {
		if t != "" {
			tagSet[t] = true
		}
	}
	i := 0
	if len(tokens) > 0 && tokens[0] == "twitter" {
		out.Source = "twitter"
		addTag("twitter")
		i++
		if i < len(tokens) {
			if medium, ok := mediumWords[tokens[i]]; ok {
				addTag(medium)
				i++
			}
		}
		// First remaining token is the artist, a long numeric token the id.
		for ; i < len(tokens); i++ {
			if isNumericID(tokens[i]) {
				if out.SourceID == "" {
					out.SourceID = tokens[i]
				}
				continue
			}
			if out.Artist == "" && i < len(orig) {
				out.Artist = orig[i]
			}
		}
	} else {
		for _, tok := range tokens {
			if isNumericID(tok) && out.SourceID == "" {
				out.SourceID = tok
			}
		}
	}
	for tag := range tagSet {
		out.Tags = append(out.Tags, tag)
	}
	sort.Strings(out.Tags)
	return out
}

var tagCleanRE = regexp.MustCompile(`[^a-z0-9_-]+`)

// NormalizeTags flattens tag input the booru way: spaces and commas
// separate tags ("anime girl white_hair" is three tags), underscores
// join words inside a tag, everything is lowercased and deduped.
func NormalizeTags(in []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, raw := range in {
		for _, token := range strings.FieldsFunc(raw, func(r rune) bool {
			return r == ' ' || r == '\t' || r == ',' || r == '\n'
		}) {
			tag := tagCleanRE.ReplaceAllString(strings.ToLower(strings.TrimSpace(token)), "_")
			tag = strings.ReplaceAll(tag, "-", "_")
			tag = strings.Trim(tag, "_")
			if tag == "" || len(tag) > 40 || seen[tag] {
				continue
			}
			seen[tag] = true
			out = append(out, tag)
			if len(out) >= 30 {
				break
			}
		}
		if len(out) >= 30 {
			break
		}
	}
	sort.Strings(out)
	return out
}
