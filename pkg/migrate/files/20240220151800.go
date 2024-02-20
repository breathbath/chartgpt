package files

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

var Migration20240220151800 = &gormigrate.Migration{
	ID: "20240220151800",
	Migrate: func(conn *gorm.DB) error {
		res := conn.Exec("ALTER TABLE `likes` DROP FOREIGN KEY `fk_likes_recommendation`")
		if res.Error != nil {
			logrus.Error(res.Error)
		}
		res2 := conn.Exec("ALTER TABLE `likes` DROP INDEX `fk_likes_recommendation`")
		if res2.Error != nil {
			logrus.Error(res2.Error)
		}
		res3 := conn.Exec("ALTER TABLE `likes` DROP `recommendation_id`")
		if res3.Error != nil {
			return errors.WithStack(res3.Error)
		}

		return nil
	},
	Rollback: func(tx *gorm.DB) error {
		return nil
	},
}
