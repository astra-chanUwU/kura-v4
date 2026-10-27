package app

import (
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/astra-chanUwU/kura-v4/internal/accounts"
	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/stretchr/testify/require"
)

func TestParseSidecarTags(t *testing.T) {
	require.Equal(t, []string{"1girl", "long_hair", "silver_hair", "yuri", "smile"}, parseSidecarTags("1girl long_hair silver_hair yuri smile\n"))
	require.Equal(t, []string{"1girl", "long_hair"}, parseSidecarTags("1girl long_hair\r\nnot_a_tag extra\n"))
	require.Nil(t, parseSidecarTags(""))
	require.Nil(t, parseSidecarTags("   \nsecond"))
	require.Equal(t, []string{"smile", "yuri"}, parseSidecarTags("  smile   yuri  "))
}

func TestReadSidecarTagsMissingAndPresent(t *testing.T) {
	dir := t.TempDir()
	img := filepath.Join(dir, "safebooru 7269708.jpg")
	require.NoError(t, os.WriteFile(img, []byte("not-an-image"), 0o644))

	tags, ok, err := readSidecarTags(img)
	require.NoError(t, err)
	require.False(t, ok)
	require.Nil(t, tags)

	require.NoError(t, os.WriteFile(sidecarPath(img), []byte("1girl long_hair silver_hair yuri smile\n"), 0o644))
	tags, ok, err = readSidecarTags(img)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, []string{"1girl", "long_hair", "silver_hair", "yuri", "smile"}, tags)
}

func TestTagsForSeedFileSidecarAndFallback(t *testing.T) {
	sha := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	want := seedTags("/pics/Anime", "image/png", sha, []string{"twitter"})
	got := tagsForSeedFile("/pics/Anime", "image/png", sha, []string{"twitter"}, []string{"ignored"}, false)
	require.Equal(t, want, got)

	got = tagsForSeedFile("/pics/Anime", "image/png", sha, []string{"twitter"}, []string{"1girl", "Long-Hair"}, true)
	require.Equal(t, []string{"1girl", "long_hair", "seed"}, got)

	many := make([]string, 40)
	for i := range many {
		many[i] = fmt.Sprintf("tag_%02d", i)
	}
	capped := tagsForSeedFile("dir", "image/jpeg", sha, nil, many, true)
	require.Contains(t, capped, "seed")
	require.LessOrEqual(t, len(capped), 30)
}

func TestAssignOwnersDeterministicAndWeighted(t *testing.T) {
	weights := demoAccountWeights()
	first := assignOwners(5000, weights, demoRand())
	second := assignOwners(5000, weights, demoRand())
	require.Equal(t, first, second)

	counts := make([]int, len(weights))
	for _, idx := range first {
		counts[idx]++
	}
	hana, sora, ren, astra := 4, 7, 5, 0
	require.Greater(t, counts[hana], counts[sora]*3)
	require.Greater(t, counts[ren], counts[astra])
	require.Empty(t, assignOwners(0, weights, demoRand()))
}

func TestEvaluateDemoGuard(t *testing.T) {
	require.NoError(t, evaluateDemoGuard(0, "http://localhost:5174"))
	require.NoError(t, evaluateDemoGuard(0, "http://127.0.0.1:8094"))
	require.NoError(t, evaluateDemoGuard(0, "http://LOCALHOST:5174/kura"))

	err := evaluateDemoGuard(3, "http://localhost:5174")
	require.Error(t, err)
	require.Contains(t, err.Error(), "kura demo refused")
	require.Contains(t, err.Error(), "zero users")
	require.Contains(t, err.Error(), "3")
	require.NotContains(t, err.Error(), "public URL")

	err = evaluateDemoGuard(0, "https://kura.example")
	require.Error(t, err)
	require.Contains(t, err.Error(), "localhost")
	require.NotContains(t, err.Error(), "zero users")

	err = evaluateDemoGuard(2, "https://kura.example")
	require.Error(t, err)
	require.Contains(t, err.Error(), "localhost")
	require.Contains(t, err.Error(), "zero users")

	require.False(t, demoPublicHostAllowed("http://127.0.0.2:5174"))
	require.False(t, demoPublicHostAllowed("http://localhost.example.com"))
	require.False(t, demoPublicHostAllowed("http://[::1]:5174"))
	require.False(t, demoPublicHostAllowed(""))
}

