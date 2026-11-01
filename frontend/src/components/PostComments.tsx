import { Link } from "@tanstack/react-router";
import { useState } from "react";
import type { Comment } from "../types/comments";
import { useCommentActions, useComments } from "../hooks/useComments";
import { ConfirmDialog } from "./ConfirmDialog";
import styles from "./PostComments.module.css";
import { shortError, useToasts } from "./Toasts";

export function PostComments({ postId }: { postId: string }) {
  const { comments, loading, error, retry, signedIn } = useComments(postId);
  const actions = useCommentActions();
  const toasts = useToasts();
  const [draft, setDraft] = useState("");
  const [editingID, setEditingID] = useState<string | null>(null);
  const [editDraft, setEditDraft] = useState("");
  const [deleteID, setDeleteID] = useState<string | null>(null);
  const failed = error != null && comments.length === 0;

  function report(err: unknown) {
    toasts.push(shortError(err), "error");
  }

  return (
    <section className={styles.section} aria-labelledby="post-comments-title">
      <h2 id="post-comments-title">Comments</h2>
      {loading && <p className={styles.muted}>Loading comments…</p>}
      {failed && !loading && (
        <div className={styles.problem}>
          <p className={styles.error}>{shortError(error)}</p>
          <button type="button" className={styles.button} onClick={() => void retry()}>
            Retry
          </button>
        </div>
      )}
      {!loading && !failed && comments.length === 0 && (
        <p className={styles.muted}>No comments yet.</p>
      )}
      {!loading && comments.length > 0 && (
        <ul className={styles.list}>
          {comments.map((comment) => (
            <li key={comment.id} className={styles.row}>
              <CommentRow
                comment={comment}
                editing={editingID === comment.id}
                editDraft={editDraft}
                saving={actions.update.isPending}
                onEditDraft={setEditDraft}
                onStartEdit={() => {
                  setEditingID(comment.id);
                  setEditDraft(comment.body);
                }}
                onCancelEdit={() => setEditingID(null)}
                onSaveEdit={() => {
                  const body = editDraft.trim();
                  if (!body || actions.update.isPending) return;
                  actions.update.mutate(
                    { postId, id: comment.id, body },
                    {
                      onSuccess: () => setEditingID(null),
                      onError: report,
                    },
                  );
                }}
                onDelete={() => setDeleteID(comment.id)}
              />
            </li>
          ))}
        </ul>
      )}
      {signedIn && !loading && !failed && (
        <form
          className={styles.composer}
          onSubmit={(e) => {
            e.preventDefault();
            const body = draft.trim();
            if (!body || actions.create.isPending) return;
            actions.create.mutate(
              { postId, body },
              {
                onSuccess: () => setDraft(""),
                onError: report,
              },
            );
          }}
        >
          <textarea
            className={styles.input}
            maxLength={2000}
            rows={3}
            value={draft}
            aria-label="Write a comment"
            onChange={(e) => setDraft(e.target.value)}
          />
          <button
            type="submit"
            className={styles.primary}
            disabled={draft.trim() === "" || actions.create.isPending}
          >
            Post
          </button>
        </form>
      )}
      <ConfirmDialog
        open={deleteID !== null}
        title="Delete this comment?"
        message="This cannot be undone."
        confirmLabel="Delete"
        destructive
        onCancel={() => setDeleteID(null)}
        onConfirm={() => {
          if (deleteID === null || actions.delete.isPending) return;
          actions.delete.mutate(
            { postId, id: deleteID },
            {
              onSuccess: () => setDeleteID(null),
              onError: (err) => {
                setDeleteID(null);
                report(err);
              },
            },
          );
        }}
      />
    </section>
  );
}

function CommentRow({
  comment,
  editing,
  editDraft,
  saving,
  onEditDraft,
  onStartEdit,
  onCancelEdit,
  onSaveEdit,
  onDelete,
}: {
  comment: Comment;
  editing: boolean;
  editDraft: string;
  saving: boolean;
  onEditDraft: (value: string) => void;
  onStartEdit: () => void;
  onCancelEdit: () => void;
  onSaveEdit: () => void;
  onDelete: () => void;
}) {
  return (
    <>
      <div className={styles.meta}>
        {comment.author_avatar_url !== "" && (
          <img className={styles.avatar} src={comment.author_avatar_url} alt="" />
        )}
        <Link to="/u/$username" params={{ username: comment.author_username }}>
          {comment.author_username}
        </Link>
        <time dateTime={comment.created_at}>{new Date(comment.created_at).toLocaleString()}</time>
        {comment.updated_at !== comment.created_at && <span className={styles.edited}>edited</span>}
      </div>
      {editing ? (
        <form
          className={styles.composer}
          onSubmit={(e) => {
            e.preventDefault();
            onSaveEdit();
          }}
        >
          <textarea
            className={styles.input}
            maxLength={2000}
            rows={3}
            value={editDraft}
            aria-label="Edit comment"
            onChange={(e) => onEditDraft(e.target.value)}
          />
          <div className={styles.actions}>
            <button
              type="submit"
              className={styles.primary}
              disabled={editDraft.trim() === "" || saving}
            >
              Save
            </button>
            <button type="button" className={styles.button} onClick={onCancelEdit}>
              Cancel
            </button>
          </div>
        </form>
      ) : (
        <p className={styles.body}>{comment.body}</p>
      )}
      {(comment.can_edit || comment.can_delete) && !editing && (
        <div className={styles.actions}>
          {comment.can_edit && (
            <button type="button" className={styles.button} onClick={onStartEdit}>
              Edit
            </button>
          )}
          {comment.can_delete && (
            <button type="button" className={styles.danger} onClick={onDelete}>
              Delete
            </button>
          )}
        </div>
      )}
    </>
  );
}
