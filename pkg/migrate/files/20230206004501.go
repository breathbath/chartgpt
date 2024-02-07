package files

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

var Migration20230206004501 = &gormigrate.Migration{
	ID: "20230206004501",
	Migrate: func(conn *gorm.DB) error {
		res := conn.Exec("UPDATE wines SET alcohol_percentage = CAST(REPLACE(`strength`, '%', '') AS DOUBLE)")
		if res.Error != nil {
			return errors.WithStack(res.Error)
		}
		return nil
	},
	Rollback: func(tx *gorm.DB) error {
		return nil
	},
}