func TestDemoRoster(t *testing.T) {
	require.Len(t, demoAccounts, 8)
	require.GreaterOrEqual(t, len(demoPassword), 15)
	require.LessOrEqual(t, len(demoPassword), 128)

	names := map[string]bool{}
	pairs := map[string]bool{}
	var super, admin, mod, member, nsfw, prefs, avatars int
	for _, account := range demoAccounts {
		require.False(t, names[account.Username])
		names[account.Username] = true
		require.NotContains(t, strings.ToLower(demoPassword), account.Username)
		require.Contains(t, accounts.Themes, account.Theme)
		require.Contains(t, accounts.Fonts, account.Font)
		pair := account.Theme + "\x00" + account.Font
		require.False(t, pairs[pair])
		pairs[pair] = true
		require.Greater(t, account.Weight, 0)
		require.GreaterOrEqual(t, account.FavMin, 15)
		require.LessOrEqual(t, account.FavMax, 80)
		require.LessOrEqual(t, account.FavMin, account.FavMax)
		require.NotEmpty(t, account.DisplayName)
		require.LessOrEqual(t, len([]rune(account.DisplayName)), 60)
		require.LessOrEqual(t, len([]rune(account.Bio)), 500)
		require.LessOrEqual(t, len([]rune(account.Location)), 80)
		switch account.Role {
		case model.RoleSuperAdmin:
			super++
		case model.RoleAdmin:
			admin++
		case model.RoleModerator:
			mod++
		case model.RoleMember:
			member++
		default:
			t.Fatalf("unexpected role %s", account.Role)
		}
		if account.ShowNSFW {
			nsfw++
		}
		if len(account.Include) > 0 || len(account.Exclude) > 0 {
			prefs++
		}
		if account.Avatar {
			avatars++
		}
	}
	require.Equal(t, 1, super)
	require.Equal(t, 1, admin)
	require.Equal(t, 2, mod)
	require.Equal(t, 4, member)
	require.GreaterOrEqual(t, nsfw, 2)
	require.LessOrEqual(t, nsfw, 3)
	require.Equal(t, 4, prefs)
	require.GreaterOrEqual(t, avatars, 6)
	require.Less(t, avatars, len(demoAccounts))

	byName := map[string]demoAccount{}
	for _, account := range demoAccounts {
		byName[account.Username] = account
	}
	require.Equal(t, model.RoleSuperAdmin, byName["astra"].Role)
	require.Equal(t, model.RoleAdmin, byName["noa"].Role)
	require.True(t, byName["noa"].ExtraRoleHistory)
	require.False(t, byName["astra"].ExtraRoleHistory)
	require.Greater(t, byName["hana"].Weight, byName["sora"].Weight)
	require.Greater(t, byName["ren"].Weight, byName["astra"].Weight)
	require.Greater(t, byName["hana"].FavMin, byName["sora"].FavMax)
}

func TestDemoListCatalog(t *testing.T) {
	require.GreaterOrEqual(t, len(demoListCatalog), 4)
	seen := map[string]bool{}
	for _, item := range demoListCatalog {
		require.NotEmpty(t, item.Title)
		require.LessOrEqual(t, len([]rune(item.Title)), 80)
		require.LessOrEqual(t, len([]rune(item.Desc)), 300)
		require.False(t, seen[item.Title])
		seen[item.Title] = true
	}
	for user := range demoAccounts {
		titles := map[string]bool{}
		var public, private int
		for list := 0; list < 4; list++ {
			title := demoListAt(user, list).Title
			require.False(t, titles[title])
			titles[title] = true
			if demoListVisibility(user, list) == "public" {
				public++
			} else {
				private++
			}
		}
		require.Positive(t, public)
		require.Positive(t, private)
	}
}

func TestDemoNSFWCount(t *testing.T) {
	require.Equal(t, 0, demoNSFWCount(0))
	require.Equal(t, 0, demoNSFWCount(12))
	require.Equal(t, 1, demoNSFWCount(13))
	require.Equal(t, 16, demoNSFWCount(200))
}

func TestPickModerationTargetsAreDisjoint(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	hide, del := pickModerationTargets(30, 5, 3, 4, rng)
	require.Len(t, hide, 5)
	require.Len(t, del, 3)
	seen := map[int]bool{4: true}
	for _, idx := range hide {
		require.False(t, seen[idx])
		seen[idx] = true
	}
	for _, idx := range del {
		require.False(t, seen[idx])
		seen[idx] = true
	}
	again := rand.New(rand.NewPCG(1, 2))
	hide2, del2 := pickModerationTargets(30, 5, 3, 4, again)
	require.Equal(t, hide, hide2)
	require.Equal(t, del, del2)

	hide, del = pickModerationTargets(3, 5, 3, 0, rand.New(rand.NewPCG(3, 4)))
	require.LessOrEqual(t, len(hide)+len(del), 2)
	for _, idx := range append(hide, del...) {
		require.NotEqual(t, 0, idx)
	}
}

func TestDemoMetaEdit(t *testing.T) {
	post := model.Post{Artist: "", Tags: []string{"1girl", "seed", "smile"}, Rating: "sfw"}
	artist, tags, ok := demoMetaEdit(post, 0)
	require.True(t, ok)
	require.Empty(t, artist)
	require.Contains(t, tags, "blush")
	require.Contains(t, tags, "seed")

	artist, tags, ok = demoMetaEdit(post, 1)
	require.True(t, ok)
	require.Empty(t, artist)
	require.Contains(t, tags, "seed")
	require.Len(t, tags, 2)

	artist, tags, ok = demoMetaEdit(post, 2)
	require.True(t, ok)
	require.NotEmpty(t, artist)
	require.Equal(t, post.Tags, tags)

	capped := []string{"seed"}
	for i := 0; i < 29; i++ {
		capped = append(capped, fmt.Sprintf("a%02d", i))
	}
	added, ok := demoAddTag(capped, 0)
	require.True(t, ok)
	require.Contains(t, added, "seed")
	require.Len(t, added, 30)
	require.NotEqual(t, capped, added)
}

func TestDemoCredentialsLine(t *testing.T) {
	line := demoCredentialsLine([]model.User{
		{Username: "astra", Role: model.RoleSuperAdmin},
		{Username: "noa", Role: model.RoleAdmin},
	})
	require.Equal(t, "demo accounts: astra super_admin, noa admin; password: "+demoPassword, line)
	require.NotContains(t, line, "\n")
}
