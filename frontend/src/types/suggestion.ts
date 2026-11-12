export type SuggestionKind = "alias" | "implication";
export type SuggestionStatus = "pending" | "accepted" | "rejected";

export interface TagSuggestion {
  id: string;
  kind: SuggestionKind;
  source_tag: string;
  target_tag: string;
  reason: string;
  status: SuggestionStatus;
  author_username: string;
  resolver_username?: string;
  created_at: string;
  resolved_at?: string;
}
