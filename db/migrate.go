package db

import (
	"gorm.io/gorm"

	"github.com/chihqiang/infra-go/logger"

	"chihqiang/dskpanel/model"
)

// Migrate 自动迁移所有实体。
func Migrate(g *gorm.DB) error {
	if err := g.AutoMigrate(
		&model.NodeMetric{},
		&model.PodMetric{},
	); err != nil {
		logger.Errorf("db migrate failed: %v", err)
		return err
	}
	logger.Info("db migrate done")
	return nil
}
