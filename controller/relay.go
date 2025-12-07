package controller

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/common/helper"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/middleware"
	dbmodel "github.com/songquanpeng/one-api/model"
	"github.com/songquanpeng/one-api/monitor"
	"github.com/songquanpeng/one-api/relay/controller"
	"github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/relaymode"
)

// https://platform.openai.com/docs/api-reference/chat

func relayHelper(c *gin.Context, relayMode int) *model.ErrorWithStatusCode {
	var err *model.ErrorWithStatusCode
	switch relayMode {
	case relaymode.ImagesGenerations:
		err = controller.RelayImageHelper(c, relayMode)
	case relaymode.AudioSpeech:
		fallthrough
	case relaymode.AudioTranslation:
		fallthrough
	case relaymode.AudioTranscription:
		err = controller.RelayAudioHelper(c, relayMode)
	case relaymode.Proxy:
		err = controller.RelayProxyHelper(c, relayMode)
	default:
		err = controller.RelayTextHelper(c)
	}
	return err
}

func Relay(c *gin.Context) {
	ctx := c.Request.Context()
	relayMode := relaymode.GetByPath(c.Request.URL.Path)
	generationID := helper.GenRequestID()
	c.Set("generation_id", generationID)
	c.Header("X-OneAPI-Generation-Id", generationID)
	if config.DebugEnabled {
		requestBody, _ := common.GetRequestBody(c)
		logger.Debugf(ctx, "request body: %s", string(requestBody))
	}
	channelId := c.GetInt(ctxkey.ChannelId)
	channelName := c.GetString(ctxkey.ChannelName)
	if channelId != 0 {
		c.Header("X-OneAPI-Channel", fmt.Sprintf("%d", channelId))
	}
	if channelName != "" {
		c.Header("X-OneAPI-Channel-Name", channelName)
	}
	userId := c.GetInt(ctxkey.Id)

	startTime := time.Now()
	bizErr := relayHelper(c, relayMode)
	latency := time.Since(startTime).Milliseconds()
	lastLatency := latency
	logCallMetadata(c, generationID, 0, latency, bizErr)

	if bizErr == nil {
		if channelId != 0 {
			_ = dbmodel.OnRequestSuccess(channelId, int(latency))
		}
		if !c.Writer.Written() {
			c.Header("X-OneAPI-Latency-Ms", fmt.Sprintf("%d", latency))
		}
		monitor.Emit(channelId, true)
		return
	}
	if channelId != 0 {
		_ = dbmodel.OnRequestFailure(channelId, fmt.Sprintf("%v", bizErr.Error.Code))
	}
	lastFailedChannelId := channelId
	group := c.GetString(ctxkey.Group)
	originalModel := c.GetString(ctxkey.OriginalModel)
	go processChannelRelayError(ctx, userId, channelId, channelName, *bizErr)
	requestId := c.GetString(helper.RequestIdKey)
	retryTimes := config.RetryTimes
	if !shouldRetry(c, bizErr.StatusCode) {
		logger.Errorf(ctx, "relay error happen, status code is %d, won't retry in this case", bizErr.StatusCode)
		retryTimes = 0
	}
	for i := retryTimes; i > 0; i-- {
		var channel *dbmodel.Channel
		var err error

		// 智能重试：使用智能路由器选择最佳渠道
		// TODO: SmartRouter 功能待实现，暂时使用随机选择
		/*
			if smartRouter, exists := c.Get("smartRouter"); exists {
				if sr, ok := smartRouter.(*middleware.SmartRouter); ok {
					channel, err = sr.SelectBestChannel(ctx, group, originalModel)
				} else {
					channel, err = dbmodel.CacheGetRandomSatisfiedChannel(group, originalModel, i != retryTimes)
				}
			} else {
				channel, err = dbmodel.CacheGetRandomSatisfiedChannel(group, originalModel, i != retryTimes)
			}
		*/
		channel, err = dbmodel.CacheGetRandomSatisfiedChannel(group, originalModel, i != retryTimes)

		if err != nil {
			logger.Errorf(ctx, "failed to get channel for retry: %+v", err)
			break
		}
		logger.Infof(ctx, "using channel #%d to retry (remain times %d)", channel.Id, i)
		if channel.Id == lastFailedChannelId {
			continue
		}
		middleware.SetupContextForSelectedChannel(c, channel, originalModel)
		c.Header("X-OneAPI-Channel", fmt.Sprintf("%d", channel.Id))
		if channel.Name != "" {
			c.Header("X-OneAPI-Channel-Name", channel.Name)
		}
		requestBody, err := common.GetRequestBody(c)
		c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))

		retryStartTime := time.Now()
		bizErr = relayHelper(c, relayMode)
		retryLatency := time.Since(retryStartTime).Milliseconds()
		lastLatency = retryLatency
		logCallMetadata(c, generationID, retryTimes-i+1, retryLatency, bizErr)

		if bizErr == nil {
			_ = dbmodel.OnRequestSuccess(channel.Id, int(retryLatency))
			if !c.Writer.Written() {
				c.Header("X-OneAPI-Latency-Ms", fmt.Sprintf("%d", retryLatency))
			}
			return
		}
		channelId := c.GetInt(ctxkey.ChannelId)
		lastFailedChannelId = channelId
		channelName := c.GetString(ctxkey.ChannelName)
		_ = dbmodel.OnRequestFailure(channelId, fmt.Sprintf("%v", bizErr.Error.Code))
		go processChannelRelayError(ctx, userId, channelId, channelName, *bizErr)
	}
	if bizErr != nil {
		if !c.Writer.Written() {
			c.Header("X-OneAPI-Latency-Ms", fmt.Sprintf("%d", lastLatency))
		}
		if bizErr.StatusCode == http.StatusTooManyRequests {
			bizErr.Error.Message = "当前分组上游负载已饱和，请稍后再试"
		}

		// BUG: bizErr is in race condition
		bizErr.Error.Message = helper.MessageWithRequestId(bizErr.Error.Message, requestId)
		c.JSON(bizErr.StatusCode, gin.H{
			"error": bizErr.Error,
		})
	}
}

