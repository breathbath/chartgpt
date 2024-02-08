package files

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

var Migration20230208092100 = &gormigrate.Migration{
	ID: "20230208092100",
	Migrate: func(conn *gorm.DB) error {
		res := conn.Exec("ALTER TABLE `wines` ADD FULLTEXT INDEX (`name`, `real_name`)")
		if res.Error != nil {
			return errors.WithStack(res.Error)
		}

		res2 := conn.Exec("ALTER TABLE `wines` ADD FULLTEXT INDEX (`recommend`)")
		if res2.Error != nil {
			return errors.WithStack(res2.Error)
		}

		return nil
	},
	Rollback: func(tx *gorm.DB) error {
		return nil
	},
}
