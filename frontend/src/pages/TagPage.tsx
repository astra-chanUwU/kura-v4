import { Link } from "@tanstack/react-router";
import { useState, type FormEvent } from "react";
import { PostCard } from "../components/PostCard";
import { ConfirmDialog } from "../components/ConfirmDialog";
import { shortError, useToasts } from "../components/Toasts";
import { atLeast } from "../domain/roles";
import { useMe } from "../hooks/useKura";
import { useSuggestTag } from "../hooks/useSuggestions";
import { useTagPage, useTagPageActions } from "../hooks/useTagPage";
import type { SuggestionKind } from "../types/suggestion";
import { TAG_CATEGORIES, TAG_CATEGORY_LABELS, type TagCategory } from "../types/tagPage";
import styles from "./TagPage.module.css";

function isCategory(value: string): value is TagCategory {
  return (TAG_CATEGORIES as string[]).includes(value);
}

export function TagPage({ name }: { name: string }) {
  const page = useTagPage(name);
  const { user } = useMe();
  const actions = useTagPageActions(page.data?.alias_of || name);
  const toasts = useToasts();
  const [category, setCategory] = useState<string | null>(null);
  const [description, setDescription] = useState<string | null>(null);
  const [alias, setAlias] = useState("");
  const [consequent, setConsequent] = useState("");
  const [pendingRemove, setPendingRemove] = useState("");
  const moderator = !!user && atLeast(user.role, "moderator");
  const signedIn = !!user;

  if (page.isPending) {
    return (
      <section className={styles.page}>
        <p className={styles.muted}>Loading tag…</p>
      </section>
    );
  }
  if (page.error) {
    return (
      <section className={styles.page}>
        <h1>{name}</h1>
        <p className={styles.error} role="alert">
          {page.missing ? "That tag is not on the board yet." : shortError(page.error)}
        </p>
        <p>
          <Link to="/">Back to the board</Link>
        </p>
      </section>
    );
  }

  const data = page.data;
  const canonical = data.alias_of || data.tag;
  const categoryValue = category ?? data.category;
  const descriptionValue = description ?? data.description;
  const dirty = categoryValue !== data.category || descriptionValue !== data.description;
  const paragraphs = data.description.trim();
  const otherAliases = data.aliases.filter((item) => item !== data.tag);

  const save = (event: FormEvent) => {
    event.preventDefault();
    if (!isCategory(categoryValue)) return;
    actions.update.mutate(
      { category: categoryValue, description: descriptionValue },
      {
        onSuccess: () => {
          setCategory(null);
          setDescription(null);
          toasts.push("Tag page saved.", "success");
        },
        onError: (error: unknown) => toasts.push(shortError(error), "error"),
      },
    );
  };
  const addAlias = (event: FormEvent) => {
    event.preventDefault();
    actions.addAlias.mutate(alias, {
      onSuccess: () => {
        setAlias("");
        toasts.push("Alias added. Searching either name finds the same pictures.", "success");
      },
      onError: (error: unknown) => toasts.push(shortError(error), "error"),
    });
  };

  return (
    <section className={styles.page}>
      <div className={styles.heading}>
        <h1>{data.tag}</h1>
        <span className={styles.category}>
          {TAG_CATEGORY_LABELS[data.category] ?? data.category}
        </span>
      </div>
      {data.alias_of && (
        <p className={styles.notice}>
          <strong>{data.tag}</strong> is an alias of{" "}
          <Link to="/tags/$tag" params={{ tag: data.alias_of }}>
            {data.alias_of}
          </Link>
          . Searching either name finds the same pictures.
        </p>
      )}
      <p className={styles.muted}>
        {data.post_count} post{data.post_count === 1 ? "" : "s"}
        {data.updated_by_username ? ` · described by @${data.updated_by_username}` : ""}
      </p>
      {paragraphs ? (
        <p className={styles.description}>{paragraphs}</p>
      ) : (
        <p className={styles.muted}>No description yet.</p>
      )}
      <p>
        <Link to="/" search={{ tags: canonical }}>
          Search this tag
        </Link>
      </p>

      <section className={styles.section} aria-labelledby="aliases-title">
        <h2 id="aliases-title">Also known as</h2>
        <p className={styles.muted}>
          An alias is another name for this tag. New tags use the canonical name, and a search for
          either name finds the same pictures.
        </p>
        {otherAliases.length === 0 ? (
          data.alias_of ? null : (
            <p className={styles.muted}>No aliases yet.</p>
          )
        ) : (
          <div className={styles.aliases}>
            {otherAliases.map((item) => (
              <span key={item} className={styles.alias}>
                <Link to="/tags/$tag" params={{ tag: item }}>
                  {item}
                </Link>
                {moderator && (
                  <button
                    type="button"
                    aria-label={`Remove alias ${item}`}
                    onClick={() => setPendingRemove(item)}
                  >
                    ×
                  </button>
                )}
              </span>
            ))}
          </div>
        )}
        {moderator && (
          <form className={styles.row} onSubmit={addAlias}>
            <input
              value={alias}
              onChange={(event) => setAlias(event.target.value)}
              placeholder="Alternate name, such as grey_hair"
              aria-label="New alias"
              maxLength={40}
            />
            <button type="submit" disabled={!alias.trim() || actions.addAlias.isPending}>
              Add alias
            </button>
          </form>
        )}
      </section>

      <section className={styles.section} aria-labelledby="implications-title">
        <h2 id="implications-title">Implications</h2>
        <p className={styles.muted}>
          A post with this tag also receives each implied tag. Removing a rule leaves tags already
          added.
        </p>
        {(data.implied_by ?? []).length > 0 && (
          <p className={styles.muted}>
            Also added by{" "}
            {data.implied_by.map((item, index) => (
              <span key={item}>
                {index > 0 ? ", " : ""}
                <Link to="/tags/$tag" params={{ tag: item }}>
                  {item}
                </Link>
              </span>
            ))}
            .
          </p>
        )}
        {(data.implies ?? []).length === 0 ? (
          <p className={styles.muted}>This tag implies nothing else.</p>
        ) : (
          <div className={styles.aliases}>
            {data.implies.map((item) => (
              <span key={item} className={styles.alias}>
                <Link to="/tags/$tag" params={{ tag: item }}>
                  {item}
                </Link>
                {moderator && (
                  <button
                    type="button"
                    aria-label={`Remove implication ${item}`}
                    onClick={() =>
                      actions.removeImplication.mutate(item, {
                        onError: (error) => toasts.push(shortError(error), "error"),
                      })
                    }
                  >
                    ×
                  </button>
                )}
              </span>
            ))}
          </div>
        )}
        {moderator && (
          <form
            className={styles.row}
            onSubmit={(event: FormEvent) => {
              event.preventDefault();
              const next = consequent.trim();
              if (!next) return;
              actions.addImplication.mutate(next, {
                onSuccess: () => setConsequent(""),
                onError: (error) => toasts.push(shortError(error), "error"),
              });
            }}
          >
            <input
              value={consequent}
              onChange={(event) => setConsequent(event.target.value)}
              placeholder="Tag this one implies, such as pleated_skirt"
              aria-label="Implied tag"
              maxLength={40}
            />
            <button type="submit" disabled={!consequent.trim() || actions.addImplication.isPending}>
              Add implication
            </button>
          </form>
        )}
      </section>

      {signedIn && <SuggestForm tag={canonical} />}

      {signedIn && (
        <form className={styles.form} onSubmit={save}>
          <label>
            Category
            <select
              value={categoryValue}
              aria-label="Category"
              onChange={(event) => setCategory(event.target.value)}
            >
              {TAG_CATEGORIES.map((item) => (
                <option key={item} value={item}>
                  {TAG_CATEGORY_LABELS[item]}
                </option>
              ))}
            </select>
          </label>
          <label>
            Description
            <textarea
              value={descriptionValue}
              aria-label="Description"
              rows={5}
              maxLength={2000}
              onChange={(event) => setDescription(event.target.value)}
            />
          </label>
          <div className={styles.row}>
            <button type="submit" disabled={!dirty || actions.update.isPending}>
              {actions.update.isPending ? "Saving…" : "Save description"}
            </button>
          </div>
        </form>
      )}

      {data.posts.length > 0 && (
        <section className={styles.section} aria-labelledby="tag-posts-title">
          <h2 id="tag-posts-title">Recent posts</h2>
          <div className={styles.grid}>
            {data.posts.map((post) => (
              <PostCard key={post.id} post={post} search={{ tags: canonical }} />
            ))}
          </div>
        </section>
      )}

      <ConfirmDialog
        open={pendingRemove !== ""}
        title={`Remove alias ${pendingRemove}?`}
        message="The name stops pointing at this tag. Pictures already moved to the canonical tag stay there."
        confirmLabel="Remove alias"
        destructive
        onCancel={() => setPendingRemove("")}
        onConfirm={() => {
          const removing = pendingRemove;
          setPendingRemove("");
          actions.removeAlias.mutate(removing, {
            onSuccess: () => toasts.push("Alias removed.", "success"),
            onError: (error: unknown) => toasts.push(shortError(error), "error"),
          });
        }}
      />
    </section>
  );
}

