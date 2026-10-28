package controllers

import (
	"errors"

	"github.com/astra-chanUwU/kura-v4/internal/accounts"
	"github.com/astra-chanUwU/kura-v4/internal/board"
	"github.com/astra-chanUwU/kura-v4/internal/collections"
	"github.com/astra-chanUwU/kura-v4/internal/dto"
	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/astra-chanUwU/kura-v4/internal/moderation"
	"github.com/gofiber/fiber/v3"
)

const sessionCookie = "kura_session"
const localUser = "kura_user"

func writeError(c fiber.Ctx, err error) error {
	var vErr *accounts.ValidationError
	var cErr *collections.ValidationError
	var mErr *moderation.ValidationError
	switch {
	case errors.As(err, &vErr):
		return fiber.NewError(400, vErr.Message)
	case errors.As(err, &cErr):
		return fiber.NewError(400, cErr.Message)
	case errors.As(err, &mErr):
		return fiber.NewError(400, mErr.Message)
	case errors.Is(err, moderation.ErrForbidden):
		return fiber.NewError(403, "You don't have permission to do that.")
	case errors.Is(err, moderation.ErrNotFound):
		return fiber.NewError(404, "Not found.")
	case errors.Is(err, accounts.ErrInvalidCredentials):
		return fiber.NewError(401, "That username and sign-in didn't match. Check it and try again.")
	case errors.Is(err, accounts.ErrUsernameTaken):
		return fiber.NewError(409, "That username is taken. Try another one.")
	case errors.Is(err, accounts.ErrAdultConfirmationRequired):
		return fiber.NewError(400, "Please confirm that you are 18 or older first.")
	case errors.Is(err, accounts.ErrLastSignInMethod):
		return fiber.NewError(400, err.Error())
	case errors.Is(err, accounts.ErrCeremonyExpired):
		return fiber.NewError(400, err.Error())
	case errors.Is(err, accounts.ErrNotFound), errors.Is(err, collections.ErrNotFound), errors.Is(err, board.ErrNotFound):
		return fiber.NewError(404, "Not found.")
	case errors.Is(err, collections.ErrForbidden):
		return fiber.NewError(403, "That's not yours.")
	default:
		return err
	}
}

// viewerSeesNSFW is the server-side gate: NSFW posts only exist for
// signed-in users who opted in (with the 18+ confirmation).
func viewerSeesNSFW(c fiber.Ctx) bool {
	u := currentUser(c)
	return u != nil && u.ShowNSFW
}

func currentUser(c fiber.Ctx) *model.User {
	if u, ok := c.Locals(localUser).(model.User); ok {
		return &u
	}
	return nil
}

func requireUser(c fiber.Ctx) (*model.User, error) {
	if u := currentUser(c); u != nil {
		return u, nil
	}
	return nil, fiber.NewError(401, "Sign in first.")
}

func userDTO(u model.User) dto.User {
	avatarURL := ""
	if u.AvatarKey != "" {
		// Versioned by the content hash: a new picture is a new URL, so
		// browsers never serve the previous avatar from cache.
		v := u.AvatarKey
		if len(v) > 12 {
			v = v[:12]
		}
		avatarURL = "/media/avatars/" + u.ID + "?v=" + v
	}
	return dto.User{ID: u.ID, Role: string(u.Role), Username: u.Username, DisplayName: u.DisplayName, Bio: u.Bio, Location: u.Location, Website: u.Website, CreatedAt: u.CreatedAt.Format("2006-01-02T15:04:05Z07:00"), AvatarURL: avatarURL}
}

func postDTO(p model.Post, favs map[string]bool) dto.Post {
	sourceURL := ""
	if p.Source == "twitter" && p.Artist != "" && p.SourceID != "" {
		sourceURL = "https://x.com/" + p.Artist + "/status/" + p.SourceID
	}
	tags := p.Tags
	if tags == nil {
		tags = []string{}
	}
	return dto.Post{ID: p.ID, OwnerUsername: p.OwnerUsername, ContentType: p.ContentType, SizeBytes: p.SizeBytes,
		CreatedAt: p.CreatedAt.Format("2006-01-02T15:04:05Z07:00"), IsFavorite: favs[p.ID], MediaURL: "/media/posts/" + p.ID,
		ThumbnailURL: "/media/posts/" + p.ID + "/thumb", Artist: p.Artist, SourceURL: sourceURL, Tags: tags, Width: p.Width, Height: p.Height, Rating: p.Rating, Hidden: p.Hidden, Deleted: p.Deleted}
}

func postsDTO(posts []model.Post, favs map[string]bool) []dto.Post {
	out := make([]dto.Post, 0, len(posts))
	for _, p := range posts {
		out = append(out, postDTO(p, favs))
	}
	return out
}

func listDTO(l model.List) dto.List {
	return dto.List{ID: l.ID, OwnerUsername: l.OwnerUsername, Title: l.Title, Description: l.Description, Visibility: l.Visibility,
		ItemCount: l.ItemCount, CreatedAt: l.CreatedAt.Format("2006-01-02T15:04:05Z07:00"), UpdatedAt: l.UpdatedAt.Format("2006-01-02T15:04:05Z07:00")}
}

func listsDTO(lists []model.List) []dto.List {
	out := make([]dto.List, 0, len(lists))
	for _, l := range lists {
		out = append(out, listDTO(l))
	}
	return out
}

func preferencesDTO(u model.User) dto.Preferences {
	include := u.DefaultIncludeTags
	if include == nil {
		include = []string{}
	}
	exclude := u.DefaultExcludeTags
	if exclude == nil {
		exclude = []string{}
	}
	theme := u.PreferredTheme
	if theme == "" {
		theme = "kura"
	}
	font := u.PreferredFont
	if font == "" {
		font = "system"
	}
	rows := u.TagSuggestionRows
	if rows < 0 || rows > 10 {
		rows = 1
	}
	return dto.Preferences{ShowNSFW: u.ShowNSFW, DefaultIncludeTags: include, DefaultExcludeTags: exclude, Theme: theme, Font: font, TagSuggestionRows: rows}
}

func securityDTO(sec model.AccountSecurity) dto.Security {
	out := dto.Security{HasPassword: sec.HasPassword, Passkeys: []dto.Passkey{}, HasRecoveryKey: sec.HasRecoveryKey}
	if sec.RecoveryKeyCreatedAt != nil {
		s := sec.RecoveryKeyCreatedAt.Format("2006-01-02T15:04:05Z07:00")
		out.RecoveryKeyCreatedAt = &s
	}
	for _, p := range sec.Passkeys {
		var last *string
		if p.LastUsedAt != nil {
			s := p.LastUsedAt.Format("2006-01-02T15:04:05Z07:00")
			last = &s
		}
		out.Passkeys = append(out.Passkeys, dto.Passkey{ID: p.ID, Name: p.Name, CreatedAt: p.CreatedAt.Format("2006-01-02T15:04:05Z07:00"), LastUsedAt: last})
	}
	return out
}

// canSeePost reports whether the viewer may see a post at all: hidden
// posts are staff/owner-only; NSFW gating is applied separately.
func canSeePost(c fiber.Ctx, p model.Post) bool {
	if !p.Hidden {
		return true
	}
	u := currentUser(c)
	if u == nil {
		return false
	}
	if u.Role.AtLeast(model.RoleModerator) {
		return true
	}
	return p.OwnerID != nil && *p.OwnerID == u.ID
}
