import { useEffect } from "react";

/**
 * After the current list page has loaded, warm page±1 so Next/Previous
 * feel instant. `prefetch` should call queryClient.prefetchQuery with the
 * same key/fn the list hook uses.
 */
export function usePrefetchPage(
  prefetch: (page: number) => void,
  page: number,
  pages: number,
  ready: boolean,
) {
  useEffect(() => {
    if (!ready) return;
    if (page > 1) prefetch(page - 1);
    if (page < pages) prefetch(page + 1);
  }, [prefetch, page, pages, ready]);
}
