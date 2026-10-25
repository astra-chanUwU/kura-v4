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
	ID            string
	OwnerID       *string
	OwnerUsername string
	OriginalName  string
	StorageKey    string
	ContentType   string
	SizeBytes     int64
	CreatedAt     time.Time
	Artist        string
	Source        string // e.g. "twitter"
	SourceID      string // e.g. the tweet id
	ThumbnailKey  string
	Tags          []string
	Width         int
	Height        int
	Rating        string // sfw | nsfw
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
}
