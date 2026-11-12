import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  acceptTagSuggestion,
  listTagSuggestions,
  rejectTagSuggestion,
  suggestTag,
} from "../api/suggestions";
import type { SuggestionKind, SuggestionStatus } from "../types/suggestion";
import { useMe } from "./useKura";
import { atLeast } from "../domain/roles";

const suggestionsKey = ["tag-alias-suggestions"] as const;

export function useTagSuggestions(status: SuggestionStatus) {
  return useQuery({
    queryKey: [...suggestionsKey, status],
    queryFn: ({ signal }) => listTagSuggestions(status, signal),
    retry: false,
  });
}

export function useSuggestTag(tag: string) {
  return useMutation({
    mutationFn: (body: { kind: SuggestionKind; other: string; reason: string }) =>
      suggestTag(tag, body),
  });
}

export function useSuggestionActions() {
  const client = useQueryClient();
  const refresh = () => {
    void client.invalidateQueries({ queryKey: suggestionsKey });
  };
  const accept = useMutation({
    mutationFn: acceptTagSuggestion,
    onSuccess: refresh,
  });
  const reject = useMutation({
    mutationFn: rejectTagSuggestion,
    onSuccess: refresh,
  });
  return { accept, reject };
}

export function useCanModerate() {
  const { user } = useMe();
  return !!user && atLeast(user.role, "moderator");
}
