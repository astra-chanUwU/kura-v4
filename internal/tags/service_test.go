package tags

import (
	"context"
	"testing"
	"time"

	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/stretchr/testify/require"
)

type fakeRepo struct {
	pages       map[string]model.TagWiki
	aliases     map[string]string
	counts      map[string]int64
	posts       map[string][]model.Post
	moved       []string
	lastExclude []string
	edges       []model.TagImplication
	postTags    map[string]bool
	backfilled  []string
}

func (f *fakeRepo) GetPage(_ context.Context, tag string) (model.TagWiki, bool, error) {
	wiki, ok := f.pages[tag]
	return wiki, ok, nil
}
func (f *fakeRepo) UpsertPage(_ context.Context, tag, category, description, userID string) error {
	if f.pages == nil {
		f.pages = map[string]model.TagWiki{}
	}
	f.pages[tag] = model.TagWiki{Tag: tag, Category: category, Description: description, UpdatedByUsername: userID, UpdatedAt: time.Unix(10, 0).UTC()}
	return nil
}
func (f *fakeRepo) CountPosts(_ context.Context, tag string, _ bool, exclude []string) (int64, error) {
	f.lastExclude = exclude
	return f.counts[tag], nil
}
func (f *fakeRepo) RecentPosts(_ context.Context, tag string, _ bool, exclude []string) ([]model.Post, error) {
	f.lastExclude = exclude
	return f.posts[tag], nil
}
func (f *fakeRepo) AliasTarget(_ context.Context, alias string) (string, bool, error) {
	target, ok := f.aliases[alias]
	return target, ok, nil
}
func (f *fakeRepo) AliasesOf(_ context.Context, target string) ([]string, error) {
	var out []string
	for alias, to := range f.aliases {
		if to == target {
			out = append(out, alias)
		}
	}
	return out, nil
}
func (f *fakeRepo) IsAliasTarget(_ context.Context, tag string) (bool, error) {
	for _, target := range f.aliases {
		if target == tag {
			return true, nil
		}
	}
	return false, nil
}
func (f *fakeRepo) CreateAlias(_ context.Context, alias, target, _ string) error {
	if f.aliases == nil {
		f.aliases = map[string]string{}
	}
	f.aliases[alias] = target
	f.moved = append(f.moved, alias+"->"+target)
	return nil
}
func (f *fakeRepo) ListImplications(context.Context) ([]model.TagImplication, error) {
	return f.edges, nil
}
func (f *fakeRepo) CreateImplication(_ context.Context, antecedent, consequent, _ string, tags []string) error {
	f.edges = append(f.edges, model.TagImplication{Antecedent: antecedent, Consequent: consequent})
	f.backfilled = append([]string{}, tags...)
	if f.postTags == nil {
		f.postTags = map[string]bool{}
	}
	for _, tag := range tags {
		f.postTags[tag] = true
	}
	return nil
}
func (f *fakeRepo) DeleteImplication(_ context.Context, antecedent, consequent string) (int64, error) {
	kept := make([]model.TagImplication, 0, len(f.edges))
	var n int64
	for _, edge := range f.edges {
		if edge.Antecedent == antecedent && edge.Consequent == consequent {
			n++
			continue
		}
		kept = append(kept, edge)
	}
	f.edges = kept
	return n, nil
}
func (f *fakeRepo) DeleteAlias(_ context.Context, alias, target string) (int64, error) {
	if f.aliases[alias] != target {
		return 0, nil
	}
	delete(f.aliases, alias)
	return 1, nil
}

func TestGetUnknownTag(t *testing.T) {
	_, err := NewService(&fakeRepo{}).Get(context.Background(), nil, "no_such_tag")
	require.ErrorIs(t, err, ErrNotFound)
}

func TestGetRejectsBlankTag(t *testing.T) {
	_, err := NewService(&fakeRepo{}).Get(context.Background(), nil, "---")
	var vErr *ValidationError
	require.ErrorAs(t, err, &vErr)
	require.Equal(t, tagMessage, vErr.Message)
}

