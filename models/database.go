package models

import (
	"os"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

func GetDB() *gorm.DB {
	var err error = nil

	if DB != nil {
		sqlDB, err := DB.DB()
		if err == nil {
			err = sqlDB.Ping()
			if err == nil {
				return DB
			}
		}
	}

	for i := 0; i <= 3; i++ {
		DB, err = gorm.Open(mysql.Open(os.Getenv("MYSQL_DSN")), &gorm.Config{})
		if err == nil {
			sqlDB, err := DB.DB()
			if err == nil {
				sqlDB.SetMaxIdleConns(5)
				sqlDB.SetMaxOpenConns(10)
				sqlDB.SetConnMaxLifetime(time.Hour)
				return DB
			}
		}

		time.Sleep(100 * time.Millisecond)
	}

	return DB
}
