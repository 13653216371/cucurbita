package storage

import (
	"os"
	"path"

	"github.com/glebarez/sqlite"
	"github.com/lanthora/cucurbita/logger"
	"github.com/lanthora/cucurbita/settings"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

var db *gorm.DB

func init() {
	err := os.MkdirAll(settings.Storage, os.ModeDir|os.ModePerm)
	if err != nil {
		logger.Fatal(err)
	}
	db, err = gorm.Open(sqlite.Open(path.Join(settings.Storage, "sqlite.db")), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		logger.Fatal(err)
	}

	if err := AutoMigrate(Config{}); err != nil {
		logger.Fatal(err)
	}
}

func AutoMigrate(dst ...interface{}) error {
	return db.AutoMigrate(dst...)
}

func Raw(sql string, values ...interface{}) (tx *gorm.DB) {
	return db.Raw(sql, values)
}

func Create(value interface{}) (tx *gorm.DB) {
	return db.Create(value)

}

func Delete(value interface{}, conds ...interface{}) (tx *gorm.DB) {
	return db.Delete(value, conds...)
}

func Updates(value interface{}) (tx *gorm.DB) {
	return db.Updates(value)
}

func Save(value interface{}) (tx *gorm.DB) {
	return db.Save(value)
}

func Model(value interface{}) (tx *gorm.DB) {
	return db.Model(value)
}

func Where(query interface{}, args ...interface{}) (tx *gorm.DB) {
	return db.Where(query, args...)
}

func Find(dest interface{}, conds ...interface{}) (tx *gorm.DB) {
	return db.Find(dest, conds...)
}

type Config struct {
	Key   string `gorm:"primaryKey"`
	Value string
}
