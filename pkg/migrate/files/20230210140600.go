package files

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

var Migration20230210140600 = &gormigrate.Migration{
	ID: "20230210140600",
	Migrate: func(conn *gorm.DB) error {
		res := conn.Exec("ALTER TABLE `usage_stats` DROP `error`,DROP `first_name`, DROP `gen_completion_tokens`, DROP `gen_prompt_tokens`, DROP `is_voice_input`, DROP `last_name`, DROP `voice_to_text_model`")
		if res.Error != nil {
			return errors.WithStack(res.Error)
		}

		return nil
	},
	Rollback: func(tx *gorm.DB) error {
		return nil
	},
}
