import { Link, useNavigate } from "@tanstack/react-router";
import { useState } from "react";
import {
  IconCamera,
  IconHeart,
  IconList,
  IconShield,
  IconUpload,
  IconUser,
} from "../components/icons";
import { PostCard } from "../components/PostCard";
import { ThemedSelect } from "../components/ThemedSelect";
import { RecoveryKeyPanel } from "../components/RecoveryKeyPanel";
import { ShareProfileButton } from "../components/ShareProfileButton";
import { shortError, useToasts } from "../components/Toasts";
import {
  normalizeTagToken,
  passkeysSupported,
  tagTokens,
  useAccountActions,
  useAuthActions,
  useFavorites,
  useInvalidate,
  useListActions,
  useMe,
  useMyLists,
  useMyPosts,
} from "../hooks/useKura";
import { ROLE_LABELS } from "../domain/roles";
import styles from "./Simple.module.css";

const DISMISS_KEY = "kura-passkey-tip-dismissed";

export function MePage() {
  const { me, loading } = useMe();
  const navigate = useNavigate();
  const account = useAccountActions();
  const auth = useAuthActions();
  const listActions = useListActions();
  const invalidate = useInvalidate();
  const toasts = useToasts();
  // Remember the active tab across reloads.
  const [tab, setTab] = useState<"profile" | "contributions" | "favorites" | "lists" | "security">(
    () => {
      const saved = localStorage.getItem("kura-me-tab");
      return saved === "contributions" ||
        saved === "favorites" ||
        saved === "lists" ||
        saved === "security"
        ? saved
        : "profile";
    },
  );
  const changeTab = (name: typeof tab) => {
    setTab(name);
    localStorage.setItem("kura-me-tab", name);
  };
  const [listFilter, setListFilter] = useState("");
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [displayDraft, setDisplayDraft] = useState<string | null>(null);
  const [bioDraft, setBioDraft] = useState<string | null>(null);
  const [locationDraft, setLocationDraft] = useState<string | null>(null);
  const [websiteDraft, setWebsiteDraft] = useState<string | null>(null);
  const [shownKey, setShownKey] = useState("");
  const [includeDraft, setIncludeDraft] = useState<string[] | null>(null);
  const [excludeDraft, setExcludeDraft] = useState<string[] | null>(null);
  const [includeInput, setIncludeInput] = useState("");
  const [excludeInput, setExcludeInput] = useState("");
  const [showAgeGate, setShowAgeGate] = useState(false);
  const [tipDismissed, setTipDismissed] = useState(() => localStorage.getItem(DISMISS_KEY) === "1");
  const favorites = useFavorites(tab === "favorites");
  const myPosts = useMyPosts(tab === "contributions");
  const myLists = useMyLists(tab === "lists" || tab === "profile");

  if (loading) return <p>Loading…</p>;
  if (!me) {
    return (
      <section className={styles.page}>
        <h1>Your profile</h1>
        <p className={styles.muted}>Sign in to see your favorites, lists and profile.</p>
        <div className={styles.actions}>
          <Link to="/login">Sign in</Link>
          <Link to="/register">Create account</Link>
        </div>
      </section>
    );
  }
  const { user, security, preferences } = me;
  const showPasskeyTip = !tipDismissed && security.passkeys.length === 0 && passkeysSupported();
  const profileDirty =
    (displayDraft !== null && displayDraft !== user.display_name) ||
    (bioDraft !== null && bioDraft !== user.bio) ||
    (locationDraft !== null && locationDraft !== user.location) ||
    (websiteDraft !== null && websiteDraft !== user.website);
  const savedIncludeTags = preferences.default_include_tags ?? [];
  const savedExcludeTags = preferences.default_exclude_tags ?? [];
  const tagsDirty =
    (includeDraft !== null && JSON.stringify(includeDraft) !== JSON.stringify(savedIncludeTags)) ||
    (excludeDraft !== null && JSON.stringify(excludeDraft) !== JSON.stringify(savedExcludeTags));

  const run = (action: Promise<unknown>, ok: string) => {
    setError("");
    setNotice("");
    void action
      .then(() => {
        toasts.push(ok, "success");
        invalidate();
      })
      .catch((e: unknown) => toasts.push(shortError(e), "error"));
  };

  return (
    <section className={styles.page} aria-labelledby="me-title" style={{ maxWidth: "80rem" }}>
      <div style={{ display: "flex", gap: "1rem", alignItems: "center", flexWrap: "wrap" }}>
        <label
          title={
            user.avatar_url
              ? "Click to change your profile picture"
              : "Click to upload a profile picture"
          }
          style={{ position: "relative", cursor: "pointer", flex: "none" }}
        >
          {user.avatar_url ? (
            <img
              src={user.avatar_url}
              alt=""
              style={{
                width: "4rem",
                height: "4rem",
                borderRadius: "999px",
                objectFit: "cover",
                border: "1px solid var(--border)",
                display: "block",
              }}
            />
          ) : (
            <span
              aria-hidden="true"
              style={{
                width: "4rem",
                height: "4rem",
                borderRadius: "999px",
                display: "grid",
                placeItems: "center",
                background: "var(--surface-soft)",
                border: "1px solid var(--border)",
                fontSize: "1.5rem",
                color: "var(--accent)",
              }}
            >
              {(user.display_name || user.username).charAt(0).toUpperCase()}
            </span>
          )}
          <span
            aria-hidden="true"
            style={{
              position: "absolute",
              right: "-2px",
              bottom: "-2px",
              display: "grid",
              placeItems: "center",
              width: "1.45rem",
              height: "1.45rem",
              borderRadius: "999px",
              background: "var(--surface)",
              border: "1px solid var(--border-strong)",
              color: "var(--text)",
            }}
          >
            <IconCamera />
          </span>
          <input
            type="file"
            accept="image/*"
            aria-label="Upload a profile picture"
            style={{ display: "none" }}
            onChange={(e) => {
              const file = e.target.files?.[0];
              if (file) run(account.setAvatar(file), "Profile picture saved.");
              e.target.value = "";
            }}
          />
        </label>
        <div>
          <h1 id="me-title">{user.display_name || user.username}</h1>
          <p className={styles.muted}>
            @{user.username} · {ROLE_LABELS[user.role]} ·{" "}
            <Link to="/u/$username" params={{ username: user.username }}>
              View public profile
            </Link>
            {user.avatar_url && (
              <>
                {" · "}
                <button
                  type="button"
                  onClick={() => run(account.removeAvatar(), "Profile picture removed.")}
                  style={{
                    border: "none",
                    background: "none",
                    color: "var(--muted)",
                    cursor: "pointer",
                    padding: 0,
                    font: "inherit",
                    textDecoration: "underline",
                  }}
                >
                  Remove picture
                </button>
              </>
            )}
          </p>
        </div>
        <ShareProfileButton username={user.username} />
      </div>

      {showPasskeyTip && (
        <div className={styles.banner}>
          <span>
            Tip: add a passkey for quicker, safer sign-ins — no password to type.{" "}
            <button
              type="button"
              onClick={() => changeTab("security")}
              style={{ textDecoration: "underline" }}
            >
              Set one up
            </button>
          </span>
          <button
            type="button"
            aria-label="Dismiss passkey tip"
            onClick={() => {
              localStorage.setItem(DISMISS_KEY, "1");
              setTipDismissed(true);
            }}
          >
            ✕
          </button>
        </div>
      )}

      <div className={styles.tabs} role="tablist" aria-label="Your stuff">
        {(
          [
            { name: "profile", label: "Profile", icon: <IconUser /> },
            { name: "contributions", label: "Contributions", icon: <IconUpload /> },
            { name: "favorites", label: "Favorites", icon: <IconHeart /> },
            { name: "lists", label: "My lists", icon: <IconList /> },
            { name: "security", label: "Sign-in & security", icon: <IconShield /> },
          ] as const
        ).map(({ name, label, icon }) => (
          <button
            key={name}
            type="button"
            role="tab"
            aria-selected={tab === name}
            className={tab === name ? styles.activeTab : ""}
            onClick={() => changeTab(name)}
            style={{ display: "inline-flex", alignItems: "center", gap: "0.4rem" }}
          >
            {icon}
            <span className={styles.tabLabel}>{label}</span>
          </button>
        ))}
      </div>
      {error && (
        <p role="alert" className={styles.error}>
          {error}
        </p>
      )}
      {notice && <p className={styles.hint}>{notice}</p>}

      {tab === "profile" && (
        <div className={styles.profileGrid}>
          <form
            className={styles.form}
            style={{ maxWidth: "32rem" }}
            onSubmit={(event) => {
              event.preventDefault();
              run(
                account.updateProfile({
                  display_name: displayDraft ?? user.display_name,
                  bio: bioDraft ?? user.bio,
                  location: locationDraft ?? user.location,
                  website: websiteDraft ?? user.website,
                }),
                "Profile saved.",
              );
              setDisplayDraft(null);
              setBioDraft(null);
              setLocationDraft(null);
              setWebsiteDraft(null);
            }}
          >
            <label>
              Display name{" "}
              <span className={styles.count}>{(displayDraft ?? user.display_name).length}/60</span>
              <input
                value={displayDraft ?? user.display_name}
                onChange={(e) => setDisplayDraft(e.target.value)}
                required
                maxLength={60}
              />
            </label>
            <label>
              Bio{" "}
              <span className={styles.count}>{(bioDraft ?? user.bio).length}/500 characters</span>
              <textarea
                value={bioDraft ?? user.bio}
                onChange={(e) => setBioDraft(e.target.value)}
                maxLength={500}
                rows={4}
                placeholder="Tell people a little about you and what you collect…"
              />
            </label>
            <label>
              Location{" "}
              <span className={styles.count}>{(locationDraft ?? user.location).length}/80</span>
              <input
                value={locationDraft ?? user.location}
                onChange={(e) => setLocationDraft(e.target.value)}
                maxLength={80}
              />
            </label>
            <label>
              Website{" "}
              <span className={styles.count}>{(websiteDraft ?? user.website).length}/200</span>
              <input
                value={websiteDraft ?? user.website}
                onChange={(e) => setWebsiteDraft(e.target.value)}
                maxLength={200}
                placeholder="https://…"
              />
            </label>
            <button type="submit" className={profileDirty ? styles.saveDirty : undefined}>
              Save profile
            </button>
          </form>

          <div className={styles.card}>
            <h2>Default board tags</h2>
            <p className={styles.muted}>
              Applied every time you open the board. Always show posts with your tags; never show
              posts with your excluded tags. On the board you can still change them for the session,
              or type <code>-tag</code> to exclude.
            </p>
            {(
              [
                {
                  label: "Always show",
                  placeholder: "Add a tag…",
                  tags: includeDraft ?? preferences.default_include_tags ?? [],
                  input: includeInput,
                  setInput: setIncludeInput,
                  add: (raw: string) => {
                    const tokens = tagTokens(raw);
                    if (tokens.length === 0) return;
                    const current = includeDraft ?? preferences.default_include_tags ?? [];
                    setIncludeDraft([...new Set([...current, ...tokens])]);
                    const excluded = excludeDraft ?? preferences.default_exclude_tags ?? [];
                    setExcludeDraft(excluded.filter((t) => !tokens.includes(t)));
                    setIncludeInput("");
                  },
                  remove: (tag: string) => {
                    const current = includeDraft ?? preferences.default_include_tags ?? [];
                    setIncludeDraft(current.filter((t) => t !== tag));
                  },
                },
                {
                  label: "Never show",
                  placeholder: "Add an excluded tag…",
                  tags: excludeDraft ?? preferences.default_exclude_tags ?? [],
                  input: excludeInput,
                  setInput: setExcludeInput,
                  add: (raw: string) => {
                    const tokens = tagTokens(raw.replace(/^-+/, ""));
                    if (tokens.length === 0) return;
                    const current = excludeDraft ?? preferences.default_exclude_tags ?? [];
                    setExcludeDraft([...new Set([...current, ...tokens])]);
                    const included = includeDraft ?? preferences.default_include_tags ?? [];
                    setIncludeDraft(included.filter((t) => !tokens.includes(t)));
                    setExcludeInput("");
                  },
                  remove: (tag: string) => {
                    const current = excludeDraft ?? preferences.default_exclude_tags ?? [];
                    setExcludeDraft(current.filter((t) => t !== tag));
                  },
                },
              ] as const
            ).map((field) => (
              <div key={field.label} style={{ marginTop: "0.9rem" }}>
                <span style={{ fontWeight: 600, fontSize: "0.9rem" }}>{field.label}</span>
                <div
                  style={{ display: "flex", gap: "0.35rem", flexWrap: "wrap", margin: "0.45rem 0" }}
                >
                  {field.tags.map((tag) => (
                    <span
                      key={tag}
                      style={{
                        display: "inline-flex",
                        gap: "0.3rem",
                        alignItems: "center",
                        padding: "0.2rem 0.55rem",
                        border: "1px solid var(--border-strong)",
                        borderRadius: "999px",
                        background: "var(--surface-soft)",
                        fontSize: "0.8rem",
                      }}
                    >
                      {field.label === "Never show" ? `−${tag}` : tag}
                      <button
                        type="button"
                        aria-label={`Remove ${tag}`}
                        onClick={() => field.remove(tag)}
                        style={{
                          border: "none",
                          background: "none",
                          color: "var(--muted)",
                          cursor: "pointer",
                          padding: 0,
                        }}
                      >
                        ×
                      </button>
                    </span>
                  ))}
                  {field.tags.length === 0 && <span className={styles.muted}>None</span>}
                </div>
                <form
                  className={styles.inlineForm}
                  onSubmit={(e) => {
                    e.preventDefault();
                    field.add(field.input);
                  }}
                >
                  <input
                    value={field.input}
                    onChange={(e) => field.setInput(e.target.value)}
                    placeholder={field.placeholder}
                    aria-label={`${field.label} tag`}
                  />
                  <button type="submit" disabled={!normalizeTagToken(field.input)}>
                    Add
                  </button>
                </form>
              </div>
            ))}
            <div className={styles.actions} style={{ marginTop: "1rem" }}>
              <button
                type="button"
                className={tagsDirty ? styles.saveDirty : undefined}
                onClick={() => {
                  run(
                    account.setTagPreferences(
                      includeDraft ?? preferences.default_include_tags ?? [],
                      excludeDraft ?? preferences.default_exclude_tags ?? [],
                    ),
                    "Default board tags saved.",
                  );
                  setIncludeDraft(null);
                  setExcludeDraft(null);
                }}
              >
                Save board tags
              </button>
              <button
                type="button"
                onClick={() => {
                  setIncludeDraft([]);
                  setExcludeDraft([]);
                  setIncludeInput("");
                  setExcludeInput("");
                }}
              >
                Clear both
              </button>
            </div>
          </div>
        </div>
      )}

      {tab === "contributions" && (
        <div>
          <p className={styles.muted}>
            Everything you have added to the board, newest first.
            {(myPosts.data ?? []).length > 0
              ? ` ${myPosts.data?.length} picture${(myPosts.data?.length ?? 0) === 1 ? "" : "s"} here — hidden NSFW uploads only appear while “Show NSFW posts” is on.`
              : ""}
          </p>
          {myPosts.isPending && <p className={styles.muted}>Loading…</p>}
          {(myPosts.data ?? []).length === 0 && !myPosts.isPending && (
            <p className={styles.muted}>
              Nothing yet. Use “Upload” on the board to add your first picture — it will show up
              here and at the top of the board.
            </p>
          )}
          <div className={styles.grid}>
            {(myPosts.data ?? []).map((post) => (
              <PostCard key={post.id} post={post} />
            ))}
          </div>
        </div>
      )}

      {tab === "favorites" && (
        <div>
          <p className={styles.muted}>Private — only you can see your favorites.</p>
          {(favorites.data ?? []).length === 0 && (
            <p className={styles.muted}>No favorites yet. Tap “♡ Save” on any picture.</p>
          )}
          <div className={styles.grid}>
            {(favorites.data ?? []).map((post) => (
              <PostCard key={post.id} post={post} />
            ))}
          </div>
        </div>
      )}

      {tab === "lists" && (
        <div style={{ display: "grid", gap: "1rem" }}>
          <form
            className={styles.inlineForm}
            onSubmit={(event) => {
              event.preventDefault();
              const form = new FormData(event.currentTarget);
              run(
                listActions.createList({
                  title: String(form.get("title") ?? ""),
                  description: String(form.get("description") ?? ""),
                  visibility: String(form.get("visibility") ?? "private"),
                }),
                "List created.",
              );
              event.currentTarget.reset();
            }}
          >
            <input
              name="title"
              required
              maxLength={80}
              placeholder="New list title"
              aria-label="New list title"
            />
            <input
              name="description"
              maxLength={300}
              placeholder="Description (optional)"
              aria-label="List description"
            />
            <ThemedSelect name="visibility" label="Visibility" defaultValue="private">
              <option value="private">Private</option>
              <option value="public">Public</option>
            </ThemedSelect>
            <button type="submit">Create list</button>
          </form>
          {(myLists.data ?? []).length > 5 && (
            <input
              value={listFilter}
              onChange={(e) => setListFilter(e.target.value)}
              placeholder={`Filter ${myLists.data?.length ?? ""} lists…`}
              aria-label="Filter your lists"
              style={{ maxWidth: "20rem" }}
            />
          )}
          <div>
            {(myLists.data ?? [])
              .filter((list) => list.title.toLowerCase().includes(listFilter.trim().toLowerCase()))
              .map((list) => (
                <div key={list.id} className={styles.listRow}>
                  <span>
                    <Link to="/lists/$id" params={{ id: list.id }}>
                      {list.title}
                    </Link>{" "}
                    <span className={styles.muted}>
                      {list.visibility} · {list.item_count} pictures
                    </span>
                  </span>
                </div>
              ))}
            {(myLists.data ?? []).length === 0 && <p className={styles.muted}>No lists yet.</p>}
          </div>
        </div>
      )}

      {tab === "security" && (
        <div className={styles.securityGrid}>
          <div className={styles.card}>
            <h2>Content</h2>
            <label
              style={{
                display: "flex",
                gap: "0.55rem",
                alignItems: "flex-start",
                cursor: "pointer",
              }}
            >
              <input
                type="checkbox"
                checked={preferences.show_nsfw}
                onChange={(e) => {
                  // Turning on asks first (pop-up below); only "Yes, I'm
                  // over 18" switches it on. Turning off applies at once.
                  if (e.target.checked) setShowAgeGate(true);
                  else run(account.setPreferences(false, false), "NSFW posts hidden.");
                }}
                style={{ marginTop: "0.2rem" }}
              />
              <span>
                Show NSFW posts
                <span className={styles.muted} style={{ display: "block" }}>
                  Off by default. NSFW pictures stay hidden everywhere — board, search, lists —
                  until you turn this on and confirm your age.
                </span>
              </span>
            </label>
          </div>

          {showAgeGate && (
            <div
              className={styles.modalBackdrop}
              role="presentation"
              onMouseDown={(e) => {
                if (e.target === e.currentTarget) setShowAgeGate(false);
              }}
            >
              <div
                className={styles.modal}
                role="dialog"
                aria-modal="true"
                aria-labelledby="age-gate-title"
              >
                <h2 id="age-gate-title">Are you over 18?</h2>
                <p>
                  NSFW posts contain adult content. They will start showing on the board, in search
                  and in lists for this account. You can turn this off again at any time.
                </p>
                <div className={styles.actions}>
                  <button type="button" onClick={() => setShowAgeGate(false)}>
                    No, keep it off
                  </button>
                  <button
                    type="button"
                    className={styles.modalPrimary}
                    onClick={() => {
                      setShowAgeGate(false);
                      run(account.setPreferences(true, true), "NSFW posts on — age confirmed.");
                    }}
                  >
                    Yes, I&apos;m over 18
                  </button>
                </div>
              </div>
            </div>
          )}

          <div className={styles.card}>
            <h2>Passkeys {security.passkeys.length > 0 ? "" : "(recommended)"}</h2>
            {security.passkeys.length === 0 && (
              <p className={styles.muted}>
                Sign in with your face, fingerprint or device PIN — nothing to remember.
              </p>
            )}
            {security.passkeys.map((key) => (
              <div key={key.id} className={styles.listRow}>
                <span>
                  {key.name}{" "}
                  <span className={styles.muted}>
                    added {new Date(key.created_at).toLocaleDateString()}
                  </span>
                </span>
                <button
                  type="button"
                  onClick={() => run(account.removePasskey(key.id), "Passkey removed.")}
                >
                  Remove
                </button>
              </div>
            ))}
            <div className={styles.actions}>
              <button
                type="button"
                disabled={!passkeysSupported()}
                onClick={() => run(account.addPasskey(), "Passkey added.")}
              >
                Add a passkey
              </button>
            </div>
          </div>

          <div className={styles.card}>
            <h2>Password</h2>
            <p className={styles.muted}>
              {security.has_password
                ? "You have a password set. Passkeys and passwords work interchangeably."
                : "No password set — you sign in with a passkey."}
            </p>
            <form
              className={styles.inlineForm}
              onSubmit={(event) => {
                event.preventDefault();
                run(account.setPassword(newPassword), "Password saved.");
                setNewPassword("");
              }}
            >
              <input
                type="password"
                value={newPassword}
                onChange={(e) => setNewPassword(e.target.value)}
                minLength={15}
                required
                placeholder="New password (15+ characters — a short sentence works great)"
                aria-label="New password"
                autoComplete="new-password"
              />
              <button type="submit">
                {security.has_password ? "Change password" : "Set password"}
              </button>
            </form>
            {security.has_password && (
              <div className={styles.actions}>
                <button
                  type="button"
                  onClick={() => run(account.removePassword(), "Password removed.")}
                >
                  Remove password
                </button>
              </div>
            )}
            <p className={styles.muted}>
              You always keep at least one way to sign in — Kura stops you removing your last one.
            </p>
          </div>

          {shownKey ? (
            <RecoveryKeyPanel
              recoveryKey={shownKey}
              onDone={() => setShownKey("")}
              doneLabel="Done"
            />
          ) : (
            <div className={styles.card}>
              <h2>Recovery key</h2>
              <>
                <p className={styles.muted}>
                  {security.has_recovery_key
                    ? `You have a recovery key (created ${
                        security.recovery_key_created_at
                          ? new Date(security.recovery_key_created_at).toLocaleDateString()
                          : ""
                      }). It is the only way back in if you lose your password and passkeys.`
                    : "You don't have a recovery key yet. Without one, losing your password and passkeys means losing the account."}
                </p>
                <div className={styles.actions}>
                  <button
                    type="button"
                    onClick={() => {
                      setError("");
                      void account
                        .regenerateRecoveryKey()
                        .then((me) => {
                          if (me.recovery_key) setShownKey(me.recovery_key);
                          invalidate();
                        })
                        .catch((e: unknown) =>
                          setError(e instanceof Error ? e.message : "Something went wrong."),
                        );
                    }}
                  >
                    {security.has_recovery_key
                      ? "Make a new key (old one stops working)"
                      : "Create a recovery key"}
                  </button>
                </div>
              </>
            </div>
          )}

          <div className={styles.actions} style={{ gridColumn: "1 / -1" }}>
            <button
              type="button"
              onClick={() => {
                void auth.logout().then(() => {
                  invalidate();
                  void navigate({ to: "/" });
                });
              }}
            >
              Sign out
            </button>
          </div>
        </div>
      )}
    </section>
  );
}
