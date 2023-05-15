package auth

import (
	"encoding/json"
	"fmt"
	"time"

	"breathbathChatGPT/pkg/errs"
)

type UserState uint

const (
	UserUnverified UserState = iota
	UserVerified
)

const (
	AdminRole = "admin"
	UserRole  = "user"
)

func (us UserState) String() string {
	switch us {
	case UserVerified:
		return "verified"
	case UserUnverified:
		return "unverified"
	default:
		return ""
	}
}

type CachedUser struct {
	Uid          string    `json:"uid"`
	Login        string    `json:"login"`
	State        UserState `json:"state"`
	PlatformName string    `json:"platform"`
	Role         string    `json:"role"`
	PasswordHash string    `json:"password_hash"`
	LoginTill    int64     `json:"login_till"`
}

func (cu *CachedUser) String() string {
	var loginTillP *time.Time

	if cu.LoginTill > 0 {
		loginTill := time.Unix(cu.LoginTill, 0)
		loginTillP = &loginTill
	}

	res := struct {
		Uid          string     `json:"uid"`
		Login        string     `json:"login"`
		State        string     `json:"state"`
		PlatformName string     `json:"platform"`
		Role         string     `json:"role"`
		LoginTill    *time.Time `json:"login_till"`
	}{
		Uid:          cu.Uid,
		Login:        cu.Login,
		State:        cu.State.String(),
		PlatformName: cu.PlatformName,
		Role:         cu.Role,
		LoginTill:    loginTillP,
	}

	userRaw, err := json.Marshal(res)
	if err != nil {
		return fmt.Sprintf("%+v", res)
	}

	return string(userRaw)
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
