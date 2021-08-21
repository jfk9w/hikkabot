package gorm

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DefaultConfig = &gorm.Config{Logger: LogrusLogger}

func NewPostgres(conn string) (*gorm.DB, error) {
	return gorm.Open(postgres.Open(conn), DefaultConfig)
}
