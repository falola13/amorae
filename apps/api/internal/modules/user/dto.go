package user

import "time"

// DTO deliberately has no field that could hold a password hash.
type DTO struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	Timezone    string    `json:"timezone"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func ToDTO(u User) DTO {
	return DTO{
		ID:          u.ID.String(),
		Email:       u.Email,
		DisplayName: u.DisplayName,
		Timezone:    u.Timezone,
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
	}
}
