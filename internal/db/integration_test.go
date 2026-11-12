//go:build integration

package db_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/astra-chanUwU/kura-v4/internal/board"
	"github.com/astra-chanUwU/kura-v4/internal/dao"
	"github.com/astra-chanUwU/kura-v4/internal/db"
	"github.com/astra-chanUwU/kura-v4/internal/jobs"
	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/astra-chanUwU/kura-v4/internal/reports"
	"github.com/astra-chanUwU/kura-v4/internal/repository"
	"github.com/astra-chanUwU/kura-v4/internal/tags"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestPostgresContracts(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	container, err := postgres.Run(ctx, "postgres:18-alpine", postgres.WithDatabase("kura_test"), postgres.WithUsername("test"), postgres.WithPassword("test"), postgres.BasicWaitStrategies())
	require.NoError(t, err, "Docker must be running for disposable database tests")
	t.Cleanup(func() { _ = container.Terminate(context.Background()) })
	url, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	conn, err := db.Open(ctx, url)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	require.NoError(t, db.Migrate(ctx, conn))
	require.NoError(t, db.Migrate(ctx, conn), "migrations are repeatable")
	info, err := dao.NewSite(conn).Get(ctx)
	require.NoError(t, err)
	require.Equal(t, "Kura", info.Name)

	queue := dao.NewJobs(conn)
	id, err := queue.Enqueue(ctx, "test", json.RawMessage(`{"key":"original"}`))
	require.NoError(t, err)
	var wg sync.WaitGroup
	claimed := make(chan model.Job, 2)
	failures := make(chan error, 2)
	for range 2 {
		wg.Go(func() {
			job, err := queue.Claim(ctx, []string{"test"})
			if err == nil {
				claimed <- job
			} else {
				failures <- err
			}
		})
	}
	wg.Wait()
	require.Len(t, claimed, 1, "two workers must not lease the same job")
	require.Len(t, failures, 1)
	require.ErrorIs(t, <-failures, sql.ErrNoRows)
	first := <-claimed
	require.Equal(t, id, first.ID)
	_, err = conn.ExecContext(ctx, "UPDATE jobs SET lease_until = now() - interval '1 second' WHERE id = $1", id)
	require.NoError(t, err)
	restartedWorker := dao.NewJobs(conn)
	second, err := restartedWorker.Claim(ctx, []string{"test"})
	require.NoError(t, err)
	require.Equal(t, first.ID, second.ID)
	require.NotEqual(t, first.Token, second.Token)
	require.ErrorIs(t, queue.Complete(ctx, first), sql.ErrNoRows, "stale worker must not acknowledge a new lease")
	require.NoError(t, restartedWorker.Fail(ctx, second, "retry"))
	_, err = conn.ExecContext(ctx, "UPDATE jobs SET available_at = now() WHERE id = $1", id)
	require.NoError(t, err)
	third, err := queue.Claim(ctx, []string{"test"})
	require.NoError(t, err)
	require.NoError(t, queue.Fail(ctx, third, "final failure"))
	_, err = queue.Claim(ctx, []string{"test"})
	require.ErrorIs(t, err, sql.ErrNoRows)
	var state string
	require.NoError(t, conn.QueryRowContext(ctx, "SELECT state FROM jobs WHERE id = $1", id).Scan(&state))
	require.Equal(t, "failed", state)

	postsDAO := dao.NewPosts(conn)
	seed := func(name, artist, rating string, tags []string) model.Post {
		t.Helper()
		created, err := postsDAO.CreatePost(ctx, model.Post{
			OriginalName: name, StorageKey: name, ContentType: "image/jpeg", SizeBytes: 1,
			Artist: artist, Rating: rating,
		}, name+"-sha")
		require.NoError(t, err)
		for _, tag := range tags {
			require.NoError(t, postsDAO.InsertPostTag(ctx, created.ID, tag))
		}
		return created
	}
	cute := seed("cute.jpg", "Serraphyll", "sfw", []string{"yuri", "cute"})
	gif := seed("gif.jpg", "", "sfw", []string{"yuri", "gif"})
	nsfw := seed("nsfw.jpg", "", "nsfw", []string{"landscape"})
	hidden := seed("hidden.jpg", "", "sfw", []string{"yuri"})
	_, err = conn.ExecContext(ctx, `UPDATE posts SET created_at = now() - interval '3 days' WHERE id = $1`, cute.ID)
	require.NoError(t, err)
	_, err = conn.ExecContext(ctx, `UPDATE posts SET created_at = now() - interval '2 days' WHERE id = $1`, nsfw.ID)
	require.NoError(t, err)
	_, err = conn.ExecContext(ctx, `UPDATE posts SET created_at = now() - interval '1 day' WHERE id = $1`, gif.ID)
	require.NoError(t, err)
	_, err = conn.ExecContext(ctx, `UPDATE posts SET hidden_at = now() WHERE id = $1`, hidden.ID)
	require.NoError(t, err)

	prepared, err := board.PrepareQuery(model.PostQuery{Tags: []string{"yuri"}, Exclude: []string{"gif"}})
	require.NoError(t, err)
	total, err := postsDAO.CountBoardMatches(ctx, prepared)
	require.NoError(t, err)
	require.Equal(t, int64(1), total, "tag AND plus exclude, and hidden and nsfw stay out")
	page, err := postsDAO.SearchPosts(ctx, prepared)
	require.NoError(t, err)
	require.Equal(t, []string{cute.ID}, idsOf(page))

	typo, err := board.PrepareQuery(model.PostQuery{Q: "serraphyl"})
	require.NoError(t, err)
	found, err := postsDAO.SearchPosts(ctx, typo)
	require.NoError(t, err)
	require.Equal(t, []string{cute.ID}, idsOf(found))

	oldest, err := board.PrepareQuery(model.PostQuery{Tags: []string{"yuri"}, Sort: "oldest", IncludeNSFW: true})
	require.NoError(t, err)
	ordered, err := postsDAO.SearchPosts(ctx, oldest)
	require.NoError(t, err)
	require.Equal(t, []string{cute.ID, gif.ID}, idsOf(ordered))
	counted, err := postsDAO.CountBoardMatches(ctx, oldest)
	require.NoError(t, err)
	require.Equal(t, int64(2), counted)

	neighbors, err := postsDAO.PostNeighbors(ctx, gif.ID, oldest)
	require.NoError(t, err)
	require.Equal(t, 1, neighbors.Index)
	require.Equal(t, int64(2), neighbors.Total)
	require.Equal(t, cute.ID, neighbors.PrevID)
	require.Empty(t, neighbors.NextID)

	suggestions, err := postsDAO.SuggestTags(ctx, "yur", model.PostQuery{Tags: []string{}, Exclude: []string{}}, 8)
	require.NoError(t, err)
	require.NotEmpty(t, suggestions)
	require.Equal(t, "yuri", suggestions[0].Tag)

	scoped, err := postsDAO.SuggestTags(ctx, "", prepared, 8)
	require.NoError(t, err)
	require.Equal(t, []string{"cute", "yuri"}, tagNames(scoped))

	oldest.Limit = 1
	firstPage, err := postsDAO.SearchPosts(ctx, oldest)
	require.NoError(t, err)
	require.Equal(t, []string{cute.ID}, idsOf(firstPage))
	oldest.AfterID = firstPage[0].ID
	oldest.AfterAt = firstPage[0].CreatedAt
	oldest.Offset = 0
	secondPage, err := postsDAO.SearchPosts(ctx, oldest)
	require.NoError(t, err)
	require.Equal(t, []string{gif.ID}, idsOf(secondPage), "after continues the page and ignores offset")

	outside, err := postsDAO.PostNeighbors(ctx, nsfw.ID, prepared)
	require.NoError(t, err)
	require.Equal(t, -1, outside.Index)
	require.Equal(t, int64(1), outside.Total)

	favs := dao.NewCollections(conn)
	var fanA, fanB string
	require.NoError(t, conn.QueryRowContext(ctx, `INSERT INTO users (username, display_name) VALUES ('fana', 'Fan A') RETURNING id`).Scan(&fanA))
	require.NoError(t, conn.QueryRowContext(ctx, `INSERT INTO users (username, display_name) VALUES ('fanb', 'Fan B') RETURNING id`).Scan(&fanB))
	require.NoError(t, favs.AddFavorite(ctx, fanA, cute.ID))
	require.NoError(t, favs.AddFavorite(ctx, fanB, cute.ID))
	require.NoError(t, favs.AddFavorite(ctx, fanA, gif.ID))
	byFav, err := postsDAO.SearchPosts(ctx, model.PostQuery{
		Tags: []string{"yuri"}, Exclude: []string{}, Sort: "favorites", IncludeNSFW: true, Limit: 40,
	})
	require.NoError(t, err)
	require.Equal(t, []string{cute.ID, gif.ID}, idsOf(byFav), "higher favorite_count sorts first")
	kept, err := postsDAO.RelatedPosts(ctx, cute.ID, true, nil, 8)
	require.NoError(t, err)
	require.Equal(t, []string{gif.ID}, idsOf(kept))
	blocked, err := postsDAO.RelatedPosts(ctx, cute.ID, true, []string{"gif"}, 8)
	require.NoError(t, err)
	require.Empty(t, blocked, "exclude drops a post carrying that tag")

	var ownerID string
	require.NoError(t, conn.QueryRowContext(ctx, `INSERT INTO users (username, display_name) VALUES ('pager', 'Pager') RETURNING id`).Scan(&ownerID))
	_, err = conn.ExecContext(ctx, `UPDATE posts SET owner_id = $1 WHERE id = $2 OR id = $3`, ownerID, cute.ID, nsfw.ID)
	require.NoError(t, err)
	visibleTotal, err := postsDAO.CountVisiblePostsByOwner(ctx, ownerID, false)
	require.NoError(t, err)
	require.Equal(t, int64(1), visibleTotal, "owner count hides nsfw until opted in")
	ownerPage, err := postsDAO.ListPostsByOwner(ctx, ownerID, false, 40, 0)
	require.NoError(t, err)
	require.Equal(t, []string{cute.ID}, idsOf(ownerPage))

	collectionsDAO := dao.NewCollections(conn)
	require.NoError(t, collectionsDAO.AddFavorite(ctx, ownerID, cute.ID))
	require.NoError(t, collectionsDAO.AddFavorite(ctx, ownerID, nsfw.ID))
	favTotal, err := collectionsDAO.CountFavorites(ctx, ownerID, false)
	require.NoError(t, err)
	require.Equal(t, int64(1), favTotal)
	favPage, err := collectionsDAO.ListFavorites(ctx, ownerID, false, 40, 0)
	require.NoError(t, err)
	require.Equal(t, []string{cute.ID}, idsOf(favPage))
	require.Equal(t, "Serraphyll", favPage[0].Artist)
	require.Equal(t, []string{"cute", "yuri"}, favPage[0].Tags)

	list, err := collectionsDAO.CreateList(ctx, ownerID, "Shelf", "", "public")
	require.NoError(t, err)
	require.NoError(t, collectionsDAO.AddItem(ctx, list.ID, cute.ID, 0))
	require.NoError(t, collectionsDAO.AddItem(ctx, list.ID, gif.ID, 1))
	require.NoError(t, collectionsDAO.AddItem(ctx, list.ID, nsfw.ID, 2))
	itemTotal, err := collectionsDAO.CountListItems(ctx, list.ID, false, true, nil)
	require.NoError(t, err)
	require.Equal(t, int64(2), itemTotal)
	itemPage, err := collectionsDAO.ListItems(ctx, list.ID, false, true, 1, 0, nil)
	require.NoError(t, err)
	require.Equal(t, []string{cute.ID}, idsOf(itemPage))
	require.Equal(t, "Serraphyll", itemPage[0].Artist)
	require.Equal(t, []string{"cute", "yuri"}, itemPage[0].Tags)
	itemIDs, err := collectionsDAO.ListItemIDs(ctx, list.ID)
	require.NoError(t, err)
	require.Equal(t, []string{cute.ID, gif.ID, nsfw.ID}, itemIDs)
	require.NoError(t, collectionsDAO.AddItem(ctx, list.ID, hidden.ID, 3))
	publicTotal, err := collectionsDAO.CountListItems(ctx, list.ID, true, false, nil)
	require.NoError(t, err)
	require.Equal(t, int64(3), publicTotal, "a public list omits hidden posts")
	ownerTotal, err := collectionsDAO.CountListItems(ctx, list.ID, true, true, nil)
	require.NoError(t, err)
	require.Equal(t, int64(4), ownerTotal)
	publicPage, err := collectionsDAO.ListItems(ctx, list.ID, true, false, 10, 0, nil)
	require.NoError(t, err)
	require.Equal(t, []string{cute.ID, gif.ID, nsfw.ID}, idsOf(publicPage))

	hiddenCount, deletedCount, err := postsDAO.CountRemovedPosts(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(1), hiddenCount)
	require.Equal(t, int64(0), deletedCount)
	hiddenPage, err := postsDAO.ListHiddenPosts(ctx, "hidden", "", "", "", 40, 0)
	require.NoError(t, err)
	require.Equal(t, []string{hidden.ID}, idsOf(hiddenPage))

	tagged, err := repository.NewPosts(postsDAO).CreatePost(ctx, model.Post{
		OriginalName: "tagged.jpg", StorageKey: "tagged.jpg", ContentType: "image/jpeg", SizeBytes: 1,
		Artist: "Ada", Rating: "sfw", Tags: []string{"yuri", "cute"},
	}, "tagged-sha")
	require.NoError(t, err)
	stored, err := postsDAO.GetPost(ctx, tagged.ID)
	require.NoError(t, err)
	require.Equal(t, "Ada", stored.Artist)
	require.Equal(t, []string{"cute", "yuri"}, stored.Tags)

	freshOwner := ownerID
	fresh, err := postsDAO.CreatePost(ctx, model.Post{
		OwnerID: &freshOwner, OriginalName: "quota.jpg", StorageKey: "quota.jpg", ContentType: "image/jpeg", SizeBytes: 1, Rating: "sfw",
	}, "quota-sha")
	require.NoError(t, err)
	hourCount, err := postsDAO.CountPostsByOwnerSince(ctx, ownerID, time.Hour)
	require.NoError(t, err)
	require.Equal(t, int64(1), hourCount, "only the new post is inside the rolling hour")
	require.NoError(t, board.NewService(repository.NewPosts(postsDAO), nil).CheckUploadQuota(ctx, ownerID))
	_, err = conn.ExecContext(ctx, `UPDATE posts SET deleted_at = now() WHERE id = $1`, fresh.ID)
	require.NoError(t, err)
	hourCount, err = postsDAO.CountPostsByOwnerSince(ctx, ownerID, time.Hour)
	require.NoError(t, err)
	require.Equal(t, int64(1), hourCount, "a deleted post still counts toward the cap")

	uniform := seed("uniform.jpg", "", "sfw", []string{"uniform"})
	mod := model.User{ID: fanB, Username: "fanb", Role: model.RoleModerator}
	_, err = tags.NewService(repository.NewTagPages(dao.NewTagPages(conn))).AddImplication(ctx, mod, "uniform", "skirt")
	require.NoError(t, err)
	implied, err := postsDAO.GetPost(ctx, uniform.ID)
	require.NoError(t, err)
	require.Equal(t, []string{"skirt", "uniform"}, implied.Tags)

	member := model.User{ID: fanA, Username: "fana", Role: model.RoleMember}
	reportSvc := reports.NewService(repository.NewReports(dao.NewReports(conn)))
	filed, err := reportSvc.Create(ctx, member, cute.ID, "spam", "wrong tags")
	require.NoError(t, err)
	open, err := reportSvc.ListOpen(ctx, mod, 40, 0)
	require.NoError(t, err)
	require.Equal(t, int64(1), open.Total)
	_, err = reportSvc.Dismiss(ctx, mod, filed.ID)
	require.NoError(t, err)
	var hiddenAt sql.NullTime
	require.NoError(t, conn.QueryRowContext(ctx, `SELECT hidden_at FROM posts WHERE id = $1`, cute.ID).Scan(&hiddenAt))
	require.False(t, hiddenAt.Valid, "dismissing a report leaves the post visible")

	staffJobs := jobs.NewService(repository.NewJobs(dao.NewJobs(conn)))
	failedPage, err := staffJobs.ListFailed(ctx, mod, "failed", 40, 0)
	require.NoError(t, err)
	require.Equal(t, int64(1), failedPage.Total)
	require.NoError(t, staffJobs.Retry(ctx, mod, id))
	require.NoError(t, conn.QueryRowContext(ctx, `SELECT state FROM jobs WHERE id = $1`, id).Scan(&state))
	require.Equal(t, "queued", state)

	require.NoError(t, goose.DownToContext(ctx, conn, "migrations", 0))
	require.NoError(t, db.Migrate(ctx, conn), "fresh schema can be rebuilt after rollback")
}

func tagNames(tags []model.TagCount) []string {
	out := make([]string, len(tags))
	for i, tag := range tags {
		out[i] = tag.Tag
	}
	return out
}

func idsOf(posts []model.Post) []string {
	out := make([]string, len(posts))
	for i, post := range posts {
		out[i] = post.ID
	}
	return out
}
