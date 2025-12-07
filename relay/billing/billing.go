package billing

import (
	"context"
	"fmt"

	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/model"
)

func ReturnPreConsumedQuota(ctx context.Context, preConsumedQuota int64, tokenId int) {
	if preConsumedQuota != 0 {
		go func(ctx context.Context) {
			// return pre-consumed quota
			err := model.PostConsumeTokenQuota(tokenId, -preConsumedQuota)
			if err != nil {
				logger.Error(ctx, "error return pre-consumed quota: "+err.Error())
			}
		}(ctx)
	}
}

func PostConsumeQuota(ctx context.Context, tokenId int, quotaDelta int64, totalQuota int64, userId int, channelId int, modelRatio float64, groupRatio float64, modelName string, tokenName string) {
	// quotaDelta is remaining quota to be consumed
	err := model.PostConsumeTokenQuota(tokenId, quotaDelta)
	if err != nil {
		logger.SysError("error consuming token remain quota: " + err.Error())
	}
	err = model.CacheUpdateUserQuota(ctx, userId)
	if err != nil {
		logger.SysError("error update user quota cache: " + err.Error())
	}
	// totalQuota is total quota consumed
	if totalQuota != 0 {
		logContent := fmt.Sprintf("倍率：%.2f × %.2f", modelRatio, groupRatio)

		// 计算实际成本（美元）
		// 注意：这里的 totalQuota 通常已经是经过倍率计算后的值
		// 我们需要 Token 数量来计算实际成本
		// 暂时使用 quota 除以倍率来估算 Token 数量
		estimatedTokens := int(float64(totalQuota) / (modelRatio * groupRatio))
		costUSD, costErr := model.CalculateTokenCost(modelName, estimatedTokens, 0)
		if costErr != nil {
			logger.Warn(ctx, "failed to calculate token cost: "+costErr.Error())
			costUSD = 0
		}

		model.RecordConsumeLog(ctx, &model.Log{
			UserId:           userId,
			ChannelId:        channelId,
			PromptTokens:     int(totalQuota),
			CompletionTokens: 0,
			ModelName:        modelName,
			TokenName:        tokenName,
			Quota:            int(totalQuota),
			Content:          logContent,
		})
		model.UpdateUserUsedQuotaAndRequestCount(userId, totalQuota)
		model.UpdateChannelUsedQuota(channelId, totalQuota)

		// 记录成本到余额交易记录
		if costUSD > 0 {
			err = model.RecordUsageCost(ctx, userId, costUSD, 0) // logId 暂时为 0
			if err != nil {
				logger.Error(ctx, "failed to record usage cost: "+err.Error())
			}
		}
	}
	if totalQuota <= 0 {
		logger.Error(ctx, fmt.Sprintf("totalQuota consumed is %d, something is wrong", totalQuota))
	}
}

// PostConsumeQuotaWithTokens 支持精确 Token 计数的扣费函数
func PostConsumeQuotaWithTokens(ctx context.Context, tokenId int, quotaDelta int64, totalQuota int64, userId int, channelId int, modelRatio float64, groupRatio float64, modelName string, tokenName string, promptTokens int, completionTokens int) {
	// quotaDelta is remaining quota to be consumed
	err := model.PostConsumeTokenQuota(tokenId, quotaDelta)
	if err != nil {
		logger.SysError("error consuming token remain quota: " + err.Error())
	}
	err = model.CacheUpdateUserQuota(ctx, userId)
	if err != nil {
		logger.SysError("error update user quota cache: " + err.Error())
	}

	// 计算实际成本（美元）
	var costUSD float64
	if promptTokens > 0 || completionTokens > 0 {
		costUSD, err = model.CalculateTokenCost(modelName, promptTokens, completionTokens)
		if err != nil {
			logger.Warn(ctx, "failed to calculate token cost: "+err.Error())
			costUSD = 0
		}
	}

	// totalQuota is total quota consumed
	if totalQuota != 0 {
		logContent := fmt.Sprintf("倍率：%.2f × %.2f | 成本：$%.6f", modelRatio, groupRatio, costUSD)

		model.RecordConsumeLog(ctx, &model.Log{
			UserId:           userId,
			ChannelId:        channelId,
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			ModelName:        modelName,
			TokenName:        tokenName,
			Quota:            int(totalQuota),
			Content:          logContent,
		})
		model.UpdateUserUsedQuotaAndRequestCount(userId, totalQuota)
		model.UpdateChannelUsedQuota(channelId, totalQuota)

		// 记录成本到余额交易记录
		if costUSD > 0 {
			err = model.RecordUsageCost(ctx, userId, costUSD, 0)
			if err != nil {
				logger.Error(ctx, "failed to record usage cost: "+err.Error())
			}
		}
	}
	if totalQuota <= 0 {
		logger.Error(ctx, fmt.Sprintf("totalQuota consumed is %d, something is wrong", totalQuota))
	}
}
