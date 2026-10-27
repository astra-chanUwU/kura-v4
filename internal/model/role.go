package model

// Role is a stored account tier. Visitor is not a Role: a visitor is an
// anonymous request with no account (see IsVisitor helpers in services).
type Role string

const (
	RoleMember     Role = "member"
	RoleModerator  Role = "moderator"
	RoleAdmin      Role = "admin"
	RoleSuperAdmin Role = "super_admin"
)

func (r Role) Valid() bool {
	switch r {
	case RoleMember, RoleModerator, RoleAdmin, RoleSuperAdmin:
		return true
	}
	return false
}

// Rank orders tiers: member < moderator < admin < super_admin.
func (r Role) Rank() int {
	switch r {
	case RoleModerator:
		return 1
	case RoleAdmin:
		return 2
	case RoleSuperAdmin:
		return 3
	default:
		return 0
	}
}

func (r Role) AtLeast(other Role) bool { return r.Rank() >= other.Rank() }

// CanManageRole reports whether actor may set target (currently targetRole)
// to newRole. Super admins manage anyone; admins manage member<->moderator
// only and can never grant or touch admin/super_admin; nobody else manages
// roles. Callers additionally enforce the last-super-admin guard.
func CanManageRole(actor, target Role, newRole Role) bool {
	if !actor.Valid() || !newRole.Valid() {
		return false
	}
	if actor == RoleSuperAdmin {
		return true
	}
	if actor == RoleAdmin {
		return target.Rank() <= RoleModerator.Rank() && newRole.Rank() <= RoleModerator.Rank()
	}
	return false
}

// PostSnapshot is the rollback-able state of a post entry.
type PostSnapshot struct {
	Artist  string   `json:"artist"`
	Tags    []string `json:"tags"`
	Rating  string   `json:"rating"`
	Hidden  bool     `json:"hidden"`
	Deleted bool     `json:"deleted"`
}

// RoleSnapshot is the rollback-able state of a user entry.
type RoleSnapshot struct {
	Role Role `json:"role"`
}
