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
}

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
