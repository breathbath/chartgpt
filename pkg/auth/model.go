package auth

import (
	"breathbathChatGPT/pkg/errs"
	"fmt"
	"strings"
)

type UserState uint

const (
	UserUnverified UserState = iota
	UserVerified
)

type CachedUser struct {
	Id           string    `json:"id"`
	State        UserState `json:"state"`
	PlatformName string    `json:"platform"`
	Role         string    `json:"role"`
	PasswordHash string    `json:"password_hash"`
	LoginTill    int64     `json:"login_till"`
}

type ConfiguredUser struct {
	Role         string `json:"role"`
	Login        string `json:"login"`
	PlatformName string `json:"platform_name"`
	PasswordHash string `json:"password_hash"`
}

func (u ConfiguredUser) Validate() error {
	multiErr := errs.NewMulti()
	if u.Role == "" {
		multiErr.Err("role field cannot be empty in one of users in AUTH_USERS")
	}
	if u.Login == "" {
		multiErr.Err("user login field cannot be empty in one of users in AUTH_USERS")
	}
	if u.PasswordHash == "" {
		multiErr.Err("password_hash field cannot be empty in one of users in AUTH_USERS")
	}
	if u.PlatformName == "" {
		multiErr.Err("platform field cannot be empty in one of users in AUTH_USERS")
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func GenerateUserCacheKey(platform, login string) string {
	return fmt.Sprintf("users/%s/%s", strings.ToLower(platform), strings.ToLower(login))
}
