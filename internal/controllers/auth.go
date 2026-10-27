package controllers

import (
	"net/url"
	"strings"
	"time"

	"github.com/astra-chanUwU/kura-v4/internal/accounts"
	"github.com/astra-chanUwU/kura-v4/internal/board"
	"github.com/astra-chanUwU/kura-v4/internal/collections"
	"github.com/astra-chanUwU/kura-v4/internal/dto"
	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/gofiber/fiber/v3"
)

type Auth struct {
	accounts    *accounts.Service
	collections *collections.Service
	board       *board.Service
	secure      bool
}

func NewAuth(accountsSvc *accounts.Service, collectionsSvc *collections.Service, boardSvc *board.Service, secure bool) *Auth {
	return &Auth{accounts: accountsSvc, collections: collectionsSvc, board: boardSvc, secure: secure}
}

// OptionalAuth resolves the session cookie when present; it never rejects.
func (h *Auth) OptionalAuth(c fiber.Ctx) error {
	token := c.Cookies(sessionCookie)
	if token != "" {
		if user, err := h.accounts.UserForSession(c.Context(), token); err == nil {
			c.Locals(localUser, user)
		}
	}
	return c.Next()
}

func (h *Auth) setSession(c fiber.Ctx, token string) {
	c.Cookie(&fiber.Cookie{Name: sessionCookie, Value: token, Path: "/", HTTPOnly: true, Secure: h.secure, SameSite: "Lax", MaxAge: 30 * 24 * 3600, Expires: time.Now().Add(30 * 24 * time.Hour)})
}

func (h *Auth) clearSession(c fiber.Ctx) {
	c.Cookie(&fiber.Cookie{Name: sessionCookie, Value: "", Path: "/", HTTPOnly: true, Secure: h.secure, SameSite: "Lax", MaxAge: -1})
}

func (h *Auth) meResponse(c fiber.Ctx, user model.User) error {
	return h.meResponseWithKey(c, user, "")
}

// meResponseWithKey includes a freshly issued recovery key (shown once)
// when one was just created for this response.
func (h *Auth) meResponseWithKey(c fiber.Ctx, user model.User, recoveryKey string) error {
	sec, err := h.accounts.Security(c.Context(), user.ID)
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(dto.Me{User: userDTO(user), Security: securityDTO(sec), Preferences: preferencesDTO(user), RecoveryKey: recoveryKey})
}

func (h *Auth) RegisterRoutes(api fiber.Router) {
	api.Post("/auth/register", h.register)
	api.Post("/auth/login", h.login)
	api.Post("/auth/logout", h.logout)
	api.Post("/auth/passkey/register/begin", h.passkeyRegisterBegin)
	api.Post("/auth/passkey/register/finish", h.passkeyRegisterFinish)
	api.Post("/auth/passkey/login/begin", h.passkeyLoginBegin)
	api.Post("/auth/passkey/login/finish", h.passkeyLoginFinish)
	api.Get("/me", h.me)
	api.Patch("/me/profile", h.updateProfile)
	api.Post("/auth/recover", h.recover)
	api.Patch("/me/preferences", h.updatePreferences)
	api.Patch("/me/tag-preferences", h.updateTagPreferences)
	api.Patch("/me/appearance", h.updateAppearance)
	api.Post("/me/avatar", h.setAvatar)
	api.Delete("/me/avatar", h.removeAvatar)
	api.Post("/me/password", h.setPassword)
	api.Post("/me/recovery-key", h.regenerateRecoveryKey)
	api.Delete("/me/password", h.removePassword)
	api.Delete("/me/passkeys/:id", h.removePasskey)
	api.Get("/users/:username", h.publicProfile)
}

func (h *Auth) register(c fiber.Ctx) error {
	var in dto.Credentials
	if err := c.Bind().Body(&in); err != nil {
		return fiber.NewError(400, "Invalid request.")
	}
	user, token, key, err := h.accounts.RegisterWithPassword(c.Context(), in.Username, in.Password, in.DisplayName)
	if err != nil {
		return writeError(c, err)
	}
	h.setSession(c, token)
	sec, err := h.accounts.Security(c.Context(), user.ID)
	if err != nil {
		return writeError(c, err)
	}
	return c.Status(201).JSON(dto.Me{User: userDTO(user), Security: securityDTO(sec), Preferences: preferencesDTO(user), RecoveryKey: key})
}

func (h *Auth) login(c fiber.Ctx) error {
	var in dto.Credentials
	if err := c.Bind().Body(&in); err != nil {
		return fiber.NewError(400, "Invalid request.")
	}
	user, token, err := h.accounts.LoginWithPassword(c.Context(), in.Username, in.Password)
	if err != nil {
		return writeError(c, err)
	}
	h.setSession(c, token)
	return h.meResponse(c, user)
}

func (h *Auth) logout(c fiber.Ctx) error {
	if err := h.accounts.Logout(c.Context(), c.Cookies(sessionCookie)); err != nil {
		return writeError(c, err)
	}
	h.clearSession(c)
	return c.SendStatus(204)
}

