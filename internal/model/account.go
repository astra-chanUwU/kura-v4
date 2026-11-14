package model

import "time"

type User struct {
	ID          string
	Username    string
	DisplayName string
	Bio         string
	Location    string
	Website     string
	CreatedAt   time.Time
	AvatarKey   string
	AvatarType  string
	ShowNSFW    bool
	// Default appearance: applied on every device and shown on the
	// public profile. Ids are validated in the accounts service.
	PreferredTheme string
	PreferredFont  string
	// How many rows of on-screen tag suggestions the board shows.
	// 0 hides the strip. Default 1 matches the single scrolling row.
	TagSuggestionRows int
	// How many posts the board shows per page. Default 40; allowed 12–100.
	PostsPerPage int
	Role         Role
	// Default board tag filters, applied when the board loads.
	DefaultIncludeTags []string
	DefaultExcludeTags []string
}

// UserPage is one page of the staff people directory.
// Member, Moderator, Admin, and SuperAdmin are global role counts
// (they ignore the search and role filter). Total is how many accounts
// match that filter. Users is one page.
type UserPage struct {
	Total      int64
	Member     int64
	Moderator  int64
	Admin      int64
	SuperAdmin int64
	Users      []User
}

type Passkey struct {
	ID         string
	Name       string
	CreatedAt  time.Time
	LastUsedAt *time.Time
}

type StoredCredential struct {
	RowID           string
	CredentialID    []byte
	PublicKey       []byte
	AttestationType string
	Transports      []string
	UserPresent     bool
	UserVerified    bool
	BackupEligible  bool
	BackupState     bool
	AAGUID          []byte
	SignCount       int64
	Name            string
	CreatedAt       time.Time
	LastUsedAt      *time.Time
}

type Challenge struct {
	Purpose     string
	Username    string
	UserID      *string
	SessionData []byte
}

type AccountSecurity struct {
	HasPassword          bool
	Passkeys             []Passkey
	HasRecoveryKey       bool
	RecoveryKeyCreatedAt *time.Time
}

type Post struct {
	ID                 string
	OwnerID            *string
	OwnerUsername      string
	OriginalName       string
	StorageKey         string
	ContentType        string
	SizeBytes          int64
	CreatedAt          time.Time
	Artist             string
	Source             string // e.g. "twitter"
	SourceID           string // e.g. the tweet id
	SourceURL          string // member-set http(s) link; empty when unset
	ThumbnailKey       string
	SampleKey          string // medium JPEG; empty until post.derive stores one
	Tags               []string
	Width              int
	Height             int
	Rating             string // sfw | nsfw
	Hidden             bool   // hidden by a moderator/admin; restorable via history
	Deleted            bool   // soft-deleted by an admin (hidden, and flagged deleted)
	FavoriteCount      int64  // public number of favorites
	LastTaggerUsername string // latest post.meta_updated actor; empty when nobody has retagged
}

type List struct {
	ID               string
	OwnerID          string
	OwnerUsername    string
	OwnerDisplayName string
	Title            string
	Description      string
	Visibility       string // private | public
	ItemCount        int64
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type ListDetail struct {
	List  List
	Items []Post
	Total int64
}

type HiddenPage struct {
	Total   int64
	Hidden  int64
	Deleted int64
	Posts   []Post
}
