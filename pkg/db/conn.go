package db

import (
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func NewConn() (*gorm.DB, error) {
	config, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	e := config.Validate()
	if e.HasErrors() {
		return nil, err
	}

	connStr := config.ConnString + "?parseTime=true"

	conn, err := gorm.Open(mysql.Open(connStr), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, err
	}

	return conn, nil
}
