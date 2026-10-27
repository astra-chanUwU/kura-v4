import { describe, expect, it } from "vitest";
import { assignableRoles, atLeast, canManageRole } from "./roles";

describe("canManageRole (mirrors Go model.CanManageRole)", () => {
  it("super admin manages anyone, including granting admin", () => {
    expect(canManageRole("super_admin", "member", "admin")).toBe(true);
    expect(canManageRole("super_admin", "admin", "member")).toBe(true);
    expect(canManageRole("super_admin", "super_admin", "member")).toBe(true);
  });
  it("admin only moves member<->moderator", () => {
    expect(canManageRole("admin", "member", "moderator")).toBe(true);
    expect(canManageRole("admin", "moderator", "member")).toBe(true);
    expect(canManageRole("admin", "member", "admin")).toBe(false);
    expect(canManageRole("admin", "admin", "member")).toBe(false);
    expect(canManageRole("admin", "super_admin", "member")).toBe(false);
  });
  it("moderators and members manage nobody", () => {
    expect(canManageRole("moderator", "member", "member")).toBe(false);
    expect(canManageRole("member", "member", "moderator")).toBe(false);
    expect(assignableRoles("moderator", "member")).toEqual([]);
    expect(atLeast("moderator", "moderator")).toBe(true);
    expect(atLeast("member", "moderator")).toBe(false);
  });
});
