package controllers

import (
	"strconv"
	"strings"

	"github.com/astra-chanUwU/kura-v4/internal/board"
	"github.com/astra-chanUwU/kura-v4/internal/collections"
	"github.com/astra-chanUwU/kura-v4/internal/dto"
	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/gofiber/fiber/v3"
)

type Posts struct {
	board       *board.Service
	collections *collections.Service
}

func NewPosts(b *board.Service, coll *collections.Service) *Posts {
	return &Posts{board: b, collections: coll}
}

func (h *Posts) favs(c fiber.Ctx) map[string]bool {
	if u := currentUser(c); u != nil {
		if ids, err := h.collections.FavoriteIDs(c.Context(), u.ID); err == nil {
			return ids
		}
	}
	return map[string]bool{}
}

func (h *Posts) RegisterRoutes(api fiber.Router) {
	api.Get("/search", h.search)
	api.Get("/tags", h.tags)
	api.Get("/posts", h.list)
	api.Get("/posts/:id/neighbors", h.neighbors)
	api.Get("/posts/:id", h.get)
	api.Post("/posts", h.create)
	api.Patch("/posts/:id/meta", h.updateMeta)
	api.Get("/me/posts", h.myPosts)
}

// myPosts lists the signed-in user's own uploads (their contributions),
// newest first, with the viewer's NSFW visibility applied.
func (h *Posts) myPosts(c fiber.Ctx) error {
	user, err := requireUser(c)
	if err != nil {
		return err
	}
	limit, offset := queryPage(c)
	page, err := h.board.ListByOwner(c.Context(), user.ID, viewerSeesNSFW(c), limit, offset)
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(dto.SearchPage{Total: page.Total, Posts: postsDTO(page.Posts, h.favs(c))})
}

// RegisterMedia serves original bytes for public posts. Favorites and lists
// are private collections, but the posts on this board are public.
// RegisterMedia registers post media on the given router (mounted
// behind optional-auth so NSFW gating can see the viewer).
func (h *Posts) RegisterMedia(app fiber.Router) {
	app.Get("/posts/:id", h.media)
	app.Get("/posts/:id/thumb", h.thumbnail)
}

func (h *Posts) searchQuery(c fiber.Ctx) model.PostQuery {
	limit := int32(40)
	if v, err := strconv.Atoi(c.Query("limit")); err == nil && v > 0 {
		limit = min(int32(v), 100)
	}
	offset := int32(0)
	if v, err := strconv.Atoi(c.Query("offset")); err == nil && v > 0 {
		offset = int32(v)
	}
	return model.PostQuery{
		Tags:        splitCSV(c.Query("tags")),
		Exclude:     splitCSV(c.Query("exclude")),
		Q:           c.Query("q"),
		Artist:      c.Query("artist"),
		Sort:        c.Query("sort"),
		Limit:       limit,
		Offset:      offset,
		AfterID:     c.Query("after"),
		IncludeNSFW: viewerSeesNSFW(c),
	}
}

func splitCSV(value string) []string {
	if strings.TrimSpace(value) == "" {
		return []string{}
	}
	return strings.Split(value, ",")
}

// search is the board: required tags, excluded tags, free text, and sort,
// one page at a time.
func (h *Posts) search(c fiber.Ctx) error {
	page, err := h.board.Search(c.Context(), h.searchQuery(c))
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(dto.SearchPage{Total: page.Total, Posts: postsDTO(page.Posts, h.favs(c))})
}

func (h *Posts) tags(c fiber.Ctx) error {
	limit := int32(8)
	if v, err := strconv.Atoi(c.Query("limit")); err == nil && v > 0 {
		limit = min(int32(v), 24)
	}
	filter := h.searchQuery(c)
	filter.Q = c.Query("text")
	filter.AfterID = ""
	tags, err := h.board.SuggestTags(c.Context(), c.Query("q"), filter, limit)
	if err != nil {
		return writeError(c, err)
	}
	out := make([]dto.TagCount, 0, len(tags))
	for _, tag := range tags {
		out = append(out, dto.TagCount{Tag: tag.Tag, Count: tag.Count})
	}
	return c.JSON(out)
}

func (h *Posts) neighbors(c fiber.Ctx) error {
	n, err := h.board.Neighbors(c.Context(), c.Params("id"), h.searchQuery(c))
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(dto.PostNeighbors{Index: n.Index, Total: n.Total, PrevID: n.PrevID, NextID: n.NextID})
}

func (h *Posts) list(c fiber.Ctx) error {
	limit := int32(40)
	if v, err := strconv.Atoi(c.Query("limit")); err == nil && v > 0 {
		limit = min(int32(v), 100)
	}
	offset := int32(0)
	if v, err := strconv.Atoi(c.Query("offset")); err == nil && v > 0 {
		offset = int32(v)
	}
	posts, err := h.board.List(c.Context(), limit, offset, viewerSeesNSFW(c))
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(postsDTO(posts, h.favs(c)))
}

