import { Link } from "@tanstack/react-router";
import { ShareProfileButton } from "../components/ShareProfileButton";
import { useProfile } from "../hooks/useKura";
import styles from "./Profile.module.css";

export function ProfilePage({ username }: { username: string }) {
  const { data, isPending, error } = useProfile(username);
  if (isPending) return <p>Loading…</p>;
  if (error || !data) return <p role="alert">That profile could not be found.</p>;
  const { user, public_lists: lists, contributions } = data;
  return (
    <section className={styles.page} aria-labelledby="profile-title">
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
            @{user.username}
            {user.location ? ` · ${user.location}` : ""}
          </p>
          <p className={styles.stats}>
            <strong>{contributions}</strong> contribution{contributions === 1 ? "" : "s"} ·{" "}
            <strong>{lists.length}</strong> public list{lists.length === 1 ? "" : "s"} · joined{" "}
            {new Date(user.created_at).toLocaleDateString()}
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
