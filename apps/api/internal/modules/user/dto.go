package user

import "time"

// birthdayDTO is null on the wire when a profile has none.
type birthdayDTO struct {
	Month int  `json:"month"`
	Day   int  `json:"day"`
	Year  *int `json:"year"`
}

// DTO deliberately has no field that could hold a password hash.
type DTO struct {
	ID          string       `json:"id"`
	Email       string       `json:"email"`
	DisplayName string       `json:"display_name"`
	Timezone    string       `json:"timezone"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
	Birthday    *birthdayDTO `json:"birthday"`
	// Signed, unguessable, generated per request; absent when there is no
	// photo. Mirrors memories' memoryDTO.PhotoURL.
	PhotoURL string `json:"photo_url,omitempty"`
}

// ToDTO renders a User without a photo_url — callers that have no Service
// at hand (auth's AuthResult, the data export) never claimed a photo, so
// they never had one to lose. ToDTOWithPhoto is what /users/me and its
// writes use instead.
func ToDTO(u User) DTO {
	out := DTO{
		ID:          u.ID.String(),
		Email:       u.Email,
		DisplayName: u.DisplayName,
		Timezone:    u.Timezone,
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
	}
	if u.BirthMonth != nil && u.BirthDay != nil {
		out.Birthday = &birthdayDTO{Month: *u.BirthMonth, Day: *u.BirthDay, Year: u.BirthYear}
	}
	return out
}

// ToDTOWithPhoto is ToDTO plus the caller-supplied photo_url (Service.PhotoURL) — separate
// so ToDTO itself needs nothing beyond a User.
func ToDTOWithPhoto(u User, photoURL string) DTO {
	out := ToDTO(u)
	out.PhotoURL = photoURL
	return out
}
