package agentcli

import "github.com/astra-chanUwU/kura-v4/internal/model"

// SeesNSFW matches the HTTP gate: only a signed-in account that opted in.
func SeesNSFW(actor *model.User) bool {
	return actor != nil && actor.ShowNSFW
}

// CanSee matches the HTTP post gate. Hidden posts are staff or owner only.
// NSFW posts additionally require an opt-in. A miss is reported as not
// found so a hidden post is not confirmed to exist.
func CanSee(actor *model.User, p model.Post) bool {
	if p.Rating == "nsfw" && !SeesNSFW(actor) {
		return false
	}
	if !p.Hidden {
		return true
	}
	if actor == nil {
		return false
	}
	if actor.Role.AtLeast(model.RoleModerator) {
		return true
	}
	return p.OwnerID != nil && *p.OwnerID == actor.ID
}
