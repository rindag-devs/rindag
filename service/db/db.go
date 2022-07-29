package db

import (
	"context"
	"fmt"

	"rindag/model"
	"rindag/service/etc"

	"github.com/go-redis/redis/v9"
	log "github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	PDB *gorm.DB
	RDB *redis.Client
)

func getDSNFromConfig(c *etc.Configuration) string {
	conf := c.Database.Postgres
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s",
		conf.Host, conf.Port, conf.User, conf.Password, conf.DBName)
	if !conf.UseSSL {
		dsn += " sslmode=disable"
	}
	return dsn
}

func setupPostgres() {
	dsn := getDSNFromConfig(etc.Config)
	var err error
	PDB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.WithError(err).Fatal("Postgres connection failed")
	}
	err = PDB.AutoMigrate(&model.User{})
	if err != nil {
		log.WithError(err).Fatal("Postgres migration failed")
	}
	log.Info("Postgres connected")
}

func setupRedis() {
	conf := etc.Config.Database.Redis
	RDB = redis.NewClient(&redis.Options{
		Addr:     conf.Host,
		Password: conf.Password,
		DB:       conf.DB,
	})
	if _, err := RDB.Ping(context.TODO()).Result(); err != nil {
		log.WithError(err).Fatal("Redis connection failed")
	}
}

func init() {
	setupPostgres()
	setupRedis()
}
