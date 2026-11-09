import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ApiError } from "../api/client";
import { addTagAlias, getTagPage, removeTagAlias, updateTagPage } from "../api/tagPages";
import { queryKeys } from "../api/queryKeys";

export function useTagPage(name: string) {
  const query = useQuery({
    queryKey: queryKeys.tagPage(name),
    queryFn: ({ signal }) => getTagPage(name, signal),
    retry: false,
  });
  return {
    ...query,
    missing: query.error instanceof ApiError && query.error.status === 404,
  };
}

export function useTagPageActions(name: string) {
  const client = useQueryClient();
  const refresh = async () => {
    await client.invalidateQueries({ queryKey: queryKeys.tagPages() });
  };
  const update = useMutation({
    mutationFn: (input: { category: string; description: string }) =>
      updateTagPage(name, input.category, input.description),
    onSuccess: refresh,
  });
  const addAlias = useMutation({
    mutationFn: (alias: string) => addTagAlias(name, alias),
    onSuccess: refresh,
  });
  const removeAlias = useMutation({
    mutationFn: (alias: string) => removeTagAlias(name, alias),
    onSuccess: refresh,
  });
  return { update, addAlias, removeAlias };
}
