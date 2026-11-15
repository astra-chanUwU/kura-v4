import { useQuery, useQueryClient } from "@tanstack/react-query";
import { ApiError } from "../api/client";
import { api, DuplicatePostError } from "../api/kura";
import { queryKeys } from "../api/queryKeys";
import { clearBoardMemory } from "../domain/boardMemory";
import type { Me, Post } from "../types/api";
import { invalidateSearch } from "./useBoardData";
import { useInvalidate } from "./useCollections";

export { passkeysSupported } from "../api/webauthn";

/** The existing post when an upload was a duplicate, else undefined. */
export function duplicateOf(err: unknown): Post | undefined {
  return err instanceof DuplicatePostError ? err.post : undefined;
}

export function useMe() {
  const query = useQuery({
    queryKey: queryKeys.me(),
    // A 401 is a signed-out answer, not an error to retry over: map it
    // to null so a stale cached user can never survive sign-out or an
    // expired session (React Query keeps old data on failed refetches).
    queryFn: async ({ signal }) => {
      try {
        return await api.me(signal);
      } catch (e) {
        if (e instanceof ApiError && e.status === 401) return null;
        throw e;
      }
    },
    retry: false,
    staleTime: 30_000,
  });
  return {
    me: query.data,
    user: query.data?.user,
    loading: query.isPending,
    signedIn: !!query.data,
    error: query.error,
  };
}

export function isSignedOut(error: unknown): boolean {
  return error instanceof ApiError && error.status === 401;
}

export function useAuthActions() {
  const client = useQueryClient();
  const invalidate = useInvalidate();
  // Drop everything that belongs to the previous account, then refresh
  // the public caches whose is_favorite flags depend on the viewer.
  const purgeUserCaches = () => {
    client.removeQueries({ queryKey: queryKeys.favorites() });
    client.removeQueries({ queryKey: queryKeys.myPosts() });
    client.removeQueries({ queryKey: queryKeys.myLists() });
  };
  const refreshMe = (me: Me) => {
    // Set the fresh session user directly instead of invalidating:
    // viewers update in the same render, with no stale in-between.
    purgeUserCaches();
    client.setQueryData(queryKeys.me(), me);
    invalidateSearch(client);
    void client.invalidateQueries({ queryKey: ["list"] });
    return me;
  };
  return {
    registerPassword: (username: string, password: string, displayName: string) =>
      api.registerPassword(username, password, displayName).then(refreshMe),
    loginPassword: (username: string, password: string) =>
      api.loginPassword(username, password).then(refreshMe),
    registerPasskey: (username: string, displayName: string) =>
      api.registerPasskey(username, displayName).then(refreshMe),
    loginPasskey: () => api.loginPasskey().then(refreshMe),
    recoverAccount: (username: string, recoveryKey: string, newPassword: string) =>
      api.recoverAccount(username, recoveryKey, newPassword).then(refreshMe),
    logout: () =>
      api.logout().then(() => {
        // Clear the user synchronously — invalidating alone leaves the
        // old data visible until (and unless) the refetch lands.
        // Drop the tab's board too, so the next account does not inherit it.
        clearBoardMemory();
        client.setQueryData(queryKeys.me(), null);
        purgeUserCaches();
        invalidate();
      }),
  };
}

export function useAccountActions() {
  const client = useQueryClient();
  const refresh = () => {
    void client.invalidateQueries({ queryKey: queryKeys.me() });
  };
  return {
    updateProfile: (body: {
      display_name: string;
      bio: string;
      location: string;
      website: string;
    }) => api.updateProfile(body).then(refresh),
    setPreferences: (showNsfw: boolean, over18: boolean) =>
      api.setPreferences(showNsfw, over18).then((me) => {
        refresh();
        invalidateSearch(client);
        return me;
      }),
    setTagPreferences: (includeTags: string[], excludeTags: string[]) =>
      api.setTagPreferences(includeTags, excludeTags).then(refresh),
    setAppearance: (theme: string, font: string) => api.setAppearance(theme, font).then(refresh),
    setPostsPerPage: (postsPerPage: number) =>
      api.setPostsPerPage(postsPerPage).then((me) => {
        refresh();
        return me;
      }),
    setAvatar: (file: File) => api.setAvatar(file).then(refresh),
    removeAvatar: () => api.removeAvatar().then(refresh),
    setPassword: (password: string) => api.setPassword(password).then(refresh),
    removePassword: () => api.removePassword().then(refresh),
    addPasskey: () => api.addPasskey().then(refresh),
    removePasskey: (id: string) => api.removePasskey(id).then(refresh),
    regenerateRecoveryKey: () => api.regenerateRecoveryKey(),
  };
}
