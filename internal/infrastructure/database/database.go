// Package database wires Blog Service persistence using the shared GORM helper.
// Schema changes are managed by Atlas — never call AutoMigrate here.
package database

import (
	"context"

	"gorm.io/gorm"

	shareddb "github.com/marketing-digest/pkg/database"

	"github.com/marketing-digest/blog-service/internal/config"
)

func Open(cfg config.Config) (*gorm.DB, error) {
	return shareddb.Open(shareddb.Config{
		Host:            cfg.DatabaseHost,
		Port:            cfg.DatabasePort,
		Name:            cfg.DatabaseName,
		User:            cfg.DatabaseUser,
		Password:        cfg.DatabasePassword,
		SSLMode:         cfg.DatabaseSSLMode,
		MaxOpenConns:    cfg.DBMaxOpenConns,
		MaxIdleConns:    cfg.DBMaxIdleConns,
		ConnMaxLifetime: cfg.DBConnMaxLifetime,
	})
}

func Ping(ctx context.Context, db *gorm.DB) error { return shareddb.Ping(ctx, db) }
func Close(db *gorm.DB) error                     { return shareddb.Close(db) }
