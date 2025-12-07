package model

import (
	"time"

	"github.com/songquanpeng/one-api/common/logger"
	"gorm.io/gorm"
)

// ChannelHealth 渠道健康状态
type ChannelHealth struct {
	ChannelID        int    `json:"channel_id" gorm:"primaryKey"`
	SuccessCount     int    `json:"success_count" gorm:"default:0"`
	FailureCount     int    `json:"failure_count" gorm:"default:0"`
	ConsecutiveFails int    `json:"consecutive_fails" gorm:"default:0"`
	LastSuccessAt    int64  `json:"last_success_at" gorm:"bigint"`
	LastFailureAt    int64  `json:"last_failure_at" gorm:"bigint"`
	AvgLatency       int    `json:"avg_latency" gorm:"default:0"` // 平均延迟（ms）
	Status           string `json:"status" gorm:"type:varchar(20);default:'unknown'"` // healthy/unhealthy/unknown
	UpdatedAt        int64  `json:"updated_at" gorm:"bigint"`
}

// GetChannelHealth 获取渠道健康状态
func GetChannelHealth(channelID int) (*ChannelHealth, error) {
	health := &ChannelHealth{ChannelID: channelID}
	err := DB.FirstOrCreate(health, ChannelHealth{ChannelID: channelID}).Error
	if err != nil {
		return nil, err
	}
	return health, nil
}

// GetAllChannelHealth 获取所有渠道健康状态
func GetAllChannelHealth() ([]*ChannelHealth, error) {
	var healths []*ChannelHealth
	err := DB.Find(&healths).Error
	return healths, err
}

// UpdateChannelHealth 更新渠道健康状态
func UpdateChannelHealth(health *ChannelHealth) error {
	health.UpdatedAt = time.Now().Unix()
	return DB.Save(health).Error
}

// OnRequestSuccess 请求成功时更新
func OnRequestSuccess(channelID int, latency int) error {
	health, err := GetChannelHealth(channelID)
	if err != nil {
		return err
	}

	health.SuccessCount++
	health.LastSuccessAt = time.Now().Unix()
	health.ConsecutiveFails = 0

	// 更新平均延迟（指数移动平均）
	alpha := 0.2
	if health.AvgLatency == 0 {
		health.AvgLatency = latency
	} else {
		health.AvgLatency = int(alpha*float64(latency) + (1-alpha)*float64(health.AvgLatency))
	}

	// 判断是否恢复健康
	if health.Status == "unhealthy" {
		totalRequests := health.SuccessCount + health.FailureCount
		if totalRequests > 0 {
			successRate := float64(health.SuccessCount) / float64(totalRequests)
			if successRate > 0.9 && health.ConsecutiveFails == 0 {
				health.Status = "healthy"
				logger.SysLog("Channel " + string(rune(channelID)) + " recovered to healthy")
			}
		}
	}

	return UpdateChannelHealth(health)
}

// OnRequestFailure 请求失败时更新
func OnRequestFailure(channelID int, errorCode string) error {
	health, err := GetChannelHealth(channelID)
	if err != nil {
		return err
	}

	health.FailureCount++
	health.LastFailureAt = time.Now().Unix()
	health.ConsecutiveFails++

	// 连续失败 3 次，标记为不健康
	if health.ConsecutiveFails >= 3 {
		health.Status = "unhealthy"
		logger.SysError("Channel " + string(rune(channelID)) + " marked as unhealthy due to consecutive failures")
	}

	// 失败率超过 10%，标记为不健康
	totalRequests := health.SuccessCount + health.FailureCount
	if totalRequests >= 10 {
		failureRate := float64(health.FailureCount) / float64(totalRequests)
		if failureRate > 0.1 {
			health.Status = "unhealthy"
			logger.SysError("Channel " + string(rune(channelID)) + " marked as unhealthy due to high failure rate")
		}
	}

	return UpdateChannelHealth(health)
}

// IsChannelHealthy 检查渠道是否健康
func IsChannelHealthy(channelID int) bool {
	health, err := GetChannelHealth(channelID)
	if err != nil {
		return false
	}
	return health.Status == "healthy" || health.Status == "unknown"
}

// ResetChannelHealth 重置渠道健康状态
func ResetChannelHealth(channelID int) error {
	return DB.Model(&ChannelHealth{}).Where("channel_id = ?", channelID).Updates(map[string]interface{}{
		"success_count":     0,
		"failure_count":     0,
		"consecutive_fails": 0,
		"status":            "unknown",
		"updated_at":        time.Now().Unix(),
	}).Error
}

// CleanupOldHealth 清理旧的健康记录
func CleanupOldHealth(daysAgo int) error {
	threshold := time.Now().AddDate(0, 0, -daysAgo).Unix()
	return DB.Where("updated_at < ?", threshold).Delete(&ChannelHealth{}).Error
}

// GetHealthyChannels 获取所有健康的渠道 ID
func GetHealthyChannels() ([]int, error) {
	var channelIDs []int
	err := DB.Model(&ChannelHealth{}).
		Where("status = ? OR status = ?", "healthy", "unknown").
		Pluck("channel_id", &channelIDs).Error
	return channelIDs, err
}

// BatchUpdateChannelHealth 批量更新渠道健康状态
func BatchUpdateChannelHealth(healths []*ChannelHealth) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		for _, health := range healths {
			health.UpdatedAt = time.Now().Unix()
			if err := tx.Save(health).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
