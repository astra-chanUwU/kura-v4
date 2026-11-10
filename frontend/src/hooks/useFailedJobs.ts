import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "../api/kura";
import { queryKeys } from "../api/queryKeys";

export function useFailedJobs(enabled: boolean, offset: number) {
  const client = useQueryClient();
  const query = useQuery({
    queryKey: queryKeys.failedJobs(offset),
    queryFn: ({ signal }) => api.failedJobs(offset, signal),
    enabled,
    retry: false,
  });
  const retry = useMutation({
    mutationFn: (id: number) => api.retryFailedJob(id),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: queryKeys.failedJobs() });
    },
  });
  return { ...query, retry };
}
