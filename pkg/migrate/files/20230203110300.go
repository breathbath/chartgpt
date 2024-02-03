package files

import (
	_ "embed"
	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

//go:embed 20230203110300-1.sql
var file20230203110300_1 string

//go:embed 20230203110300-2.sql
var file20230203110300_2 string

//go:embed 20230203110300-3.sql
var file20230203110300_3 string

//go:embed 20230203110300-4.sql
var file20230203110300_4 string

//go:embed 20230203110300-5.sql
var file20230203110300_5 string

var Migration20230203110300 = &gormigrate.Migration{
	ID: "20230203110300",
	Migrate: func(conn *gorm.DB) error {
		for _, mig := range []string{file20230203110300_1, file20230203110300_2, file20230203110300_3, file20230203110300_4, file20230203110300_5} {
			res := conn.Exec(mig)
			if res.Error != nil {
				return errors.Wrapf(res.Error, "failed to execute %s", mig)
			}
		}
		return nil
	},
	Rollback: func(tx *gorm.DB) error {
		return nil
	},
}