func shouldRetry(c *gin.Context, statusCode int) bool {
	if _, ok := c.Get(ctxkey.SpecificChannelId); ok {
		return false
	}
	if statusCode == http.StatusTooManyRequests {
		return true
	}
	if statusCode/100 == 5 {
		return true
	}
	if statusCode == http.StatusBadRequest {
		return false
	}
	if statusCode/100 == 2 {
		return false
	}
	return true
}

func processChannelRelayError(ctx context.Context, userId int, channelId int, channelName string, err model.ErrorWithStatusCode) {
	logger.Errorf(ctx, "relay error (channel id %d, user id: %d): %s", channelId, userId, err.Message)
	// https://platform.openai.com/docs/guides/error-codes/api-errors
	if monitor.ShouldDisableChannel(&err.Error, err.StatusCode) {
		monitor.DisableChannel(channelId, channelName, err.Message)
	} else {
		monitor.Emit(channelId, false)
	}
}

func RelayNotImplemented(c *gin.Context) {
	err := model.Error{
		Message: "API not implemented",
		Type:    "one_api_error",
		Param:   "",
		Code:    "api_not_implemented",
	}
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": err,
	})
}

func RelayNotFound(c *gin.Context) {
	err := model.Error{
		Message: fmt.Sprintf("Invalid URL (%s %s)", c.Request.Method, c.Request.URL.Path),
		Type:    "invalid_request_error",
		Param:   "",
		Code:    "",
	}
	c.JSON(http.StatusNotFound, gin.H{
		"error": err,
	})
}

func logCallMetadata(c *gin.Context, generationID string, attempt int, latency int64, bizErr *model.ErrorWithStatusCode) {
	status := http.StatusOK
	errCode := ""
	if bizErr != nil {
		status = bizErr.StatusCode
		errCode = fmt.Sprintf("%v", bizErr.Error.Code)
	}
	meta := &dbmodel.CallMetadata{
		GenerationID:     generationID,
		RequestID:        c.GetString(helper.RequestIdKey),
		UserID:           c.GetInt(ctxkey.Id),
		TokenID:          c.GetInt(ctxkey.TokenId),
		ChannelID:        c.GetInt(ctxkey.ChannelId),
		Model:            c.GetString(ctxkey.OriginalModel),
		APIPath:          c.Request.URL.Path,
		IsStream:         c.GetBool("is_stream"),
		StatusCode:       status,
		LatencyMs:        latency,
		PromptTokens:     c.GetInt("prompt_tokens"),
		CompletionTokens: c.GetInt("completion_tokens"),
		Attempt:          attempt,
	}
	if err := dbmodel.InsertCallMetadata(meta); err != nil {
		logger.Debugf(c.Request.Context(), "failed to insert call metadata: %v (code=%s)", err, errCode)
	}
}
