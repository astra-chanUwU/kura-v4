import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { listNotifications, markNotificationsRead } from "../api/notifications";
import { useMe } from "./useKura";

const notificationsKey = ["notifications"] as const;

export function useNotifications(offset: number) {
  const { user } = useMe();
  return useQuery({
    queryKey: [...notificationsKey, "page", offset],
    queryFn: ({ signal }) => listNotifications(offset, signal),
    enabled: !!user,
    retry: false,
    placeholderData: keepPreviousData,
  });
}

export function useUnreadNotifications() {
  const { user } = useMe();
  const query = useQuery({
    queryKey: [...notificationsKey, "unread"],
    queryFn: ({ signal }) => listNotifications(0, signal, 1),
    enabled: !!user,
    retry: false,
    refetchInterval: 30_000,
  });
  return query.data?.unread ?? 0;
}

export function useMarkNotificationsRead() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (ids: string[]) => markNotificationsRead(ids),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: notificationsKey });
    },
  });
}
