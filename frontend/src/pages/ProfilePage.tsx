import { Link, useNavigate } from "@tanstack/react-router";
import type { CSSProperties } from "react";
import { Pager } from "../components/Pager";
import { PostCard } from "../components/PostCard";
import { ShareProfileButton } from "../components/ShareProfileButton";
import { PROFILE_PAGE_SIZE } from "../domain/profileSearch";
import { useProfile } from "../hooks/useCollections";
import styles from "./Profile.module.css";
import { ROLE_LABELS } from "../domain/roles";
import {
  FONT_LABELS,
  THEME_LABELS,
  THEME_PAPERS,
  THEME_SWATCHES,
  normalizeFont,
  normalizeTheme,
} from "../domain/appearance";

export function ProfilePage({ username, page = 1 }: { username: string; page?: number }) {
  const navigate = useNavigate();
  const { data, isPending, error } = useProfile(username, page);
  const goToPage = (next: number) => {
    if (!Number.isInteger(next) || next < 1 || next === page) return;
    void navigate({
      to: "/u/$username",
      params: { username },
      search: next > 1 ? { page: next } : {},
    });
  };
  if (isPending) return <p>Loading…</p>;
  if (error || !data) return <p role="alert">That profile could not be found.</p>;
  const { user, public_lists: lists, contributions } = data;
  const uploadTotal = data.upload_total ?? 0;
  const editCount = data.edit_count ?? 0;
  const uploads = data.uploads ?? [];
  const theme = normalizeTheme(data.theme);
  const font = normalizeFont(data.font);
  return (
    <section
      className={`${styles.page} ${styles.themed}`}
      aria-labelledby="profile-title"
      data-theme={theme}
      data-font={font}
    >
      <div className={styles.header}>
        {user.avatar_url ? (
          <img className={styles.avatar} src={user.avatar_url} alt="" />
        ) : (
          <span className={styles.avatarFallback} aria-hidden="true">
            {(user.display_name || user.username).charAt(0).toUpperCase()}
          </span>
        )}
        <div className={styles.headerText}>
          <h1 id="profile-title">{user.display_name || user.username}</h1>
          <p className={styles.muted}>
            @{user.username} · {ROLE_LABELS[user.role]}
            {user.location ? ` · ${user.location}` : ""}
          </p>
          <p className={styles.appearance}>
            <span
              className={styles.themeChip}
              style={
                {
                  "--swatch": THEME_SWATCHES[theme],
                  "--paper": THEME_PAPERS[theme],
                } as CSSProperties
              }
              aria-hidden="true"
            />
            {THEME_LABELS[theme]} · {FONT_LABELS[font]}
          </p>
          <p className={styles.stats}>
            <strong>{uploadTotal}</strong> {uploadTotal === 1 ? "upload" : "uploads"} ·{" "}
            <strong>{editCount}</strong> {editCount === 1 ? "tag edit" : "tag edits"} ·{" "}
            <strong>{lists.length}</strong> {lists.length === 1 ? "public list" : "public lists"} ·
            joined {new Date(user.created_at).toLocaleDateString()}
          </p>
        </div>
        <div className={styles.share}>
          <ShareProfileButton username={user.username} />
        </div>
      </div>
      {user.bio && <p className={styles.bio}>{user.bio}</p>}
      {user.website && (
        <p>
          <a href={user.website} target="_blank" rel="noreferrer">
            {user.website}
          </a>
        </p>
      )}
      <div>
        <h2>Uploads</h2>
        {uploadTotal === 0 && <p className={styles.muted}>No pictures on the board yet.</p>}
        <div className={styles.uploads}>
          {uploads.map((post) => (
            <PostCard key={post.id} post={post} />
          ))}
        </div>
        {contributions > uploadTotal && (
          <p className={styles.muted}>Hidden pictures stay off this profile.</p>
        )}
        <Pager page={page} total={uploadTotal} pageSize={PROFILE_PAGE_SIZE} onPage={goToPage} />
      </div>
      <div>
        <h2>Public lists</h2>
        {lists.length === 0 && <p className={styles.muted}>No public lists yet.</p>}
        <div className={styles.grid}>
          {lists.map((list) => (
            <article key={list.id} className={styles.card}>
              <h3>
                <Link to="/lists/$id" params={{ id: list.id }}>
                  {list.title}
                </Link>
              </h3>
              {list.description && <p className={styles.muted}>{list.description}</p>}
              <p className={styles.muted}>{list.item_count} pictures</p>
            </article>
          ))}
        </div>
      </div>
    </section>
  );
}
