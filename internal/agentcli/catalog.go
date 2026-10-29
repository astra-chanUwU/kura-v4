package agentcli

import "github.com/astra-chanUwU/kura-v4/internal/model"

// Command is one entry in `kura agent schema`.
// MinRole is visitor, member, moderator, or admin.
// super_admin is not a separate gate: admin commands still call the
// moderation service, which applies the finer hierarchy.
type Command struct {
	Name    string `json:"name"`
	Usage   string `json:"usage"`
	Summary string `json:"summary"`
	MinRole string `json:"min_role"`
	Mutates bool   `json:"mutates"`
}

// Catalog is the full agent surface, in the order schema prints it.
func Catalog() []Command {
	return []Command{
		{Name: "schema", Usage: "kura agent schema", Summary: "List every agent command, its arguments, and the minimum role.", MinRole: "visitor", Mutates: false},
		{Name: "login", Usage: "kura agent login --username <name> [--password <secret>]", Summary: "Sign in with a password and store a session in .local/agent-session. Password defaults to KURA_AGENT_PASSWORD. The token is never printed.", MinRole: "visitor", Mutates: true},
		{Name: "logout", Usage: "kura agent logout", Summary: "Delete the stored session and the server-side session.", MinRole: "visitor", Mutates: true},
		{Name: "whoami", Usage: "kura agent whoami", Summary: "Show the signed-in account, role, NSFW opt-in, and contribution count. Visitors get role visitor.", MinRole: "visitor", Mutates: false},
		{Name: "capabilities", Usage: "kura agent capabilities", Summary: "List the commands the current actor's role is allowed to run.", MinRole: "visitor", Mutates: false},
		{Name: "site", Usage: "kura agent site", Summary: "Show the site name and description.", MinRole: "visitor", Mutates: false},
		{Name: "posts.list", Usage: "kura agent posts list [--limit N] [--offset N]", Summary: "List public board posts, newest first. NSFW posts appear only when the actor has opted in. Limit defaults to 40 and caps at 100.", MinRole: "visitor", Mutates: false},
		{Name: "posts.get", Usage: "kura agent posts get <post-id>", Summary: "Show one post. Hidden posts are visible to their owner and to moderators. NSFW posts require an opt-in.", MinRole: "visitor", Mutates: false},
		{Name: "posts.search", Usage: "kura agent posts search [--tags a,b] [--exclude c] [--q text] [--artist text] [--sort newest|oldest] [--limit N] [--offset N]", Summary: "Find public posts. --tags must all match, --exclude drops a post carrying any of them, and --q matches an artist or a tag including a close spelling. --artist is an extra artist substring. Sort defaults to newest. Returns total and one page (limit defaults to 40, max 100). At least one of --tags, --exclude, --q, or --artist is required.", MinRole: "visitor", Mutates: false},
		{Name: "posts.mine", Usage: "kura agent posts mine [--limit N] [--offset N]", Summary: "List one page of the signed-in user's own uploads, including hidden ones. NSFW posts still follow the opt-in. Returns total and one page (limit defaults to 40, max 100).", MinRole: "member", Mutates: false},
		{Name: "posts.upload", Usage: "kura agent posts upload <file> [--artist A] [--tags a,b] [--rating sfw|nsfw]", Summary: "Upload a picture or video as the signed-in user. Identical bytes return the existing post instead of a second copy. Omitted artist and tags are read from the filename when it matches the seed pattern.", MinRole: "member", Mutates: true},
		{Name: "posts.edit", Usage: "kura agent posts edit <post-id> [--artist A] [--tags a,b] [--rating sfw|nsfw]", Summary: "Replace artist, tags, or rating on a post. Omitted fields keep their current values. Any signed-in member may edit; the change is written to history.", MinRole: "member", Mutates: true},
		{Name: "posts.hide", Usage: "kura agent posts hide <post-id>", Summary: "Hide a post from the board. The original bytes stay, and the hide is recorded in history.", MinRole: "moderator", Mutates: true},
		{Name: "posts.unhide", Usage: "kura agent posts unhide <post-id>", Summary: "Put a hidden post back on the board.", MinRole: "moderator", Mutates: true},
		{Name: "posts.restore", Usage: "kura agent posts restore <post-id>", Summary: "Clear a hide or a soft-delete and return the post to the board.", MinRole: "moderator", Mutates: true},
		{Name: "posts.delete", Usage: "kura agent posts delete <post-id>", Summary: "Soft-delete a post. Bytes and history stay so it can be restored. This is not a file deletion.", MinRole: "admin", Mutates: true},
		{Name: "posts.hidden", Usage: "kura agent posts hidden [--kind all|hidden|deleted] [--limit N] [--offset N]", Summary: "List one page of posts currently off the board, newest removal first. Returns total for the chosen kind plus the full hidden and deleted counts.", MinRole: "moderator", Mutates: false},
		{Name: "favorites.list", Usage: "kura agent favorites list [--limit N] [--offset N]", Summary: "List one page of the signed-in user's favorites. NSFW posts follow the opt-in. Returns total and one page (limit defaults to 40, max 100).", MinRole: "member", Mutates: false},
		{Name: "favorites.add", Usage: "kura agent favorites add <post-id>", Summary: "Favorite a post.", MinRole: "member", Mutates: true},
		{Name: "favorites.remove", Usage: "kura agent favorites remove <post-id>", Summary: "Remove a favorite.", MinRole: "member", Mutates: true},
		{Name: "lists.list", Usage: "kura agent lists list", Summary: "List the signed-in user's lists, public and private.", MinRole: "member", Mutates: false},
		{Name: "lists.get", Usage: "kura agent lists get <list-id> [--limit N] [--offset N]", Summary: "Show a list and one page of its posts. Private lists are visible only to their owner. total is how many posts this viewer can see.", MinRole: "visitor", Mutates: false},
		{Name: "lists.create", Usage: "kura agent lists create --title <title> [--description D] [--visibility public|private]", Summary: "Create a list. Visibility defaults to private.", MinRole: "member", Mutates: true},
		{Name: "lists.add", Usage: "kura agent lists add <list-id> <post-id>", Summary: "Add a post to a list the actor owns.", MinRole: "member", Mutates: true},
		{Name: "lists.remove", Usage: "kura agent lists remove <list-id> <post-id>", Summary: "Remove a post from a list the actor owns.", MinRole: "member", Mutates: true},
		{Name: "history.post", Usage: "kura agent history post <post-id> [--limit N]", Summary: "Show a post's change history, newest first. Any signed-in member may read it.", MinRole: "member", Mutates: false},
		{Name: "history.recent", Usage: "kura agent history recent [--limit N]", Summary: "Show the staff-wide change feed, newest first.", MinRole: "moderator", Mutates: false},
		{Name: "history.actor", Usage: "kura agent history actor <username> [--limit N]", Summary: "Show every change one account made, so each row can be rolled back.", MinRole: "moderator", Mutates: false},
		{Name: "history.rollback", Usage: "kura agent history rollback <history-id>", Summary: "Restore one history row's previous snapshot and append a rollback row. Post rows need moderator. Role rows follow the same hierarchy as a fresh role change.", MinRole: "moderator", Mutates: true},
		{Name: "users.list", Usage: "kura agent users list", Summary: "List accounts and their roles.", MinRole: "admin", Mutates: false},
		{Name: "users.role", Usage: "kura agent users role <username> <member|moderator|admin|super_admin>", Summary: "Change an account's role through the in-app hierarchy. Admins may only move member and moderator. Super admins may change anyone, and the last super admin cannot be demoted. This does not bypass checks the way `kura promote` does.", MinRole: "admin", Mutates: true},
	}
}

// Allows reports whether an actor role meets a command's minimum role.
// An empty actor role is a visitor. member is above visitor.
func Allows(actorRole, minRole string) bool {
	return roleRank(actorRole) >= roleRank(minRole)
}

// Allowed filters Catalog to commands this actor role may run.
func Allowed(actorRole string) []Command {
	out := make([]Command, 0, len(Catalog()))
	for _, cmd := range Catalog() {
		if Allows(actorRole, cmd.MinRole) {
			out = append(out, cmd)
		}
	}
	return out
}

func roleRank(role string) int {
	switch model.Role(role) {
	case model.RoleMember:
		return 1
	case model.RoleModerator:
		return 2
	case model.RoleAdmin:
		return 3
	case model.RoleSuperAdmin:
		return 4
	default:
		return 0
	}
}
