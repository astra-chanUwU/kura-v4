package agentcli

import (
	"context"
	"testing"

	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/stretchr/testify/require"
)

type implyTags struct {
	Tags
	added string
}

func (f *implyTags) AddImplication(_ context.Context, actor model.User, antecedent, consequent string) (model.TagPage, error) {
	f.added = actor.Username + " " + antecedent + " " + consequent
	return model.TagPage{Tag: antecedent, Implies: []string{consequent}}, nil
}

func TestTagsImplyIsModeratorOnly(t *testing.T) {
	session, acc := postsSession(t, model.RoleMember)
	code, env, _ := postsRun(t, Deps{Accounts: acc, Tags: &implyTags{}}, []string{"tags", "imply", "school_uniform", "skirt"}, session)
	require.Equal(t, 1, code)
	require.Equal(t, "forbidden", env.Error.Code)

	session, acc = postsSession(t, model.RoleModerator)
	tags := &implyTags{}
	code, env, _ = postsRun(t, Deps{Accounts: acc, Tags: tags}, []string{"tags", "imply", "school_uniform", "skirt"}, session)
	require.Equal(t, 0, code)
	require.Equal(t, "tags.imply", env.Command)
	require.Equal(t, "ada school_uniform skirt", tags.added)
	require.Equal(t, []any{"skirt"}, postsData(t, env)["implies"])
}