func (h *Posts) get(c fiber.Ctx) error {
	post, err := h.board.Get(c.Context(), c.Params("id"))
	if err != nil {
		return writeError(c, err)
	}
	if !canSeePost(c, post) || (post.Rating == "nsfw" && !viewerSeesNSFW(c)) {
		return writeError(c, board.ErrNotFound)
	}
	return c.JSON(postDTO(post, h.favs(c)))
}

func (h *Posts) create(c fiber.Ctx) error {
	user, err := requireUser(c)
	if err != nil {
		return err
	}
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return fiber.NewError(400, "Choose a file to upload.")
	}
	src, err := fileHeader.Open()
	if err != nil {
		return err
	}
	defer src.Close() //nolint:errcheck
	head := make([]byte, 512)
	n, _ := src.Read(head)
	contentType := board.SniffContentType(head[:n], fileHeader.Filename)
	if !board.IsAllowedMedia(contentType) {
		return fiber.NewError(400, "That file type isn't supported yet. Try an image, GIF or MP4/WebM video.")
	}
	if _, err := src.Seek(0, 0); err != nil {
		return err
	}
	key, sha, err := h.board.Upload(c.Context(), src)
	if err != nil {
		return err
	}
	// Identical bytes are already a post. Say so plainly (409) instead
	// of returning the old post as if the upload had added it, and
	// include that post so the client can show the existing entry.
	if existing, err := h.board.GetBySHA(c.Context(), sha); err == nil {
		return c.Status(409).JSON(fiber.Map{
			"code":    "duplicate_post",
			"message": "Already on the board.",
			"post":    postDTO(existing, h.favs(c)),
		})
	}
	rating := c.FormValue("rating")
	if rating != "" && rating != "sfw" && rating != "nsfw" {
		return fiber.NewError(400, "Rating must be sfw or nsfw.")
	}
	meta := board.ParseSeedName(fileHeader.Filename)
	tags := meta.Tags
	if extra := strings.TrimSpace(c.FormValue("tags")); extra != "" {
		tags = append(tags, extra) // board.NormalizeTags splits/dedupes
	}
	post, err := h.board.Create(c.Context(), model.Post{OwnerID: &user.ID, OriginalName: fileHeader.Filename, StorageKey: key, ContentType: contentType, SizeBytes: fileHeader.Size,
		Artist: meta.Artist, Source: meta.Source, SourceID: meta.SourceID, Tags: tags, Rating: rating}, sha)
	if err != nil {
		// Content-addressed blobs may be shared with an existing post;
		// never delete them here.
		return writeError(c, err)
	}
	post.OwnerUsername = user.Username
	h.board.RecordCreation(c.Context(), *user, post)
	return c.Status(201).JSON(postDTO(post, h.favs(c)))
}

func (h *Posts) updateMeta(c fiber.Ctx) error {
	user, err := requireUser(c)
	if err != nil {
		return err
	}
	var in dto.UpdatePostMeta
	if err := c.Bind().Body(&in); err != nil {
		return fiber.NewError(400, "Invalid request.")
	}
	post, err := h.board.SetMetaAs(c.Context(), user, c.Params("id"), in.Artist, in.Tags, in.Rating)
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(postDTO(post, h.favs(c)))
}

func (h *Posts) thumbnail(c fiber.Ctx) error {
	post, err := h.board.Get(c.Context(), c.Params("id"))
	if err != nil || !canSeePost(c, post) || (post.Rating == "nsfw" && !viewerSeesNSFW(c)) {
		return fiber.ErrNotFound
	}
	file, contentType, err := h.board.Thumbnail(c.Context(), post)
	if err != nil {
		return fiber.ErrNotFound
	}
	c.Set("Content-Type", contentType)
	c.Set("Cache-Control", "public, max-age=3600")
	return c.SendStream(file)
}

func (h *Posts) media(c fiber.Ctx) error {
	post, err := h.board.Get(c.Context(), c.Params("id"))
	if err != nil || !canSeePost(c, post) || (post.Rating == "nsfw" && !viewerSeesNSFW(c)) {
		return fiber.ErrNotFound
	}
	file, err := h.board.OpenMedia(c.Context(), post)
	if err != nil {
		return fiber.ErrNotFound
	}
	// No deferred Close: fasthttp owns and closes the body stream after sending.
	c.Set("Content-Type", post.ContentType)
	c.Set("Cache-Control", "public, max-age=3600")
	return c.SendStream(file)
}