function SuggestForm({ tag }: { tag: string }) {
  const [kind, setKind] = useState<SuggestionKind>("alias");
  const [other, setOther] = useState("");
  const [reason, setReason] = useState("");
  const suggest = useSuggestTag(tag);
  const toasts = useToasts();
  const submit = (event: FormEvent) => {
    event.preventDefault();
    const next = other.trim();
    if (!next) return;
    suggest.mutate(
      { kind, other: next, reason: reason.trim() },
      {
        onSuccess: () => {
          setOther("");
          setReason("");
          toasts.push("Suggestion sent. A moderator can accept it.", "success");
        },
        onError: (error: unknown) => toasts.push(shortError(error), "error"),
      },
    );
  };
  return (
    <form className={styles.form} onSubmit={submit}>
      <h2>Suggest a change</h2>
      <p className={styles.muted}>
        {kind === "alias"
          ? `Another name for ${tag}. A moderator accepts it before it applies.`
          : `${tag} would also add the other tag. A moderator accepts it before it applies.`}
      </p>
      <div className={styles.row}>
        <select
          aria-label="Suggestion kind"
          value={kind}
          onChange={(event) => setKind(event.target.value === "implication" ? "implication" : "alias")}
        >
          <option value="alias">Alias</option>
          <option value="implication">Implication</option>
        </select>
        <input
          value={other}
          onChange={(event) => setOther(event.target.value)}
          placeholder={kind === "alias" ? "Alternate name" : "Tag this one implies"}
          aria-label={kind === "alias" ? "Alternate name" : "Implied tag"}
          maxLength={40}
        />
        <input
          value={reason}
          onChange={(event) => setReason(event.target.value)}
          placeholder="Why (optional)"
          aria-label="Reason"
          maxLength={300}
        />
        <button type="submit" disabled={!other.trim() || suggest.isPending}>
          Suggest
        </button>
      </div>
    </form>
  );
}
