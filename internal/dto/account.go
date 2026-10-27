package dto

// Mirrored in frontend/src/types/api.ts.

type User struct {
	ID          string `json:"id"`
	Role        string `json:"role"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Bio         string `json:"bio"`
	Location    string `json:"location"`
	Website     string `json:"website"`
	CreatedAt   string `json:"created_at"`
	AvatarURL   string `json:"avatar_url"`
}

type Profile struct {
	User          User   `json:"user"`
	PublicLists   []List `json:"public_lists"`
	Contributions int64  `json:"contributions"`
}

type Post struct {
	ID            string   `json:"id"`
	OwnerUsername string   `json:"owner_username"`
	ContentType   string   `json:"content_type"`
	SizeBytes     int64    `json:"size_bytes"`
	CreatedAt     string   `json:"created_at"`
	IsFavorite    bool     `json:"is_favorite"`
	MediaURL      string   `json:"media_url"`
	ThumbnailURL  string   `json:"thumbnail_url"`
	Artist        string   `json:"artist"`
	Width         int      `json:"width"`
	Height        int      `json:"height"`
	Rating        string   `json:"rating"`
	Hidden        bool     `json:"hidden"`
	SourceURL     string   `json:"source_url"`
	Tags          []string `json:"tags"`
}

// UpdatePostMeta replaces a post's artist and tags.
type UpdatePostMeta struct {
	Artist string   `json:"artist"`
	Tags   []string `json:"tags"`
	Rating string   `json:"rating"`
}

// Preferences are the account's content settings.
type Preferences struct {
	ShowNSFW           bool     `json:"show_nsfw"`
	DefaultIncludeTags []string `json:"default_include_tags"`
	DefaultExcludeTags []string `json:"default_exclude_tags"`
}

// UpdateTagPreferences replaces the account's default board tag filters.
type UpdateTagPreferences struct {
	IncludeTags []string `json:"include_tags"`
	ExcludeTags []string `json:"exclude_tags"`
}

// UpdatePreferences changes content settings; enabling NSFW requires
// the over-18 confirmation flag.
type UpdatePreferences struct {
	ShowNSFW bool `json:"show_nsfw"`
	Over18   bool `json:"over_18"`
}

type List struct {
	ID            string `json:"id"`
	OwnerUsername string `json:"owner_username"`
	Title         string `json:"title"`
	Description   string `json:"description"`
	Visibility    string `json:"visibility"`
	ItemCount     int64  `json:"item_count"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

type ListDetail struct {
	List  List   `json:"list"`
	Items []Post `json:"items"`
}

type Passkey struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	CreatedAt  string  `json:"created_at"`
	LastUsedAt *string `json:"last_used_at"`
}

type Security struct {
	HasPassword          bool      `json:"has_password"`
	Passkeys             []Passkey `json:"passkeys"`
	HasRecoveryKey       bool      `json:"has_recovery_key"`
	RecoveryKeyCreatedAt *string   `json:"recovery_key_created_at"`
}

type Me struct {
	User        User        `json:"user"`
	Security    Security    `json:"security"`
	Preferences Preferences `json:"preferences"`
	// RecoveryKey is set only in the response that issues a key
	// (registration, key regeneration, account recovery). Shown once.
	RecoveryKey string `json:"recovery_key,omitempty"`
}

// RecoverAccount resets sign-in with a recovery key and a new password.
type RecoverAccount struct {
	Username    string `json:"username"`
	RecoveryKey string `json:"recovery_key"`
	NewPassword string `json:"new_password"`
}

type Credentials struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
}

type UpdateProfile struct {
	DisplayName string `json:"display_name"`
	Bio         string `json:"bio"`
	Location    string `json:"location"`
	Website     string `json:"website"`
}

type ListInput struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Visibility  string `json:"visibility"`
}

type ReorderInput struct {
	PostIDs []string `json:"post_ids"`
}

// WebAuthnCeremony starts a ceremony; the client echoes CeremonyID on finish.
type WebAuthnCeremony struct {
	CeremonyID string `json:"ceremony_id"`
	Options    any    `json:"options"`
}

// Role mirrors model.Role: member | moderator | admin | super_admin.
// Visitor is anonymous and has no role value.
type SetRole struct {
	Role string `json:"role"`
}

// ChangeEntry is one history row. Before/After decode per entity:
// post -> PostSnapshot fields; user -> {"role": "..."}.
type ChangeEntry struct {
	ID            string         `json:"id"`
	EntityType    string         `json:"entity_type"`
	EntityID      string         `json:"entity_id"`
	ActorID       string         `json:"actor_id"`
	ActorUsername string         `json:"actor_username"`
	Action        string         `json:"action"`
	Before        map[string]any `json:"before"`
	After         map[string]any `json:"after"`
	RollbackOf    *string        `json:"rollback_of"`
	CreatedAt     string         `json:"created_at"`
}
