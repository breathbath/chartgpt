package auth

import (
	"breathbathChatGPT/pkg/storage"
	"context"
	"github.com/sirupsen/logrus"
)

func MigrateUsers(ctx context.Context, cfg *Config, db storage.Client) error {
	log := logrus.WithContext(ctx)
	log.Debug("Will migrate configured users to db")

	for _, u := range cfg.Users {
		cacheKey := GenerateUserCacheKey(u.PlatformName, u.Login)
		cachedUser := new(CachedUser)
		found, err := db.Load(ctx, cacheKey, cachedUser)
		if err != nil {
			return err
		}

		if found {
			continue
		}

		err = db.Save(ctx, cacheKey, CachedUser{
			Id:           u.Login,
			State:        UserUnverified,
			PlatformName: u.PlatformName,
			Role:         u.Role,
			PasswordHash: u.PasswordHash,
		}, 0)

		if err != nil {
			return err
		}

		log.Debugf("saved user %q to cache", u.Login)
	}

	return nil
}
