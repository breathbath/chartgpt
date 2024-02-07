package migrate

import (
	"breathbathChatGPT/pkg/monitoring"
	"breathbathChatGPT/pkg/recommend"
	"gorm.io/gorm"
)

func Execute(dbConn *gorm.DB) error {
	err := schema(dbConn)
	if err != nil {
		return err
	}

	err = data(dbConn)
	if err != nil {
		return err
	}

	return nil
}

func schema(dbConn *gorm.DB) error {
	err := dbConn.AutoMigrate(recommend.Wine{}, monitoring.UsageStats{}, monitoring.Recommendation{})
	if err != nil {
		return err
	}

	return nil
}
