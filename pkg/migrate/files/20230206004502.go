package files

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

var Migration20230206004502 = &gormigrate.Migration{
	ID: "20230206004502",
	Migrate: func(conn *gorm.DB) error {
		res := conn.Exec("ALTER TABLE `wines` DROP `strength`")
		if res.Error != nil {
			return errors.WithStack(res.Error)
		}
		return nil
	},
	Rollback: func(tx *gorm.DB) error {
		return nil
	},
}
