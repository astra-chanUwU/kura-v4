import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createSavedSearch, deleteSavedSearch, listSavedSearches } from "../api/savedSearches";
import type { CreateSavedSearch } from "../types/savedSearch";
import { useMe } from "./useKura";

const savedSearchesKey = ["saved-searches"] as const;

export function useSavedSearches() {
  const { signedIn } = useMe();
  return useQuery({
    queryKey: savedSearchesKey,
    queryFn: ({ signal }) => listSavedSearches(signal),
    enabled: signedIn,
    retry: false,
  });
}

export function useCreateSavedSearch() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (input: CreateSavedSearch) => createSavedSearch(input),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: savedSearchesKey });
    },
  });
}

export function useDeleteSavedSearch() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => deleteSavedSearch(id),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: savedSearchesKey });
    },
  });
}
