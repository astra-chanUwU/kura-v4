import { queryOptions, useQuery } from "@tanstack/react-query";
import { getSite } from "../api/site";
import { queryKeys } from "../api/queryKeys";

export const siteQueryOptions = () =>
  queryOptions({
    queryKey: queryKeys.site(),
    queryFn: ({ signal }) => getSite(signal),
    staleTime: 60_000,
  });

export function useSite() {
  const query = useQuery(siteQueryOptions());
  return {
    site: query.data,
    loading: query.isPending,
    error: query.error,
    retry: () => {
      void query.refetch();
    },
  };
}