type usernameInput struct {
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
}

// requestOrigin returns the browser origin for a WebAuthn ceremony,
// falling back to the Referer origin. "" lets the service use its
// configured RPID (e.g. non-browser callers).
func requestOrigin(c fiber.Ctx) string {
	if origin := strings.TrimSpace(c.Get("Origin")); origin != "" {
		return origin
	}
	referer := strings.TrimSpace(c.Get("Referer"))
	if referer == "" {
		return ""
	}
	if parsed, err := url.Parse(referer); err == nil && parsed.Scheme != "" && parsed.Host != "" {
		return parsed.Scheme + "://" + parsed.Host
	}
	return ""
}

func (h *Auth) passkeyRegisterBegin(c fiber.Ctx) error {
	var in usernameInput
	_ = c.Bind().Body(&in)
	ceremonyID, options, err := h.accounts.BeginPasskeyRegistration(c.Context(), currentUser(c), in.Username, in.DisplayName, requestOrigin(c))
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(dto.WebAuthnCeremony{CeremonyID: ceremonyID, Options: options})
}

func (h *Auth) passkeyRegisterFinish(c fiber.Ctx) error {
	ceremonyID := c.Query("ceremony")
	user, token, key, err := h.accounts.FinishPasskeyRegistration(c.Context(), ceremonyID, c.Body(), c.Query("name"))
	if err != nil {
		return writeError(c, err)
	}
	if token != "" {
		h.setSession(c, token)
	}
	return h.meResponseWithKey(c, user, key)
}

func (h *Auth) passkeyLoginBegin(c fiber.Ctx) error {
	var in usernameInput
	if err := c.Bind().Body(&in); err != nil {
		return fiber.NewError(400, "Invalid request.")
	}
	ceremonyID, options, err := h.accounts.BeginPasskeyLogin(c.Context(), in.Username, requestOrigin(c))
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(dto.WebAuthnCeremony{CeremonyID: ceremonyID, Options: options})
}

func (h *Auth) passkeyLoginFinish(c fiber.Ctx) error {
	user, token, err := h.accounts.FinishPasskeyLogin(c.Context(), c.Query("ceremony"), c.Body())
	if err != nil {
		return writeError(c, err)
	}
	h.setSession(c, token)
	return h.meResponse(c, user)
}

func (h *Auth) me(c fiber.Ctx) error {
	user, err := requireUser(c)
	if err != nil {
		return err
	}
	return h.meResponse(c, *user)
}

func (h *Auth) updateProfile(c fiber.Ctx) error {
	user, err := requireUser(c)
	if err != nil {
		return err
	}
	var in dto.UpdateProfile
	if err := c.Bind().Body(&in); err != nil {
		return fiber.NewError(400, "Invalid request.")
	}
	updated, err := h.accounts.UpdateProfile(c.Context(), user.ID, model.User{DisplayName: in.DisplayName, Bio: in.Bio, Location: in.Location, Website: in.Website})
	if err != nil {
		return writeError(c, err)
	}
	return h.meResponse(c, updated)
}

func (h *Auth) setPassword(c fiber.Ctx) error {
	user, err := requireUser(c)
	if err != nil {
		return err
	}
	var in dto.Credentials
	if err := c.Bind().Body(&in); err != nil {
		return fiber.NewError(400, "Invalid request.")
	}
	if err := h.accounts.SetPassword(c.Context(), user.ID, in.Password); err != nil {
		return writeError(c, err)
	}
	return h.meResponse(c, *user)
}

func (h *Auth) removePassword(c fiber.Ctx) error {
	user, err := requireUser(c)
	if err != nil {
		return err
	}
	if err := h.accounts.RemovePassword(c.Context(), user.ID); err != nil {
		return writeError(c, err)
	}
	return h.meResponse(c, *user)
}

func (h *Auth) removePasskey(c fiber.Ctx) error {
	user, err := requireUser(c)
	if err != nil {
		return err
	}
	if err := h.accounts.RemovePasskey(c.Context(), user.ID, c.Params("id")); err != nil {
		return writeError(c, err)
	}
	return h.meResponse(c, *user)
}

func (h *Auth) recover(c fiber.Ctx) error {
	var in dto.RecoverAccount
	if err := c.Bind().Body(&in); err != nil {
		return fiber.NewError(400, "Invalid request.")
	}
	user, token, key, err := h.accounts.RecoverAccount(c.Context(), in.Username, in.RecoveryKey, in.NewPassword)
	if err != nil {
		return writeError(c, err)
	}
	h.setSession(c, token)
	return h.meResponseWithKey(c, user, key)
}

func (h *Auth) regenerateRecoveryKey(c fiber.Ctx) error {
	user, err := requireUser(c)
	if err != nil {
		return err
	}
	key, err := h.accounts.RegenerateRecoveryKey(c.Context(), user.ID)
	if err != nil {
		return writeError(c, err)
	}
	return h.meResponseWithKey(c, *user, key)
}

