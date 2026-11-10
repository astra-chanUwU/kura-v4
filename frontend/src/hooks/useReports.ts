import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createPostReport, dismissReport, listOpenReports } from "../api/reports";
import { queryKeys } from "../api/queryKeys";

export function useOpenReports() {
  const query = useQuery({
    queryKey: queryKeys.openReports(),
    queryFn: ({ signal }) => listOpenReports(signal),
    retry: false,
  });
  return {
    reports: query.data?.reports ?? [],
    total: query.data?.total ?? 0,
    loading: query.isPending,
    error: query.error,
  };
}

export function useReportActions() {
  const client = useQueryClient();
  const create = useMutation({
    mutationFn: ({ postId, reason, note }: { postId: string; reason: string; note: string }) =>
      createPostReport(postId, reason, note),
  });
  const dismiss = useMutation({
    mutationFn: (id: string) => dismissReport(id),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: queryKeys.openReports() });
    },
  });
  return { create, dismiss };
}
