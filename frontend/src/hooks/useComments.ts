import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createComment, deleteComment, listComments, updateComment } from "../api/comments";
import { queryKeys } from "../api/queryKeys";
import { invalidatePostSurfaces } from "./useBoardData";
import { useMe } from "./useSession";

export function useComments(postId: string) {
  const { signedIn } = useMe();
  const query = useQuery({
    queryKey: queryKeys.comments(postId),
    queryFn: ({ signal }) => listComments(postId, signal),
  });
  return {
    comments: query.data?.comments ?? [],
    total: query.data?.total ?? 0,
    loading: query.isPending,
    error: query.error,
    retry: () => void query.refetch(),
    signedIn,
  };
}

export function useCommentActions() {
  const client = useQueryClient();
  const invalidate = (postId: string) => {
    invalidatePostSurfaces(client, postId);
  };
  const create = useMutation({
    mutationFn: ({ postId, body }: { postId: string; body: string }) => createComment(postId, body),
    onSuccess: (_comment, input) => invalidate(input.postId),
  });
  const update = useMutation({
    mutationFn: ({ id, body }: { postId: string; id: string; body: string }) =>
      updateComment(id, body),
    onSuccess: (_comment, input) => invalidate(input.postId),
  });
  const remove = useMutation({
    mutationFn: ({ id }: { postId: string; id: string }) => deleteComment(id),
    onSuccess: (_void, input) => invalidate(input.postId),
  });
  return { create, update, delete: remove };
}
