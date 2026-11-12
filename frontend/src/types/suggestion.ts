export type SuggestionKind = "alias" | "implication";

export interface TagSuggestion {
  id: string;
  kind: SuggestionKind;
  source_tag: string;
  target_tag: string;
  reason: string;
  status: "pending" | "accepted" | "rejected";
  author_username: string;
  created_at: string;
}
