package files

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

var Migration20240217122600 = &gormigrate.Migration{
	ID: "20240217122600",
	Migrate: func(conn *gorm.DB) error {
		res := conn.Exec("ALTER TABLE `wines` ADD FULLTEXT INDEX (`smell_description`, `taste_description`)")
		if res.Error != nil {
			return errors.WithStack(res.Error)
		}

		return nil
	},
	Rollback: func(tx *gorm.DB) error {
		return nil
	},
}
