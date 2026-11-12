import { Link, Navigate } from "@tanstack/react-router";
import { useRef, useState } from "react";
import { ConfirmDialog } from "../components/ConfirmDialog";
import { IconCrown, IconKey, IconShield, IconUser } from "../components/icons";
import { ThemedSelect } from "../components/ThemedSelect";
import { shortError, useToasts } from "../components/Toasts";
import { assignableRoles, atLeast, isRole, ROLE_LABELS, ROLES, type Role } from "../domain/roles";
import {
  useAdminUsers,
  useDebouncedValue,
  useLoadMore,
  useMe,
  useStaffActions,
} from "../hooks/useKura";
import type { User } from "../types/api";
import { StaffSection } from "./StaffLayout";
import styles from "./AdminPage.module.css";
import simple from "./Simple.module.css";

type Draft = { name: string; saved: Role; next: Role };
type StagedRole = { id: string; name: string; next: Role };
type PeopleSort = "name" | "newest" | "oldest";

const ROLE_ICON: Record<Role, React.ReactNode> = {
  member: <IconUser />,
  moderator: <IconShield />,
  admin: <IconKey />,
  super_admin: <IconCrown />,
};

/** One role at a time. Picking another only stages it; the page Save applies it. */
function RolePicker({
  username,
  saved,
  selected,
  choices,
  onChange,
}: {
  username: string;
  saved: Role;
  selected: Role;
  choices: Role[];
  onChange: (next: Role) => void;
}) {
  const buttons = useRef<Array<HTMLButtonElement | null>>([]);
  const dirty = selected !== saved;

  function pick(next: Role) {
    onChange(next);
  }

  function onKeyDown(event: React.KeyboardEvent<HTMLDivElement>) {
    const dir =
      event.key === "ArrowRight" || event.key === "ArrowDown"
        ? 1
        : event.key === "ArrowLeft" || event.key === "ArrowUp"
          ? -1
          : 0;
    if (dir === 0) return;
    event.preventDefault();
    const index = Math.max(0, choices.indexOf(selected));
    const nextIndex = (index + dir + choices.length) % choices.length;
    const nextRole = choices[nextIndex];
    if (!nextRole) return;
    pick(nextRole);
    buttons.current[nextIndex]?.focus();
  }

  return (
    <div className={styles.roleControl}>
      <div
        className={styles.roleGroup}
        role="radiogroup"
        aria-label={`Role for @${username}`}
        tabIndex={-1}
        onKeyDown={onKeyDown}
      >
        {choices.map((role, index) => (
          <button
            key={role}
            ref={(node) => {
              buttons.current[index] = node;
            }}
            type="button"
            role="radio"
            className={styles.roleOption}
            aria-checked={selected === role}
            aria-label={ROLE_LABELS[role]}
            title={ROLE_LABELS[role]}
            tabIndex={selected === role ? 0 : -1}
            onClick={() => pick(role)}
          >
            {ROLE_ICON[role]}
          </button>
        ))}
      </div>
      {dirty && (
        <span className={styles.changedDot} role="img" aria-label="Role change not saved yet" />
      )}
    </div>
  );
}

function matchLabel(total: number): string {
  return total === 1 ? "1 match" : `${total} matches`;
}

