// Pure role logic mirroring internal/model/role.go. No React, no transport.
export type Role = "member" | "moderator" | "admin" | "super_admin";

export const ROLES: Role[] = ["member", "moderator", "admin", "super_admin"];

export const ROLE_LABELS: Record<Role, string> = {
  member: "Member",
  moderator: "Moderator",
  admin: "Admin",
  super_admin: "Super Admin",
};

export const ROLE_DESCRIPTIONS: Record<Role, string> = {
  member: "Signed-in contributor: upload, tag, favorite and create lists.",
  moderator:
    "Curates content: fix tags/artist/rating on any post, hide posts, roll back content changes.",
  admin:
    "Manages people and content: promote members to moderator, delete posts, roll back role and content changes.",
  super_admin:
    "Owner: everything an admin can do, plus create and manage other admins. At least one must always exist.",
};

const RANK: Record<Role, number> = { member: 0, moderator: 1, admin: 2, super_admin: 3 };

export function roleRank(role: Role): number {
  return RANK[role] ?? 0;
}

export function atLeast(role: Role, min: Role): boolean {
  return roleRank(role) >= roleRank(min);
}

export function isRole(value: string): value is Role {
  return (ROLES as string[]).includes(value);
}

/** Mirrors model.CanManageRole: who may set a target's role to a new role. */
export function canManageRole(actor: Role, target: Role, next: Role): boolean {
  if (actor === "super_admin") return true;
  if (actor === "admin")
    return roleRank(target) <= RANK.moderator && roleRank(next) <= RANK.moderator;
  return false;
}

/** Roles the actor may assign to this target (empty when they manage nobody). */
export function assignableRoles(actor: Role, target: Role): Role[] {
  return ROLES.filter((r) => canManageRole(actor, target, r));
}
