package files

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

var Migration20240229075800 = &gormigrate.Migration{
	ID: "20240229075800",
	Migrate: func(conn *gorm.DB) error {
		res := conn.Exec("DELETE from `likes` WHERE user_login = ''")
		if res.Error != nil {
			logrus.Error(res.Error)
		}

		return nil
	},
	Rollback: func(tx *gorm.DB) error {
		return nil
	},
}
