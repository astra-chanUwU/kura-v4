package agentcli

import (
	"encoding/json"

	"github.com/astra-chanUwU/kura-v4/internal/dto"
	"github.com/astra-chanUwU/kura-v4/internal/model"
)

// Identity is the whoami / login payload. It never includes a session token.
type Identity struct {
	ID            string `json:"id,omitempty"`
	Username      string `json:"username"`
	DisplayName   string `json:"display_name,omitempty"`
	Role          string `json:"role"`
	ShowNSFW      bool   `json:"show_nsfw"`
	Contributions int64  `json:"contributions,omitempty"`
}

func presentPost(p model.Post, favs map[string]bool) dto.Post {
	sourceURL := ""
	if p.Source == "twitter" && p.Artist != "" && p.SourceID != "" {
		sourceURL = "https://x.com/" + p.Artist + "/status/" + p.SourceID
	}
	tags := p.Tags
	if tags == nil {
		tags = []string{}
	}
	return dto.Post{
		ID: p.ID, OwnerUsername: p.OwnerUsername, ContentType: p.ContentType, SizeBytes: p.SizeBytes,
		CreatedAt: p.CreatedAt.Format("2006-01-02T15:04:05Z07:00"), IsFavorite: favs[p.ID],
		MediaURL: "/media/posts/" + p.ID, ThumbnailURL: "/media/posts/" + p.ID + "/thumb",
		Artist: p.Artist, SourceURL: sourceURL, Tags: tags, Width: p.Width, Height: p.Height,
		Rating: p.Rating, Hidden: p.Hidden, Deleted: p.Deleted,
	}
}

func presentPosts(posts []model.Post, favs map[string]bool) []dto.Post {
	out := make([]dto.Post, 0, len(posts))
	for _, p := range posts {
		out = append(out, presentPost(p, favs))
	}
	return out
}

func presentUser(u model.User) dto.User {
	avatarURL := ""
	if u.AvatarKey != "" {
		v := u.AvatarKey
		if len(v) > 12 {
			v = v[:12]
		}
		avatarURL = "/media/avatars/" + u.ID + "?v=" + v
	}
	return dto.User{
		ID: u.ID, Role: string(u.Role), Username: u.Username, DisplayName: u.DisplayName,
		Bio: u.Bio, Location: u.Location, Website: u.Website,
		CreatedAt: u.CreatedAt.Format("2006-01-02T15:04:05Z07:00"), AvatarURL: avatarURL,
	}
}

func presentUsers(users []model.User) []dto.User {
	out := make([]dto.User, 0, len(users))
	for _, u := range users {
		out = append(out, presentUser(u))
	}
	return out
}

func presentList(l model.List) dto.List {
	return dto.List{
		ID: l.ID, OwnerUsername: l.OwnerUsername, Title: l.Title, Description: l.Description,
		Visibility: l.Visibility, ItemCount: l.ItemCount,
		CreatedAt: l.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: l.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func presentLists(lists []model.List) []dto.List {
	out := make([]dto.List, 0, len(lists))
	for _, l := range lists {
		out = append(out, presentList(l))
	}
	return out
}

func presentHistory(e model.ChangeEntry) dto.ChangeEntry {
	decode := func(raw json.RawMessage) map[string]any {
		out := map[string]any{}
		if len(raw) > 0 {
			_ = json.Unmarshal(raw, &out)
		}
		return out
	}
	actorID := ""
	if e.ActorID != nil {
		actorID = *e.ActorID
	}
	thumbnailURL := ""
	if e.EntityType == "post" {
		thumbnailURL = "/media/posts/" + e.EntityID + "/thumb"
	}
	return dto.ChangeEntry{
		ID: e.ID, EntityType: e.EntityType, EntityID: e.EntityID, ThumbnailURL: thumbnailURL,
		ActorID: actorID, ActorUsername: e.ActorUsername, Action: e.Action,
		Before: decode(e.Before), After: decode(e.After), RollbackOf: e.RollbackOf,
		CreatedAt: e.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func presentHistories(entries []model.ChangeEntry) []dto.ChangeEntry {
	out := make([]dto.ChangeEntry, 0, len(entries))
	for _, entry := range entries {
		out = append(out, presentHistory(entry))
	}
	return out
}
