// Package storage owns the golangci dashboard's GORM models and its
// dedicated MySQL connection.
package storage

import (
	"fmt"
	"os"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const (
	maxOpenConns    = 20
	maxIdleConns    = 5
	connMaxLifetime = 30 * time.Minute
)

// dsn builds a MySQL DSN from the GOLANGCI_MYSQL_DB_* env vars. This is a
// self-contained connection, independent of common.OPADAPTER — see
// golangci/plans/2026-08-04-golangci-m1-implementation.md for why: OPADAPTER
// is a closed set of RT-ticket-domain repositories and has no path to add
// unrelated tables.
func dsn() string {
	host := os.Getenv("GOLANGCI_MYSQL_DB_HOST")
	port := os.Getenv("GOLANGCI_MYSQL_DB_PORT")
	user := os.Getenv("GOLANGCI_MYSQL_DB_USERNAME")
	pass := os.Getenv("GOLANGCI_MYSQL_DB_PASSWORD")
	db := os.Getenv("GOLANGCI_MYSQL_DB_DATABASE")
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		user,
		pass,
		host,
		port,
		db,
	)
}

// Connect opens the golangci dashboard's own GORM/MySQL connection, with
// bounded pool limits — an unbounded pool can exhaust MySQL's own
// max_connections under load.
func Connect() (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(dsn()), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get underlying sql.DB: %w", err)
	}
	sqlDB.SetMaxOpenConns(20)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)

	return db, nil
}

// Migrate auto-migrates every model this package owns. No migration
// tooling exists elsewhere in this repo, so AutoMigrate is the default
// for this new, isolated schema.
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(AllModels()...)
}
