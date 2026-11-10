package board

import (
	"context"
	"errors"
	"time"
)

const (
	uploadQuotaPerHour = 20
	uploadQuotaPerDay  = 100
)

// ErrUploadLimit is returned when an account is already at its rolling
// upload cap. The message is safe to show to the member.
var ErrUploadLimit = errors.New("Upload limit reached. Try again later.")

// CheckUploadQuota reports whether ownerID may add another post.
// Twenty successful posts per rolling hour and one hundred per rolling
// day. Hidden and soft-deleted posts count, so deleting one does not
// reset the cap. Staff are not exempt. Seed and demo do not call this.
func (s *Service) CheckUploadQuota(ctx context.Context, ownerID string) error {
	hour, err := s.repo.CountPostsByOwnerSince(ctx, ownerID, time.Hour)
	if err != nil {
		return err
	}
	if hour >= uploadQuotaPerHour {
		return ErrUploadLimit
	}
	day, err := s.repo.CountPostsByOwnerSince(ctx, ownerID, 24*time.Hour)
	if err != nil {
		return err
	}
	if day >= uploadQuotaPerDay {
		return ErrUploadLimit
	}
	return nil
}
