package auth

import "breathbathChartGPT/pkg/errs"

type UserState uint

const (
	UserUnverified UserState = iota
	UserReadyToBeVerified
	UserVerified
)

type User struct {
	Role         string    `json:"role"`
	UserID       string    `json:"user_id"`
	PasswordHash string    `json:"password_hash"`
	State        UserState `json:"state"`
}

func (u User) ValidateAsConfig() error {
	multiErr := errs.NewMulti()
	if u.Role == "" {
		multiErr.Err("role field cannot be empty in one of users in AUTH_USERS")
	}
	if u.UserID == "" {
		multiErr.Err("user_id field cannot be empty in one of users in AUTH_USERS")
	}
	if u.PasswordHash == "" {
		multiErr.Err("password_hash field cannot be empty in one of users in AUTH_USERS")
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}
