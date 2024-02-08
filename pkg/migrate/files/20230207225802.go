package files

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

var Migration20230207225802 = &gormigrate.Migration{
	ID: "20230207225802",
	Migrate: func(conn *gorm.DB) error {
		res := conn.Exec("UPDATE `wines` SET `body` = '' WHERE `body` LIKE '%мл.'")
		if res.Error != nil {
			return errors.WithStack(res.Error)
		}

		res = conn.Exec("UPDATE `wines` SET `recommend` = '' WHERE `recommend` LIKE '%минеральные%' or `recommend` LIKE '%фруктовые%' or `recommend` LIKE '%Элегантные%' or `recommend` LIKE '%Насыщенные%' or `recommend` LIKE '%Пряные%' or `recommend` LIKE '%Пряные%'or `recommend` LIKE '%освежающие%'")
		if res.Error != nil {
			return errors.WithStack(res.Error)
		}
		return nil
	},
	Rollback: func(tx *gorm.DB) error {
		return nil
	},
}
