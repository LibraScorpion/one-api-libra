package router

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hashicorp/golang-lru/v2"
	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/model"
)

// ChannelCache 渠道缓存
type ChannelCache struct {
	local *lru.Cache[string, []*model.Channel] // 本地 LRU 缓存
}

// NewChannelCache 创建渠道缓存
func NewChannelCache() *ChannelCache {
	// 创建 LRU 缓存，容量 1000
	cache, err := lru.New[string, []*model.Channel](1000)
	if err != nil {
		logger.SysError(fmt.Sprintf("Failed to create LRU cache: %v", err))
		panic(err)
	}

	return &ChannelCache{
		local: cache,
	}
}

// GetChannels 获取渠道列表
func (c *ChannelCache) GetChannels(ctx context.Context, group string, modelName string) ([]*model.Channel, error) {
	key := fmt.Sprintf("%s:%s", group, modelName)

	// 1. 尝试本地缓存
	if channels, ok := c.local.Get(key); ok {
		logger.Debugf(ctx, "Hit local cache for key=%s", key)
		return channels, nil
	}

	// 2. 尝试 Redis（如果启用）
	if common.RedisEnabled {
		redisKey := "router:channels:" + key
		data, err := common.RDB.Get(ctx, redisKey).Result()
		if err == nil {
			var channels []*model.Channel
			if err := json.Unmarshal([]byte(data), &channels); err == nil {
				// 写入本地缓存
				c.local.Add(key, channels)
				logger.Debugf(ctx, "Hit Redis cache for key=%s", key)
				return channels, nil
			}
		}
	}

	// 3. 查询数据库
	logger.Debugf(ctx, "Cache miss, querying database for key=%s", key)
	channels, err := model.GetSatisfiedChannels(group, modelName)
	if err != nil {
		return nil, err
	}

	// 4. 写入缓存
	c.writeToCache(ctx, key, channels)

	return channels, nil
}

// writeToCache 写入缓存
func (c *ChannelCache) writeToCache(ctx context.Context, key string, channels []*model.Channel) {
	// 写入本地缓存
	c.local.Add(key, channels)

	// 写入 Redis（如果启用）
	if common.RedisEnabled {
		redisKey := "router:channels:" + key
		data, err := json.Marshal(channels)
		if err == nil {
			err = common.RDB.Set(ctx, redisKey, data, 60*time.Second).Err()
			if err != nil {
				logger.SysError(fmt.Sprintf("Failed to write to Redis cache: %v", err))
			}
		}
	}
}

// Invalidate 使缓存失效
func (c *ChannelCache) Invalidate(ctx context.Context, channelID int) {
	logger.SysLog(fmt.Sprintf("Invalidating cache for channel %d", channelID))

	// 清空本地缓存（简单粗暴，但有效）
	c.local.Purge()

	// 删除 Redis 中相关的 key（如果启用）
	if common.RedisEnabled {
		// 使用通配符删除所有相关的 key
		pattern := "router:channels:*"
		iter := common.RDB.Scan(ctx, 0, pattern, 0).Iterator()
		for iter.Next(ctx) {
			err := common.RDB.Del(ctx, iter.Val()).Err()
			if err != nil {
				logger.SysError(fmt.Sprintf("Failed to delete Redis key %s: %v", iter.Val(), err))
			}
		}
		if err := iter.Err(); err != nil {
			logger.SysError(fmt.Sprintf("Redis scan error: %v", err))
		}
	}
}

// InvalidateAll 清空所有缓存
func (c *ChannelCache) InvalidateAll(ctx context.Context) {
	logger.SysLog("Invalidating all router cache")
	c.local.Purge()

	if common.RedisEnabled {
		pattern := "router:channels:*"
		iter := common.RDB.Scan(ctx, 0, pattern, 0).Iterator()
		for iter.Next(ctx) {
			common.RDB.Del(ctx, iter.Val())
		}
	}
}

// Preload 预加载缓存
func (c *ChannelCache) Preload(ctx context.Context) error {
	logger.SysLog("Preloading router cache")

	// 获取所有分组
	groups := []string{"default"} // TODO: 从配置或数据库获取所有分组

	// 获取所有模型
	for _, group := range groups {
		models, err := model.GetGroupModels(ctx, group)
		if err != nil {
			logger.SysError(fmt.Sprintf("Failed to get models for group %s: %v", group, err))
			continue
		}

		// 预加载每个 group-model 组合
		for _, m := range models {
			_, err := c.GetChannels(ctx, group, m)
			if err != nil {
				logger.SysError(fmt.Sprintf("Failed to preload channels for group=%s, model=%s: %v", group, m, err))
			}
		}
	}

	logger.SysLog("Router cache preloaded successfully")
	return nil
}

// GetStats 获取缓存统计
func (c *ChannelCache) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"local_cache_len": c.local.Len(),
		"local_cache_cap": 1000,
	}
}
