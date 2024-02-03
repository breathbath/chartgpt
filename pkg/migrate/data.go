package migrate

import (
	"breathbathChatGPT/pkg/migrate/files"
	"github.com/go-gormigrate/gormigrate/v2"
	"gopkg.in/errgo.v2/errors"
	"gorm.io/gorm"
)

var migrations = []*gormigrate.Migration{
	files.Migration20230203110300,
	files.Migration20230203110301,
}

func data(conn *gorm.DB) error {
	m := gormigrate.New(conn, gormigrate.DefaultOptions, migrations)

	if err := m.Migrate(); err != nil {
		return errors.Wrap(err)
	}

	return nil
}
