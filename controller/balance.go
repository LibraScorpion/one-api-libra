package controller

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/model"
)

// GetUserBalance 获取用户余额（美元）
func GetUserBalance(c *gin.Context) {
	userId := c.GetInt("id")
	if userId == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "未授权",
		})
		return
	}

	balance, err := model.GetUserBalanceInUSD(userId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取余额失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"balance":  balance,
			"currency": "USD",
		},
	})
}

// GetUserBalanceTransactions 获取用户交易记录
func GetUserBalanceTransactions(c *gin.Context) {
	userId := c.GetInt("id")
	if userId == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "未授权",
		})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	startIdx := (page - 1) * pageSize
	transactions, err := model.GetUserBalanceTransactions(userId, startIdx, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取交易记录失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    transactions,
	})
}

// AdminAddBalance 管理员为用户充值
func AdminAddBalance(c *gin.Context) {
	// 检查管理员权限
	userId := c.GetInt("id")
	if !model.IsAdmin(userId) {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": "权限不足",
		})
		return
	}

	var req struct {
		UserId      int     `json:"user_id" binding:"required"`
		Amount      float64 `json:"amount" binding:"required,gt=0"`
		Description string  `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	// 检查用户是否存在
	user, err := model.GetUserById(req.UserId, false)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "用户不存在",
		})
		return
	}

	// 生成描述
	description := req.Description
	if description == "" {
		description = "管理员充值"
	}

	// 充值
	err = model.UpdateUserBalanceInUSD(
		c.Request.Context(),
		req.UserId,
		req.Amount,
		model.TransactionTypeRecharge,
		"admin_"+strconv.Itoa(userId),
		description,
	)

	if err != nil {
		logger.Errorf(c.Request.Context(), "admin add balance failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "充值失败",
		})
		return
	}

	// 记录日志
	model.RecordLog(c.Request.Context(), req.UserId, model.LogTypeSystem, description)

	// 获取充值后的余额
	newBalance, _ := model.GetUserBalanceInUSD(req.UserId)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "充值成功",
		"data": gin.H{
			"user_id":     req.UserId,
			"username":    user.Username,
			"amount":      req.Amount,
			"new_balance": newBalance,
		},
	})
}

// GetAllModelPricings 获取所有模型定价（公开接口）
func GetAllModelPricings(c *gin.Context) {
	provider := c.Query("provider")

	pricings, err := model.GetAllModelPricings(provider)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取模型定价失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    pricings,
	})
}

// GetModelPricingDetail 获取单个模型定价详情（公开接口）
func GetModelPricingDetail(c *gin.Context) {
	modelName := c.Param("model")
	if modelName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "模型名称不能为空",
		})
		return
	}

	pricing, err := model.GetModelPricing(modelName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "模型定价不存在",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    pricing,
	})
}

// AdminUpdateModelPricing 管理员更新模型定价
func AdminUpdateModelPricing(c *gin.Context) {
	// 检查管理员权限
	userId := c.GetInt("id")
	if !model.IsAdmin(userId) {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": "权限不足",
		})
		return
	}

	var pricing model.ModelPricing
	if err := c.ShouldBindJSON(&pricing); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	err := model.CreateOrUpdateModelPricing(c.Request.Context(), &pricing)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "更新模型定价失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "更新成功",
		"data":    pricing,
	})
}
