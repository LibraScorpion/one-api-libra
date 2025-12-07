package middleware

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/common/helper"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/model"
	"github.com/songquanpeng/one-api/pkg/router"
)

// SmartDistribute 智能分发中间件
// 使用新的路由引擎进行渠道选择
func SmartDistribute(routerEngine *router.Engine) func(c *gin.Context) {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		userId := c.GetInt(ctxkey.Id)
		userGroup, _ := model.CacheGetUserGroup(userId)
		c.Set(ctxkey.Group, userGroup)

		var requestModel string
		var channel *model.Channel

		// 检查是否指定了渠道 ID
		channelId, ok := c.Get(ctxkey.SpecificChannelId)
		if ok {
			// 用户指定了渠道，直接使用
			id, err := strconv.Atoi(channelId.(string))
			if err != nil {
				abortWithMessage(c, http.StatusBadRequest, "无效的渠道 Id")
				return
			}
			channel, err = model.GetChannelById(id, true)
			if err != nil {
				abortWithMessage(c, http.StatusBadRequest, "无效的渠道 Id")
				return
			}
			if channel.Status != model.ChannelStatusEnabled {
				abortWithMessage(c, http.StatusForbidden, "该渠道已被禁用")
				return
			}
		} else {
			// 使用路由引擎自动选择
			requestModel = c.GetString(ctxkey.RequestModel)

			// 构建路由请求
			selectReq := &router.SelectRequest{
				RequestID: c.GetString(helper.RequestIdKey),
				UserID:    userId,
				Group:     userGroup,
				Model:     requestModel,
				Strategy:  router.StrategyPriority, // 默认使用优先级策略
			}

			// 调用路由引擎
			result, err := routerEngine.SelectChannel(ctx, selectReq)
			if err != nil {
				message := fmt.Sprintf("当前分组 %s 下对于模型 %s 无可用渠道: %v", userGroup, requestModel, err)
				logger.SysError(message)
				abortWithMessage(c, http.StatusServiceUnavailable, message)
				return
			}

			channel = result.Channel
			logger.Debugf(ctx, "Smart router selected channel #%d, reason: %s, candidates: %d, decision_time: %v",
				channel.Id, result.Reason, result.CandidateCount, result.DecisionTime)
		}

		logger.Debugf(ctx, "user id %d, user group: %s, request model: %s, using channel #%d",
			userId, userGroup, requestModel, channel.Id)

		SetupContextForSelectedChannel(c, channel, requestModel)
		c.Next()
	}
}