package model

import (
	"context"
	"errors"

	"github.com/songquanpeng/one-api/common/logger"
)

// 交易类型常量
const (
	TransactionTypeRecharge   = "recharge"   // 充值
	TransactionTypeUsage      = "usage"      // 消费
	TransactionTypeRefund     = "refund"     // 退款
	TransactionTypeAdjustment = "adjustment" // 调整
)

// BalanceTransaction 余额交易记录
type BalanceTransaction struct {
	Id           int64   `json:"id"`
	UserId       int     `json:"user_id" gorm:"index:idx_user_created"`
	Amount       float64 `json:"amount" gorm:"type:decimal(20,8)"`        // 交易金额（美元）
	BalanceAfter float64 `json:"balance_after" gorm:"type:decimal(20,8)"` // 交易后余额（美元）
	Type         string  `json:"type" gorm:"type:varchar(20)"`            // 交易类型（SQLite 不支持 ENUM）
	ReferenceId  string  `json:"reference_id" gorm:"type:varchar(100);index:idx_reference"`
	Description  string  `json:"description" gorm:"type:varchar(500)"`
	CreatedAt    int64   `json:"created_at" gorm:"bigint;index:idx_user_created"`
}

func (BalanceTransaction) TableName() string {
	return "balance_transactions"
}

// CreateBalanceTransaction 创建余额交易记录
func CreateBalanceTransaction(ctx context.Context, userId int, amount float64, transType string, referenceId string, description string) error {
	if userId == 0 {
		return errors.New("user id is empty")
	}

	// 获取当前余额（从 users 表的 quota 字段，单位：分）
	var currentQuota int64
	err := DB.Model(&User{}).Where("id = ?", userId).Select("quota").Find(&currentQuota).Error
	if err != nil {
		logger.Error(ctx, "failed to get user quota: "+err.Error())
		return err
	}

	// 将 quota（分）转换为美元
	currentBalance := float64(currentQuota) / 100.0
	newBalance := currentBalance + amount

	// 创建交易记录
	transaction := &BalanceTransaction{
		UserId:       userId,
		Amount:       amount,
		BalanceAfter: newBalance,
		Type:         transType,
		ReferenceId:  referenceId,
		Description:  description,
		CreatedAt:    GetTimestamp(),
	}

	err = DB.Create(transaction).Error
	if err != nil {
		logger.Error(ctx, "failed to create balance transaction: "+err.Error())
		return err
	}

	return nil
}

// GetUserBalanceTransactions 获取用户的交易记录
func GetUserBalanceTransactions(userId int, startIdx int, num int) ([]*BalanceTransaction, error) {
	var transactions []*BalanceTransaction
	err := DB.Where("user_id = ?", userId).
		Order("created_at DESC").
		Limit(num).
		Offset(startIdx).
		Find(&transactions).Error
	return transactions, err
}

// GetUserBalanceInUSD 获取用户余额（美元）
func GetUserBalanceInUSD(userId int) (float64, error) {
	var quota int64
	err := DB.Model(&User{}).Where("id = ?", userId).Select("quota").Find(&quota).Error
	if err != nil {
		return 0, err
	}
	// quota 单位是分，转换为美元
	return float64(quota) / 100.0, nil
}

// UpdateUserBalanceInUSD 更新用户余额（美元）
func UpdateUserBalanceInUSD(ctx context.Context, userId int, deltaUSD float64, transType string, referenceId string, description string) error {
	// 将美元转换为分
	deltaQuota := int64(deltaUSD * 100)

	// 开启事务
	tx := DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 更新用户 quota
	var err error
	if deltaQuota >= 0 {
		err = IncreaseUserQuota(userId, deltaQuota)
	} else {
		err = DecreaseUserQuota(userId, -deltaQuota)
	}

	if err != nil {
		tx.Rollback()
		return err
	}

	// 创建交易记录
	err = CreateBalanceTransaction(ctx, userId, deltaUSD, transType, referenceId, description)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

// RecordUsageCost 记录使用成本
func RecordUsageCost(ctx context.Context, userId int, costUSD float64, logId int) error {
	if costUSD <= 0 {
		return nil
	}

	return UpdateUserBalanceInUSD(
		ctx,
		userId,
		-costUSD, // 负数表示扣费
		TransactionTypeUsage,
		string(rune(logId)),
		"API usage cost",
	)
}