// RegisterMedia serves profile pictures on the given router.
func (h *Auth) RegisterMedia(app fiber.Router) {
	app.Get("/avatars/:id", h.avatar)
}

func (h *Auth) updatePreferences(c fiber.Ctx) error {
	user, err := requireUser(c)
	if err != nil {
		return err
	}
	var in dto.UpdatePreferences
	if err := c.Bind().Body(&in); err != nil {
		return fiber.NewError(400, "Invalid request.")
	}
	if err := h.accounts.SetShowNSFW(c.Context(), user.ID, in.ShowNSFW, in.Over18); err != nil {
		return writeError(c, err)
	}
	updated, err := h.accounts.GetUser(c.Context(), user.ID)
	if err != nil {
		return writeError(c, err)
	}
	return h.meResponse(c, updated)
}

func (h *Auth) updateTagPreferences(c fiber.Ctx) error {
	user, err := requireUser(c)
	if err != nil {
		return err
	}
	var in dto.UpdateTagPreferences
	if err := c.Bind().Body(&in); err != nil {
		return fiber.NewError(400, "Invalid request.")
	}
	if err := h.accounts.SetTagPreferences(c.Context(), user.ID, in.IncludeTags, in.ExcludeTags); err != nil {
		return writeError(c, err)
	}
	updated, err := h.accounts.GetUser(c.Context(), user.ID)
	if err != nil {
		return writeError(c, err)
	}
	return h.meResponse(c, updated)
}

func (h *Auth) updateAppearance(c fiber.Ctx) error {
	user, err := requireUser(c)
	if err != nil {
		return err
	}
	var in dto.UpdateAppearance
	if err := c.Bind().Body(&in); err != nil {
		return fiber.NewError(400, "Invalid request.")
	}
	if err := h.accounts.SetAppearance(c.Context(), user.ID, in.Theme, in.Font); err != nil {
		return writeError(c, err)
	}
	updated, err := h.accounts.GetUser(c.Context(), user.ID)
	if err != nil {
		return writeError(c, err)
	}
	return h.meResponse(c, updated)
}

func (h *Auth) avatar(c fiber.Ctx) error {
	user, file, err := h.accounts.Avatar(c.Context(), c.Params("id"))
	if err != nil {
		return fiber.ErrNotFound
	}
	c.Set("Content-Type", user.AvatarType)
	c.Set("Cache-Control", "public, max-age=3600")
	return c.SendStream(file)
}

const maxAvatarBytes = 8 << 20

func (h *Auth) setAvatar(c fiber.Ctx) error {
	user, err := requireUser(c)
	if err != nil {
		return err
	}
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return fiber.NewError(400, "Choose a picture to upload.")
	}
	if fileHeader.Size > maxAvatarBytes {
		return fiber.NewError(400, "That picture is too big — keep it under 8 MB.")
	}
	src, err := fileHeader.Open()
	if err != nil {
		return err
	}
	defer src.Close() //nolint:errcheck
	head := make([]byte, 512)
	n, _ := src.Read(head)
	contentType := board.SniffContentType(head[:n], fileHeader.Filename)
	if !strings.HasPrefix(contentType, "image/") {
		return fiber.NewError(400, "Profile pictures must be an image (JPEG, PNG, GIF, WebP or AVIF).")
	}
	if _, err := src.Seek(0, 0); err != nil {
		return err
	}
	if err := h.accounts.StoreAvatar(c.Context(), user.ID, src, contentType); err != nil {
		return writeError(c, err)
	}
	updated, err := h.accounts.GetUser(c.Context(), user.ID)
	if err != nil {
		return writeError(c, err)
	}
	return h.meResponse(c, updated)
}

func (h *Auth) removeAvatar(c fiber.Ctx) error {
	user, err := requireUser(c)
	if err != nil {
		return err
	}
	if err := h.accounts.RemoveAvatar(c.Context(), user.ID); err != nil {
		return writeError(c, err)
	}
	updated, err := h.accounts.GetUser(c.Context(), user.ID)
	if err != nil {
		return writeError(c, err)
	}
	return h.meResponse(c, updated)
}

func (h *Auth) publicProfile(c fiber.Ctx) error {
	user, err := h.accounts.GetByUsername(c.Context(), c.Params("username"))
	if err != nil {
		return writeError(c, err)
	}
	lists, err := h.collections.PublicLists(c.Context(), user.ID)
	if err != nil {
		return writeError(c, err)
	}
	var contributions int64
	if h.board != nil {
		contributions, _ = h.board.Contributions(c.Context(), user.ID)
	}
	prefs := preferencesDTO(user)
	return c.JSON(dto.Profile{User: userDTO(user), PublicLists: listsDTO(lists), Contributions: contributions, Theme: prefs.Theme, Font: prefs.Font})
}
