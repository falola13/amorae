package user

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
)

// PhotoDestroyer is the one thing DeletingRepository needs of photos.Store —
// declared here rather than reusing Photos so a caller that only has
// deletion to offer doesn't have to fake Ticket and URL too.
type PhotoDestroyer interface {
	Destroy(ctx context.Context, publicID string) error
}

// DeletingRepository wraps PostgresRepository so deleting an account also
// destroys the person's profile photo at Cloudinary. Best-effort: a
// Cloudinary hiccup must never block deleting the account, since the row
// being removed is the only record of the consent to keep it. Exists so
// auth.Service — which only knows PostgresRepository through its own
// UserRepository interface (Create/GetByEmail/GetByID/SetLastLoginAt/
// UpdateEmail/UpdatePasswordHash/DeleteMe) — can gain this without the auth
// module importing photos at all.
type DeletingRepository struct {
	*PostgresRepository
	// nil when Cloudinary isn't configured — the account still deletes.
	photos PhotoDestroyer
	log    *slog.Logger
}

func NewDeletingRepository(repo *PostgresRepository, photos PhotoDestroyer, log *slog.Logger) *DeletingRepository {
	return &DeletingRepository{PostgresRepository: repo, photos: photos, log: log}
}

// DeleteMe destroys the photo first, then defers to PostgresRepository for
// the cascade (sessions, memberships, invitations). A failed destroy is
// logged, not returned — the person asked for their account gone, and a
// leaked Cloudinary asset is a smaller wrong than refusing that.
func (r *DeletingRepository) DeleteMe(ctx context.Context, id uuid.UUID) error {
	if r.photos != nil {
		if u, err := r.GetByID(ctx, id); err == nil && u.HasPhoto() {
			if err := r.photos.Destroy(ctx, u.PhotoID); err != nil {
				r.log.Warn("destroying profile photo on account deletion", "error", err, "user_id", id)
			}
		}
	}
	return r.PostgresRepository.DeleteMe(ctx, id)
}
