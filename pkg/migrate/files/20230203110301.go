package files

import (
	_ "embed"
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

var Migration20230203110301 = &gormigrate.Migration{
	ID: "20230203110301",
	Migrate: func(conn *gorm.DB) error {
		return conn.Migrator().DropTable("winechef_wines")
	},
	Rollback: func(tx *gorm.DB) error {
		return nil
	},
}
