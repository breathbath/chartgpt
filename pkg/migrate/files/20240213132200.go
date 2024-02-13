package files

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

var Migration20240213132200 = &gormigrate.Migration{
	ID: "20240213132200",
	Migrate: func(conn *gorm.DB) error {
		res := conn.Exec("ALTER TABLE `recommendations` DROP `likes_count`")
		if res.Error != nil {
			return errors.WithStack(res.Error)
		}

		return nil
	},
	Rollback: func(tx *gorm.DB) error {
		return nil
	},
}
