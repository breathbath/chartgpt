package auth

import (
	"context"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

func MigrateUsers(ctx context.Context, cfg *Config, us *UserStorage) error {
	log := logrus.WithContext(ctx)
	log.Debug("Will migrate configured users to cache")

	for _, u := range cfg.Users {
		u := &CachedUser{
			UID:          uuid.NewString(),
			Login:        u.Login,
			State:        UserVerified,
			PlatformName: u.PlatformName,
			Role:         u.Role,
			PasswordHash: u.PasswordHash,
		}
		err := us.WriteUserToStorage(ctx, u)

		if err != nil {
			return err
		}

		log.Debugf("saved user %q to cache", u.Login)
	}

	return nil
}
