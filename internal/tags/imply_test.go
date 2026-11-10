package tags

import (
	"testing"

	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/stretchr/testify/require"
)

func TestCloseImplicationsFollowsTheChain(t *testing.T) {
	got := CloseImplications([]string{"pleated_skirt"}, []model.TagImplication{
		{Antecedent: "school_uniform", Consequent: "pleated_skirt"},
		{Antecedent: "pleated_skirt", Consequent: "skirt"},
		{Antecedent: "skirt", Consequent: "clothing"},
	})
	require.Equal(t, []string{"clothing", "pleated_skirt", "skirt"}, got)
}

func TestCloseImplicationsAddsConsequent(t *testing.T) {
	got := CloseImplications([]string{"school_uniform", "smile"}, []model.TagImplication{
		{Antecedent: "school_uniform", Consequent: "pleated_skirt"},
		{Antecedent: "pleated_skirt", Consequent: "skirt"},
	})
	require.Equal(t, []string{"pleated_skirt", "school_uniform", "skirt", "smile"}, got)
}
