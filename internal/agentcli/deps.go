package agentcli

import (
	"context"
	"io"

	"github.com/astra-chanUwU/kura-v4/internal/model"
)

// Deps is the service surface the agent CLI is allowed to call.
// Business rules stay in those services. Nil services are only valid
// for commands that do not touch the database (schema).
type Deps struct {
	Accounts    Accounts
	Board       Board
	Collections Collections
	Moderation  Moderation
	Site        Site
}

// Options configures one invocation. Tests set SessionPath; the
// operator CLI leaves it empty and uses DefaultSessionPath.
type Options struct {
	SessionPath string
}

// Accounts is the subset of accounts.Service the CLI uses.
type Accounts interface {
	LoginWithPassword(ctx context.Context, username, password string) (model.User, string, error)
	UserForSession(ctx context.Context, token string) (model.User, error)
	Logout(ctx context.Context, token string) error
	GetByUsername(ctx context.Context, username string) (model.User, error)
}

// Board is the subset of board.Service the CLI uses.
type Board interface {
	Upload(ctx context.Context, r io.Reader) (string, string, error)
	GetBySHA(ctx context.Context, sha string) (model.Post, error)
	Create(ctx context.Context, p model.Post, sha string) (model.Post, error)
	SetMetaAs(ctx context.Context, actor *model.User, id, artist string, tags []string, rating string) (model.Post, error)
	Get(ctx context.Context, id string) (model.Post, error)
	List(ctx context.Context, limit, offset int32, includeNSFW bool) ([]model.Post, error)
	ListByOwner(ctx context.Context, userID string) ([]model.Post, error)
	SearchIndex(ctx context.Context, includeNSFW bool) ([]model.Post, error)
	RecordCreation(ctx context.Context, actor model.User, post model.Post)
	Contributions(ctx context.Context, userID string) (int64, error)
}

// Collections is the subset of collections.Service the CLI uses.
type Collections interface {
	Favorite(ctx context.Context, userID, postID string) error
	Unfavorite(ctx context.Context, userID, postID string) error
	Favorites(ctx context.Context, userID string) ([]model.Post, error)
	FavoriteIDs(ctx context.Context, userID string) (map[string]bool, error)
	CreateList(ctx context.Context, ownerID, title, desc, vis string) (model.List, error)
	MyLists(ctx context.Context, ownerID string) ([]model.List, error)
	GetList(ctx context.Context, viewerID *string, id string) (model.ListDetail, error)
	AddItem(ctx context.Context, ownerID, listID, postID string) error
	RemoveItem(ctx context.Context, ownerID, listID, postID string) error
}

// Moderation is the subset of moderation.Service the CLI uses.
type Moderation interface {
	ListUsers(ctx context.Context, actor model.User) ([]model.User, error)
	SetRole(ctx context.Context, actor model.User, targetID string, newRole model.Role) (model.User, error)
	SetPostHidden(ctx context.Context, actor model.User, postID string, hidden bool, deleted bool) (model.Post, error)
	HistoryForPost(ctx context.Context, postID string, limit int32) ([]model.ChangeEntry, error)
	RecentHistory(ctx context.Context, actor model.User, limit int32) ([]model.ChangeEntry, error)
	HistoryByActor(ctx context.Context, actor model.User, actorID string, limit int32) ([]model.ChangeEntry, error)
	Rollback(ctx context.Context, actor model.User, historyID string) (model.ChangeEntry, error)
	ListHiddenPosts(ctx context.Context, actor model.User) ([]model.Post, error)
}

// Site is the subset of site.Service the CLI uses.
type Site interface {
	Get(ctx context.Context) (model.Site, error)
}
