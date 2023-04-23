package auth

import "breathbathChartGPT/pkg/errs"

type UserState uint

const (
	UserUnverified UserState = iota
	UserVerified
)

type CachedUser struct {
	Id       string    `json:"id"`
	State    UserState `json:"state"`
	Platform string    `json:"platform"`
}

type ConfiguredUser struct {
	Role         string   `json:"role"`
	UserIDs      []string `json:"user_ids"`
	PasswordHash string   `json:"password_hash"`
}

func (u ConfiguredUser) Validate() error {
	multiErr := errs.NewMulti()
	if u.Role == "" {
		multiErr.Err("role field cannot be empty in one of users in AUTH_USERS")
	}
	if len(u.UserIDs) == 0 {
		multiErr.Err("user_ids field cannot be empty in one of users in AUTH_USERS")
	}
	if u.PasswordHash == "" {
		multiErr.Err("password_hash field cannot be empty in one of users in AUTH_USERS")
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}
