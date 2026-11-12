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
		{Name: "posts.search", Usage: "kura agent posts search [--tags a,b] [--exclude c] [--q text] [--artist text] [--sort newest|oldest|favorites] [--limit N] [--offset N]", Summary: "Find public posts. --tags must all match, --exclude drops a post carrying any of them, and --q matches an artist or a tag including a close spelling. --artist is an extra artist substring. Sort defaults to newest. Returns total and one page (limit defaults to 40, max 100). At least one of --tags, --exclude, --q, or --artist is required.", MinRole: "visitor", Mutates: false},
		{Name: "posts.mine", Usage: "kura agent posts mine [--limit N] [--offset N]", Summary: "List one page of the signed-in user's own uploads, including hidden ones. NSFW posts still follow the opt-in. Returns total and one page (limit defaults to 40, max 100).", MinRole: "member", Mutates: false},
		{Name: "posts.upload", Usage: "kura agent posts upload <file> [--artist A] [--tags a,b] [--rating sfw|nsfw] [--source-url URL]", Summary: "Upload a picture or video as the signed-in user. Identical bytes return the existing post instead of a second copy. Omitted artist and tags are read from the filename when it matches the seed pattern. --source-url is an optional http(s) link.", MinRole: "member", Mutates: true},
		{Name: "posts.edit", Usage: "kura agent posts edit <post-id> [--artist A] [--tags a,b] [--rating sfw|nsfw] [--source-url URL]", Summary: "Replace artist, tags, rating, or source link on a post. Omitted fields keep their current values. An empty --source-url clears the link. Any signed-in member may edit; the change is written to history.", MinRole: "member", Mutates: true},
		{Name: "posts.hide", Usage: "kura agent posts hide <post-id>", Summary: "Hide a post from the board. The original bytes stay, and the hide is recorded in history.", MinRole: "moderator", Mutates: true},
		{Name: "posts.unhide", Usage: "kura agent posts unhide <post-id>", Summary: "Put a hidden post back on the board.", MinRole: "moderator", Mutates: true},
		{Name: "posts.restore", Usage: "kura agent posts restore <post-id>", Summary: "Clear a hide or a soft-delete and return the post to the board.", MinRole: "moderator", Mutates: true},
		{Name: "posts.delete", Usage: "kura agent posts delete <post-id>", Summary: "Soft-delete a post. Bytes and history stay so it can be restored. This is not a file deletion.", MinRole: "admin", Mutates: true},
		{Name: "posts.hidden", Usage: "kura agent posts hidden [--kind all|hidden|deleted] [--q text] [--rating sfw|nsfw] [--sort newest|oldest] [--limit N] [--offset N]", Summary: "List one page of posts currently off the board. --q matches an artist, uploader, remover, or tag. --rating is sfw or nsfw. Sort defaults to newest removal. Returns the filtered total plus the full hidden and deleted counts.", MinRole: "moderator", Mutates: false},
		{Name: "comments.list", Usage: "kura agent comments list <post-id> [--limit N] [--offset N]", Summary: "List one page of comments on a post, oldest first. NSFW posts require an opt-in, and hidden or deleted posts are visible only to the owner or a moderator. Returns total and one page (limit defaults to 40, max 100).", MinRole: "visitor", Mutates: false},
		{Name: "comments.create", Usage: "kura agent comments create <post-id> --body <text>", Summary: "Add a comment to a post the actor can see. The body is trimmed, empty text is rejected, and the limit is 2000 characters.", MinRole: "member", Mutates: true},
		{Name: "comments.update", Usage: "kura agent comments update <comment-id> --body <text>", Summary: "Replace the body of a comment. Only the author can update it.", MinRole: "member", Mutates: true},
		{Name: "comments.delete", Usage: "kura agent comments delete <comment-id>", Summary: "Delete a comment. The author can delete their own; a moderator or above can delete one they are allowed to see.", MinRole: "member", Mutates: true},
		{Name: "favorites.list", Usage: "kura agent favorites list [--limit N] [--offset N]", Summary: "List one page of the signed-in user's favorites. NSFW posts follow the opt-in. Returns total and one page (limit defaults to 40, max 100).", MinRole: "member", Mutates: false},
		{Name: "favorites.add", Usage: "kura agent favorites add <post-id>", Summary: "Favorite a post.", MinRole: "member", Mutates: true},
		{Name: "favorites.remove", Usage: "kura agent favorites remove <post-id>", Summary: "Remove a favorite.", MinRole: "member", Mutates: true},
		{Name: "lists.list", Usage: "kura agent lists list", Summary: "List the signed-in user's lists, public and private.", MinRole: "member", Mutates: false},
		{Name: "lists.get", Usage: "kura agent lists get <list-id> [--limit N] [--offset N]", Summary: "Show a list and one page of its posts. Private lists are visible only to their owner. total is how many posts this viewer can see.", MinRole: "visitor", Mutates: false},
		{Name: "lists.create", Usage: "kura agent lists create --title <title> [--description D] [--visibility public|private]", Summary: "Create a list. Visibility defaults to private.", MinRole: "member", Mutates: true},
		{Name: "lists.add", Usage: "kura agent lists add <list-id> <post-id>", Summary: "Add a post to a list the actor owns.", MinRole: "member", Mutates: true},
		{Name: "lists.remove", Usage: "kura agent lists remove <list-id> <post-id>", Summary: "Remove a post from a list the actor owns.", MinRole: "member", Mutates: true},
		{Name: "history.post", Usage: "kura agent history post <post-id> [--limit N]", Summary: "Show a post's change history, newest first. Any signed-in member may read it.", MinRole: "member", Mutates: false},
		{Name: "history.recent", Usage: "kura agent history recent [--q text] [--actor <username>] [--action <action>] [--entity post|user] [--sort newest|oldest] [--limit N] [--offset N]", Summary: "Show the staff change feed. With no --q, --actor, --action, --entity, --sort, or --offset, this is the newest rows and --limit defaults to 50 (max 200). Any of those filters returns the filtered total and one page; --limit then defaults to 40 and caps at 100. --q matches an actor, artist, role, rating, source link, or tags. --actor is a username. --action is post.created, post.meta_updated, post.hidden, post.unhidden, post.deleted, user.role_changed, or rollback. --entity is post or user. Sort defaults to newest.", MinRole: "moderator", Mutates: false},
		{Name: "history.actor", Usage: "kura agent history actor <username> [--q text] [--action <action>] [--entity post|user] [--sort newest|oldest] [--limit N] [--offset N]", Summary: "Show every change one account made, so each row can be rolled back. --limit alone keeps that newest-first list (default 50, max 200). --q, --action, --entity, --sort, or --offset return the filtered total and one page (limit defaults to 40, max 100). Filters match history recent.", MinRole: "moderator", Mutates: false},
		{Name: "history.actors", Usage: "kura agent history actors [--q text]", Summary: "Find up to 8 usernames matching --q, case-insensitive, ordered by username. An empty query returns an empty list.", MinRole: "moderator", Mutates: false},
		{Name: "history.rollback", Usage: "kura agent history rollback <history-id>", Summary: "Restore one history row's previous snapshot and append a rollback row. Post rows need moderator. Role rows follow the same hierarchy as a fresh role change.", MinRole: "moderator", Mutates: true},
		{Name: "tags.show", Usage: "kura agent tags show <tag>", Summary: "Show a tag page: category, description, aliases, post count, and recent posts. An alias name shows the canonical tag it points at.", MinRole: "visitor", Mutates: false},
		{Name: "tags.edit", Usage: "kura agent tags edit <tag> --category <general|artist|character|copyright|meta> --description <text>", Summary: "Replace a tag's category and description. Editing an alias updates the canonical tag. Any signed-in member may edit.", MinRole: "member", Mutates: true},
		{Name: "tags.alias", Usage: "kura agent tags alias <target> <alias>", Summary: "Point an alternate name at a canonical tag and move posts that used the alternate name onto the target. Searching either name then finds the same pictures. The target cannot itself be an alias.", MinRole: "moderator", Mutates: true},
		{Name: "tags.unalias", Usage: "kura agent tags unalias <target> <alias>", Summary: "Stop an alternate name pointing at a tag. Posts already moved to the canonical tag stay there.", MinRole: "moderator", Mutates: true},
		{Name: "tags.imply", Usage: "kura agent tags imply <antecedent> <consequent>", Summary: "When a post has the antecedent tag, also give it the consequent. Existing posts are updated. A loop is rejected.", MinRole: "moderator", Mutates: true},
		{Name: "tags.unimply", Usage: "kura agent tags unimply <antecedent> <consequent>", Summary: "Remove an implication. Posts that already have the consequent keep it.", MinRole: "moderator", Mutates: true},
		{Name: "reports.create", Usage: "kura agent reports create <post-id> --reason <illegal|underage|spam|wrong_rating|other> [--note text]", Summary: "Report a post the actor can see. One open report per member per post. This does not hide the post.", MinRole: "member", Mutates: true},
		{Name: "reports.list", Usage: "kura agent reports list [--limit N] [--offset N]", Summary: "List open post reports, newest first.", MinRole: "moderator", Mutates: false},
		{Name: "reports.dismiss", Usage: "kura agent reports dismiss <report-id>", Summary: "Close an open report. The post stays as it is.", MinRole: "moderator", Mutates: true},
		{Name: "users.list", Usage: "kura agent users list [--q text] [--role member|moderator|admin|super_admin] [--sort name|newest|oldest] [--limit N] [--offset N]", Summary: "List accounts and their roles. With no flags, every account is returned. Any of --q, --role, --sort, --limit, or --offset returns one page: total is how many accounts match --q and --role, and member, moderator, admin, and super_admin are the full role counts. --q matches a username or display name. Sort defaults to name. Limit defaults to 40 and caps at 100.", MinRole: "admin", Mutates: false},
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