// People & roles, the Staff section's landing tab. Moderators have no
// people tools, so they land on the hidden-posts queue instead.
export function AdminPage() {
  const { user } = useMe();
  const toasts = useToasts();
  const staff = useStaffActions();
  const isAdmin = !!user && atLeast(user.role, "admin");
  const [q, setQ] = useState("");
  const [role, setRole] = useState<"" | Role>("");
  const [sort, setSort] = useState<PeopleSort>("name");
  const [drafts, setDrafts] = useState<Record<string, Draft>>({});
  const [confirming, setConfirming] = useState<StagedRole[] | null>(null);
  const debouncedQ = useDebouncedValue(q, 300);
  const submittedQ = debouncedQ.trim();
  const people = useAdminUsers(isAdmin, { q: submittedQ, role, sort });
  const moreRef = useLoadMore(people.hasMore && !people.loadingMore, people.loadMore);

  if (!user) return null;
  if (!isAdmin) return <Navigate to="/admin/hidden" replace />;

  const everyone = people.member + people.moderator + people.admin + people.superAdmin;

  const selectedFor = (account: User): Role => {
    const draft = drafts[account.id];
    if (draft && draft.next !== account.role) return draft.next;
    return account.role;
  };

  const staged: StagedRole[] = Object.entries(drafts).map(([id, draft]) => ({
    id,
    name: draft.name,
    next: draft.next,
  }));

  const stage = (account: User, next: Role) => {
    setDrafts((prev) => {
      const copy = { ...prev };
      if (next === account.role) delete copy[account.id];
      else copy[account.id] = { name: account.username, saved: account.role, next };
      return copy;
    });
  };

  const applyStaged = (changes: StagedRole[]) => {
    for (const change of changes) {
      void staff
        .setUserRole(change.id, change.next)
        .then(() => {
          setDrafts((prev) => {
            if (!(change.id in prev)) return prev;
            const copy = { ...prev };
            delete copy[change.id];
            return copy;
          });
          toasts.push(`@${change.name} is now ${ROLE_LABELS[change.next]}.`, "success");
        })
        .catch((e: unknown) => toasts.push(shortError(e), "error"));
    }
  };

  const saveRoles = () => {
    if (staged.length === 0) return;
    if (staged.some((change) => change.next === "admin" || change.next === "super_admin")) {
      setConfirming(staged);
      return;
    }
    applyStaged(staged);
  };

  const onlyChange = confirming?.length === 1 ? confirming[0] : undefined;
  const confirmCopy = onlyChange
    ? {
        title: `Make @${onlyChange.name} a ${ROLE_LABELS[onlyChange.next]}?`,
        message: "They will gain staff powers.",
        confirmLabel: `Make ${ROLE_LABELS[onlyChange.next]}`,
      }
    : {
        title: "Save these roles?",
        message: (
          <ul className={styles.roleChanges} aria-label="Role changes">
            {(confirming ?? []).map((change) => (
              <li key={change.id}>
                <span className={`${styles.roleMark} ${styles[change.next]}`} aria-hidden="true" />
                {`@${change.name} → `}
                <span className={styles[change.next]}>{ROLE_LABELS[change.next]}</span>
              </li>
            ))}
          </ul>
        ),
        confirmLabel: "Save",
      };

  const filtered = submittedQ !== "" || role !== "";

  return (
    <StaffSection
      id="people-title"
      title="People & roles"
      intro={
        <>
          Super Admins manage anyone. Admins only move people between Member and Moderator. The last
          Super Admin can never be demoted. Visitor means anonymous: SFW reading only, no account.
        </>
      }
    >
      <div className={styles.filterRow}>
        <input
          type="text"
          className={styles.search}
          value={q}
          onChange={(event) => setQ(event.target.value)}
          placeholder="Username or display name"
          aria-label="Search people"
        />
        <ThemedSelect
          label="Role"
          value={role}
          onChange={(next) => setRole(isRole(next) ? next : "")}
        >
          <option value="">Everyone ({everyone})</option>
          <option value="member">Member ({people.member})</option>
          <option value="moderator">Moderator ({people.moderator})</option>
          <option value="admin">Admin ({people.admin})</option>
          <option value="super_admin">Super Admin ({people.superAdmin})</option>
        </ThemedSelect>
        <ThemedSelect
          label="Order"
          value={sort}
          onChange={(next) => setSort(next === "newest" || next === "oldest" ? next : "name")}
        >
          <option value="name">Name</option>
          <option value="newest">Newest</option>
          <option value="oldest">Oldest</option>
        </ThemedSelect>
        {!people.isPending && <p className={styles.matchCount}>{matchLabel(people.total)}</p>}
      </div>
      {people.isPending && <p className={simple.muted}>Loading…</p>}
      {people.error && (
        <p role="alert" className={simple.error}>
          {shortError(people.error)}
        </p>
      )}
      <div className={styles.roleBar}>
        <button
          type="button"
          className={
            staged.length > 0 ? `${styles.saveRoles} ${styles.saveRolesDirty}` : styles.saveRoles
          }
          disabled={staged.length === 0}
          onClick={saveRoles}
        >
          Save
        </button>
      </div>
      {!people.isPending && !people.error && people.users.length === 0 && (
        <p className={simple.muted}>{filtered ? "Nothing matches." : "No accounts yet."}</p>
      )}
      {people.users.length > 0 && (
        <ul className={styles.userList}>
          {people.users.map((account) => {
            const choices = ROLES.filter((choice) =>
              assignableRoles(user.role, account.role).includes(choice),
            );
            return (
              <li key={account.id} className={styles.userRow}>
                <span className={styles.userMain}>
                  <Link to="/u/$username" params={{ username: account.username }}>
                    @{account.username}
                  </Link>{" "}
                  <span className={simple.muted}>
                    {account.display_name}
                    {account.id === user.id ? " (you)" : ""} · joined{" "}
                    {new Date(account.created_at).toLocaleDateString()}
                  </span>
                </span>
                {choices.length > 0 ? (
                  <RolePicker
                    username={account.username}
                    saved={account.role}
                    selected={selectedFor(account)}
                    choices={choices}
                    onChange={(next) => stage(account, next)}
                  />
                ) : (
                  <span className={styles.roleBadge}>{ROLE_LABELS[account.role]}</span>
                )}
              </li>
            );
          })}
        </ul>
      )}
      <div ref={moreRef} aria-hidden="true" />
      {people.loadingMore && <p className={simple.muted}>Loading more…</p>}
      <ConfirmDialog
        open={confirming !== null}
        title={confirmCopy.title}
        message={confirmCopy.message}
        confirmLabel={confirmCopy.confirmLabel}
        onCancel={() => setConfirming(null)}
        onConfirm={() => {
          const pending = confirming;
          setConfirming(null);
          if (pending) applyStaged(pending);
        }}
      />
    </StaffSection>
  );
}