func TestGetPageWithoutWiki(t *testing.T) {
	repo := &fakeRepo{counts: map[string]int64{"silver_hair": 3}, posts: map[string][]model.Post{"silver_hair": {{ID: "p1"}}}}
	page, err := NewService(repo).Get(context.Background(), nil, "Silver-Hair")
	require.NoError(t, err)
	require.Equal(t, "silver_hair", page.Tag)
	require.Empty(t, page.AliasOf)
	require.Equal(t, "general", page.Category)
	require.Equal(t, int64(3), page.PostCount)
	require.Len(t, page.Posts, 1)
}

func TestAliasPageUsesCanonicalWiki(t *testing.T) {
	repo := &fakeRepo{
		aliases: map[string]string{"grey_hair": "gray_hair"},
		pages: map[string]model.TagWiki{
			"gray_hair": {Category: "general", Description: "Hair that is gray.", UpdatedByUsername: "ada"},
		},
		counts: map[string]int64{"gray_hair": 4},
	}
	page, err := NewService(repo).Get(context.Background(), nil, "grey_hair")
	require.NoError(t, err)
	require.Equal(t, "grey_hair", page.Tag)
	require.Equal(t, "gray_hair", page.AliasOf)
	require.Equal(t, "Hair that is gray.", page.Description)
	require.Equal(t, int64(4), page.PostCount)
	require.Contains(t, page.Aliases, "grey_hair")
}

func TestUpdateOnAliasWritesCanonical(t *testing.T) {
	repo := &fakeRepo{aliases: map[string]string{"grey_hair": "gray_hair"}}
	member := model.User{ID: "u1", Role: model.RoleMember}
	page, err := NewService(repo).Update(context.Background(), member, "grey_hair", "general", "Gray hair.")
	require.NoError(t, err)
	require.Equal(t, "gray_hair", page.AliasOf)
	require.Equal(t, "Gray hair.", repo.pages["gray_hair"].Description)
	require.Equal(t, "u1", repo.pages["gray_hair"].UpdatedByUsername)
}

func TestUpdateRejectsCategoryAndLength(t *testing.T) {
	svc := NewService(&fakeRepo{})
	member := model.User{ID: "u1", Role: model.RoleMember}
	_, err := svc.Update(context.Background(), member, "cat", "species", "A cat.")
	var vErr *ValidationError
	require.ErrorAs(t, err, &vErr)
	require.Equal(t, categoryMessage, vErr.Message)

	_, err = svc.Update(context.Background(), member, "cat", "general", stringsOf(2001))
	require.ErrorAs(t, err, &vErr)
}

func TestAddAliasRules(t *testing.T) {
	repo := &fakeRepo{aliases: map[string]string{"grey_hair": "gray_hair"}}
	svc := NewService(repo)
	member := model.User{ID: "m", Role: model.RoleMember}
	mod := model.User{ID: "mod", Role: model.RoleModerator}

	_, err := svc.AddAlias(context.Background(), member, "gray_hair", "grayhair")
	require.ErrorIs(t, err, ErrForbidden)

	_, err = svc.AddAlias(context.Background(), mod, "gray_hair", "gray_hair")
	require.Error(t, err)

	_, err = svc.AddAlias(context.Background(), mod, "grey_hair", "ash_hair")
	require.Error(t, err)
	require.Contains(t, err.Error(), "alias of gray_hair")

	_, err = svc.AddAlias(context.Background(), mod, "blue_hair", "grey_hair")
	require.Error(t, err)
	require.Contains(t, err.Error(), "already an alias")

	_, err = svc.AddAlias(context.Background(), mod, "blue_hair", "gray_hair")
	require.Error(t, err)
	require.Contains(t, err.Error(), "already point")

	page, err := svc.AddAlias(context.Background(), mod, "gray_hair", "silver_grey")
	require.NoError(t, err)
	require.Equal(t, "gray_hair", page.Tag)
	require.Contains(t, page.Aliases, "silver_grey")
	require.Contains(t, repo.moved, "silver_grey->gray_hair")
}

