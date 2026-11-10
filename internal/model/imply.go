package model

import "sort"

// CloseImplications returns tags plus every tag they imply, directly or
// through a chain. The result is sorted and has no duplicates. An empty
// name is skipped. edges may include a rule that is not stored yet.
func CloseImplications(tags []string, edges []TagImplication) []string {
	next := map[string][]string{}
	for _, edge := range edges {
		next[edge.Antecedent] = append(next[edge.Antecedent], edge.Consequent)
	}
	seen := map[string]bool{}
	out := make([]string, 0, len(tags))
	var walk func(string)
	walk = func(tag string) {
		if tag == "" || seen[tag] {
			return
		}
		seen[tag] = true
		out = append(out, tag)
		for _, child := range next[tag] {
			walk(child)
		}
	}
	for _, tag := range tags {
		walk(tag)
	}
	sort.Strings(out)
	return out
}
