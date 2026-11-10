package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCloseImplicationsFollowsTheChain(t *testing.T) {
	got := CloseImplications([]string{"pleated_skirt"}, []TagImplication{
		{Antecedent: "school_uniform", Consequent: "pleated_skirt"},
		{Antecedent: "pleated_skirt", Consequent: "skirt"},
		{Antecedent: "skirt", Consequent: "clothing"},
	})
	require.Equal(t, []string{"clothing", "pleated_skirt", "skirt"}, got)
}

func TestCloseImplicationsAddsConsequent(t *testing.T) {
	got := CloseImplications([]string{"school_uniform", "smile"}, []TagImplication{
		{Antecedent: "school_uniform", Consequent: "pleated_skirt"},
		{Antecedent: "pleated_skirt", Consequent: "skirt"},
	})
	require.Equal(t, []string{"pleated_skirt", "school_uniform", "skirt", "smile"}, got)
}
