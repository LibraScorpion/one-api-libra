package model

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/logger"
)

// ModelPricing 模型定价
type ModelPricing struct {
	Id            int     `json:"id"`
	ModelName     string  `json:"model_name" gorm:"type:varchar(100);uniqueIndex"`
	DisplayName   string  `json:"display_name" gorm:"type:varchar(200)"`
	Provider      string  `json:"provider" gorm:"type:varchar(50);index:idx_provider"`
	Description   string  `json:"description" gorm:"type:text"`
	ContextLength int     `json:"context_length"`
	PricingInput  float64 `json:"pricing_input" gorm:"type:decimal(12,8)"` // 美元/Token
	PricingOutput float64 `json:"pricing_output" gorm:"type:decimal(12,8)"` // 美元/Token
	IsActive      bool    `json:"is_active" gorm:"default:true;index:idx_active"`
	CreatedAt     int64   `json:"created_at" gorm:"bigint"`
	UpdatedAt     int64   `json:"updated_at" gorm:"bigint"`
}

func (ModelPricing) TableName() string {
	return "model_pricing"
}

// 缓存相关
var (
	modelPricingCache     map[string]*ModelPricing
	modelPricingCacheTime time.Time
	modelPricingCacheTTL  = 5 * time.Minute
)

// InitModelPricingCache 初始化模型定价缓存
func InitModelPricingCache() error {
	var pricings []*ModelPricing
	err := DB.Where("is_active = ?", true).Find(&pricings).Error
	if err != nil {
		return err
	}

	cache := make(map[string]*ModelPricing)
	for _, pricing := range pricings {
		cache[pricing.ModelName] = pricing
	}

	modelPricingCache = cache
	modelPricingCacheTime = time.Now()
	logger.SysLog(fmt.Sprintf("loaded %d model pricings into cache", len(cache)))
	return nil
}

// GetModelPricing 获取模型定价（带缓存）
func GetModelPricing(modelName string) (*ModelPricing, error) {
	// 检查缓存是否过期
	if time.Since(modelPricingCacheTime) > modelPricingCacheTTL {
		_ = InitModelPricingCache()
	}

	// 从缓存获取
	if pricing, ok := modelPricingCache[modelName]; ok {
		return pricing, nil
	}

	// 缓存未命中，从数据库查询
	var pricing ModelPricing
	err := DB.Where("model_name = ? AND is_active = ?", modelName, true).First(&pricing).Error
	if err != nil {
		return nil, errors.New("model pricing not found: " + modelName)
	}

	// 更新缓存
	modelPricingCache[modelName] = &pricing
	return &pricing, nil
}

// GetAllModelPricings 获取所有模型定价
func GetAllModelPricings(provider string) ([]*ModelPricing, error) {
	var pricings []*ModelPricing
	query := DB.Where("is_active = ?", true)

	if provider != "" {
		query = query.Where("provider = ?", provider)
	}

	err := query.Order("display_name").Find(&pricings).Error
	return pricings, err
}

// CalculateTokenCost 计算 Token 成本
func CalculateTokenCost(modelName string, inputTokens int, outputTokens int) (float64, error) {
	pricing, err := GetModelPricing(modelName)
	if err != nil {
		// 如果找不到定价，使用默认价格（GPT-3.5 Turbo 的价格）
		logger.SysLog("model pricing not found for " + modelName + ", using default price")
		return float64(inputTokens)*0.0000015 + float64(outputTokens)*0.000002, nil
	}

	inputCost := float64(inputTokens) * pricing.PricingInput
	outputCost := float64(outputTokens) * pricing.PricingOutput
	totalCost := inputCost + outputCost

	return totalCost, nil
}

// CreateOrUpdateModelPricing 创建或更新模型定价
func CreateOrUpdateModelPricing(ctx context.Context, pricing *ModelPricing) error {
	pricing.UpdatedAt = GetTimestamp()

	var existing ModelPricing
	err := DB.Where("model_name = ?", pricing.ModelName).First(&existing).Error

	if err == nil {
		// 已存在，更新
		pricing.Id = existing.Id
		pricing.CreatedAt = existing.CreatedAt
		err = DB.Model(&ModelPricing{}).Where("id = ?", pricing.Id).Updates(pricing).Error
	} else {
		// 不存在，创建
		pricing.CreatedAt = GetTimestamp()
		err = DB.Create(pricing).Error
	}

	if err != nil {
		logger.Error(ctx, "failed to create/update model pricing: "+err.Error())
		return err
	}

	// 刷新缓存
	_ = InitModelPricingCache()
	return nil
}

// GetTimestamp 获取当前时间戳
func GetTimestamp() int64 {
	if common.UsingSQLite {
		return time.Now().Unix()
	}
	return time.Now().UnixNano() / int64(time.Millisecond)
}
