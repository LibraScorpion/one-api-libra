package migrations

import (
	"github.com/songquanpeng/one-api/model"
)

// MigrateChannelHealth 迁移渠道健康表
func MigrateChannelHealth() error {
	// 自动迁移 ChannelHealth 表
	return model.DB.AutoMigrate(&model.ChannelHealth{})
}

// MigrateChannelEnhancements 迁移渠道增强字段
func MigrateChannelEnhancements() error {
	// 为 Channel 表添加新字段
	// 注意：这些字段可能已经存在，AutoMigrate 会自动处理
	return model.DB.AutoMigrate(&model.Channel{})
}
