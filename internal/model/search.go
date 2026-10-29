package model

// PostQuery is a board search. Tags are required together (AND).
// Exclude drops any post carrying one of those tags. Q matches the
// artist credit or a tag, including a close misspelling. Artist is an
// extra artist substring, used by the agent. Sort is newest or oldest.
type PostQuery struct {
	Tags        []string
	Exclude     []string
	Q           string
	QLike       string
	Artist      string
	ArtistLike  string
	Sort        string
	Limit       int32
	Offset      int32
	IncludeNSFW bool
}

// SearchPage is one page of a board search plus the full match count,
// so a caller can keep paging without downloading the board.
type SearchPage struct {
	Total int64
	Posts []Post
}

// TagCount is one tag and how many visible posts carry it.
type TagCount struct {
	Tag   string
	Count int64
}

// Neighbors is where one post sits in a search sequence.
// Index is zero-based, or -1 when the post is not in the sequence.
type Neighbors struct {
	Index  int
	Total  int64
	PrevID string
	NextID string
}