func TestRemoveAlias(t *testing.T) {
	repo := &fakeRepo{aliases: map[string]string{"grey_hair": "gray_hair"}, counts: map[string]int64{"gray_hair": 1}}
	mod := model.User{ID: "mod", Role: model.RoleModerator}
	page, err := NewService(repo).RemoveAlias(context.Background(), mod, "gray_hair", "grey_hair")
	require.NoError(t, err)
	require.Empty(t, page.Aliases)
	_, err = NewService(repo).RemoveAlias(context.Background(), mod, "gray_hair", "grey_hair")
	require.ErrorIs(t, err, ErrNotFound)
}

func TestGetPassesViewerExcludeTags(t *testing.T) {
	repo := &fakeRepo{counts: map[string]int64{"cat": 2}, posts: map[string][]model.Post{"cat": {{ID: "p1"}}}}
	viewer := &model.User{DefaultExcludeTags: []string{"gore"}}
	_, err := NewService(repo).Get(context.Background(), viewer, "cat")
	require.NoError(t, err)
	require.Equal(t, []string{"gore"}, repo.lastExclude)

	_, err = NewService(repo).Get(context.Background(), nil, "cat")
	require.NoError(t, err)
	require.NotNil(t, repo.lastExclude)
	require.Empty(t, repo.lastExclude)

	_, err = NewService(repo).Get(context.Background(), &model.User{}, "cat")
	require.NoError(t, err)
	require.NotNil(t, repo.lastExclude)
	require.Empty(t, repo.lastExclude)
}

func TestImplications(t *testing.T) {
	mod := model.User{ID: "mod", Role: model.RoleModerator}
	member := model.User{ID: "mem", Role: model.RoleMember}
	repo := &fakeRepo{counts: map[string]int64{"school_uniform": 1}}
	svc := NewService(repo)

	_, err := svc.AddImplication(context.Background(), member, "school_uniform", "pleated_skirt")
	require.ErrorIs(t, err, ErrForbidden)

	_, err = svc.AddImplication(context.Background(), mod, "school_uniform", "school_uniform")
	require.Error(t, err)
	require.Contains(t, err.Error(), "itself")

	page, err := svc.AddImplication(context.Background(), mod, "school_uniform", "pleated_skirt")
	require.NoError(t, err)
	require.Equal(t, []string{"pleated_skirt"}, page.Implies)
	require.Equal(t, []string{"pleated_skirt"}, repo.backfilled)
	require.True(t, repo.postTags["pleated_skirt"])

	_, err = svc.AddImplication(context.Background(), mod, "school_uniform", "pleated_skirt")
	require.Error(t, err)
	require.Contains(t, err.Error(), "already exists")

	_, err = svc.AddImplication(context.Background(), mod, "pleated_skirt", "school_uniform")
	require.Error(t, err)
	require.Contains(t, err.Error(), "loop")

	page, err = svc.RemoveImplication(context.Background(), mod, "school_uniform", "pleated_skirt")
	require.NoError(t, err)
	require.Empty(t, page.Implies)
	require.True(t, repo.postTags["pleated_skirt"], "removing the rule leaves the consequent on posts")
}

func TestAddImplicationBackfillsTheChain(t *testing.T) {
	mod := model.User{ID: "mod", Role: model.RoleModerator}
	repo := &fakeRepo{
		counts: map[string]int64{"school_uniform": 2},
		edges: []model.TagImplication{
			{Antecedent: "other_uniform", Consequent: "pleated_skirt"},
			{Antecedent: "pleated_skirt", Consequent: "skirt"},
			{Antecedent: "skirt", Consequent: "clothing"},
			{Antecedent: "school_uniform", Consequent: "uniform"},
		},
	}
	page, err := NewService(repo).AddImplication(context.Background(), mod, "school_uniform", "pleated_skirt")
	require.NoError(t, err)
	require.Equal(t, []string{"pleated_skirt", "uniform"}, page.Implies)
	require.Equal(t, []string{"clothing", "pleated_skirt", "skirt"}, repo.backfilled)
	require.NotContains(t, repo.backfilled, "school_uniform")
	require.NotContains(t, repo.backfilled, "other_uniform")
	require.NotContains(t, repo.backfilled, "uniform")
}

func stringsOf(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = 'a'
	}
	return string(b)
}
