import { useEffect, useRef, useState } from "react";
import { Link } from "@tanstack/react-router";
import { duplicateOf, useInvalidate, useMe, usePostActions } from "../hooks/useKura";
import type { Post } from "../types/api";
import { ThemedSelect } from "./ThemedSelect";
import { shortError, useToasts } from "./Toasts";
import styles from "./UploadDialog.module.css";

type Item = {
  id: number;
  file: File;
  preview: string;
  status: "queued" | "uploading" | "done" | "error";
  error?: string;
  duplicate?: Post;
};

const ACCEPT = "image/*,video/mp4,video/webm,video/quicktime";

/**
 * Upload dialog: pick or drop files, preview them, add shared tags,
 * upload one by one. Duplicates and unsupported files are reported per
 * file, never counted as added. On full success the dialog closes
 * itself and the caller announces where the pictures landed.
 */
export function UploadDialog({
  initialFiles,
  onClose,
  onUploaded,
}: {
  initialFiles: File[];
  onClose: () => void;
  onUploaded?: (count: number, hiddenFromUploader: boolean) => void;
}) {
  const { upload } = usePostActions();
  const { me } = useMe();
  const toasts = useToasts();
  const invalidate = useInvalidate();
  const [items, setItems] = useState<Item[]>([]);
  const [tags, setTags] = useState("");
  const [rating, setRating] = useState("sfw");
  const [dragging, setDragging] = useState(false);
  const [busy, setBusy] = useState(false);
  const [finished, setFinished] = useState(false);
  const [skipped, setSkipped] = useState("");
  const inputRef = useRef<HTMLInputElement>(null);
  const nextID = useRef(0);
  const closeTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  const addFiles = (files: File[]) => {
    const media = files.filter((f) => f.type.startsWith("image/") || f.type.startsWith("video/"));
    const rejected = files.length - media.length;
    setSkipped(
      rejected > 0
        ? `${rejected} file${rejected === 1 ? " was" : "s were"} skipped — only pictures and videos can be added.`
        : "",
    );
    setFinished(false);
    setItems((prev) => [
      ...prev,
      ...media.map(
        (file) =>
          ({
            id: ++nextID.current,
            file,
            preview: URL.createObjectURL(file),
            status: "queued",
          }) as Item,
      ),
    ]);
  };

  useEffect(() => {
    if (initialFiles.length > 0) addFiles(initialFiles);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape" && !busy) onClose();
    };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [onClose, busy]);

  useEffect(() => {
    return () => {
      if (closeTimer.current) clearTimeout(closeTimer.current);
      for (const item of items) URL.revokeObjectURL(item.preview);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const queued = items.filter((i) => i.status === "queued" || i.status === "error");
  const doneCount = items.filter((i) => i.status === "done").length;
  const failedCount = items.filter((i) => i.status === "error").length;
  // An NSFW upload is real, but invisible to its own uploader until
  // they opt in — say so instead of letting it "vanish".
  const hiddenFromUploader = rating === "nsfw" && !!me && !me.preferences.show_nsfw;

  const uploadAll = () => {
    if (queued.length === 0) return;
    setBusy(true);
    setFinished(false);
    void (async () => {
      let added = 0;
      for (const item of queued) {
        setItems((prev) =>
          prev.map((i) => (i.id === item.id ? { ...i, status: "uploading" as const } : i)),
        );
        try {
          await upload(item.file, tags, rating);
          added++;
          setItems((prev) =>
            prev.map((i) => (i.id === item.id ? { ...i, status: "done" as const } : i)),
          );
        } catch (e) {
          const duplicate = duplicateOf(e);
          const short = duplicate ? "Already on the board." : shortError(e, "Upload failed.");
          toasts.push(`${item.file.name} — ${short}`, "error");
          setItems((prev) =>
            prev.map((i) =>
              i.id === item.id ? { ...i, status: "error" as const, error: short, duplicate } : i,
            ),
          );
        }
      }
      invalidate();
      setBusy(false);
      setFinished(true);
      const failed = queued.length - added;
      if (added > 0 && failed === 0) {
        // Brief success beat, then get out of the way: the board behind
        // the dialog already shows the new posts at the top.
        closeTimer.current = setTimeout(() => {
          onUploaded?.(added, hiddenFromUploader);
          onClose();
        }, 1100);
      }
    })();
  };

  const closeNow = () => {
    if (busy) return;
    if (finished && doneCount > 0) onUploaded?.(doneCount, hiddenFromUploader);
    onClose();
  };

  return (
    <div
      className={styles.backdrop}
      role="presentation"
      onMouseDown={(e) => {
        if (e.target === e.currentTarget) closeNow();
      }}
    >
      {/* oxlint-disable-next-line jsx-a11y/no-noninteractive-element-interactions -- whole panel is a file drop target */}
      <div
        className={styles.panel}
        role="dialog"
        aria-modal="true"
        aria-label="Upload pictures"
        onDragOver={(e) => {
          e.preventDefault();
          setDragging(true);
        }}
        onDragLeave={() => setDragging(false)}
        onDrop={(e) => {
          e.preventDefault();
          setDragging(false);
          addFiles([...e.dataTransfer.files]);
        }}
      >
        <div className={styles.header}>
          <h2>Add pictures to the board</h2>
          <button type="button" onClick={closeNow} disabled={busy} aria-label="Close upload">
            ✕
          </button>
        </div>

        {items.length === 0 ? (
          <button
            type="button"
            className={`${styles.dropzone} ${dragging ? styles.dragging : ""}`}
            onClick={() => inputRef.current?.click()}
          >
            Drop pictures or videos here, or click to choose
            <span className={styles.dropHint}>
              Images, GIFs, MP4/WebM — a picture that is already on the board is reported, not added
              twice
            </span>
          </button>
        ) : (
          <div className={`${styles.addMore} ${dragging ? styles.dragging : ""}`}>
            <span>
              {items.length} selected · they go to the <strong>top of the board</strong> and to{" "}
              <strong>Me → Contributions</strong>
            </span>
            <button type="button" onClick={() => inputRef.current?.click()} disabled={busy}>
              Add more…
            </button>
          </div>
        )}
        <input
          ref={inputRef}
          type="file"
          accept={ACCEPT}
          multiple
          hidden
          aria-label="Choose files to upload"
          onChange={(e) => {
            addFiles([...(e.target.files ?? [])]);
            e.target.value = "";
          }}
        />

        {items.length > 0 && (
          <div className={styles.previews}>
            {items.map((item) => (
              <figure key={item.id} className={styles.preview}>
                {item.file.type.startsWith("video/") ? (
                  <video src={item.preview} muted playsInline preload="metadata" />
                ) : (
                  <img src={item.preview} alt="" />
                )}
                <figcaption>
                  <span className={styles.fileName}>{item.file.name}</span>
                  <span
                    className={`${styles.status} ${item.status === "error" ? styles.errorStatus : ""}`}
                  >
                    {item.status === "done"
                      ? "Added ✓"
                      : item.status === "uploading"
                        ? "Uploading…"
                        : item.status === "error"
                          ? (item.error ?? "Failed")
                          : "Queued"}
                  </span>
                  {item.duplicate && (
                    <Link
                      to="/posts/$id"
                      params={{ id: item.duplicate.id }}
                      className={styles.duplicateLink}
                      onClick={onClose}
                    >
                      <img src={item.duplicate.thumbnail_url} alt="" />
                      <span>
                        Already here
                        {item.duplicate.tags.length > 0
                          ? `: ${item.duplicate.tags.slice(0, 3).join(" ")}`
                          : ""}{" "}
                        — view it →
                      </span>
                    </Link>
                  )}
                </figcaption>
                {(item.status === "queued" || item.status === "error") && !busy && (
                  <button
                    type="button"
                    className={styles.remove}
                    aria-label={`Remove ${item.file.name}`}
                    onClick={() => setItems((prev) => prev.filter((i) => i.id !== item.id))}
                  >
                    ×
                  </button>
                )}
              </figure>
            ))}
          </div>
        )}

        {skipped && (
          <p className={styles.summary} role="status">
            {skipped}
          </p>
        )}

        {finished && doneCount > 0 && failedCount === 0 && (
          <p className={styles.success} role="status">
            {hiddenFromUploader
              ? `Added ${doneCount}, but NSFW is hidden for your account — turn on “Show NSFW posts” in Me → Sign-in & security to see it. Closing…`
              : `Added ${doneCount} to the board — closing…`}
          </p>
        )}
        {finished && failedCount > 0 && (
          <p className={styles.summary} role="status">
            {doneCount > 0
              ? `${doneCount} added · ${failedCount} not added.`
              : "Nothing added yet."}
          </p>
        )}

        <div className={styles.field}>
          <span>Rating</span>
          <ThemedSelect label="Content rating" value={rating} onChange={setRating}>
            <option value="sfw">Safe (SFW)</option>
            <option value="nsfw">NSFW (18+)</option>
          </ThemedSelect>
          {hiddenFromUploader && (
            <span className={styles.errorStatus}>
              Heads up: NSFW is hidden for your account, so this will not show for you until you
              turn on “Show NSFW posts” in Me → Sign-in &amp; security.
            </span>
          )}
        </div>
        <label className={styles.field}>
          Tags (optional, space separated)
          <input
            value={tags}
            onChange={(e) => setTags(e.target.value)}
            placeholder="anime girl white_hair"
          />
        </label>

        <div className={styles.footer}>
          <button type="button" onClick={closeNow} disabled={busy}>
            {finished && doneCount > 0 ? "Done" : "Cancel"}
          </button>
          <button
            type="button"
            className={styles.primary}
            disabled={queued.length === 0 || busy}
            onClick={uploadAll}
          >
            {busy
              ? "Uploading…"
              : failedCount > 0 && finished
                ? `Retry ${queued.length}`
                : `Add ${queued.length || ""} to the board`}
          </button>
        </div>
      </div>
    </div>
  );
}
