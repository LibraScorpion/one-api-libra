package model

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/helper"
)

// 扩展User模型，添加Google OAuth字段
type UserGoogle struct {
	GoogleID      string `json:"google_id" gorm:"column:google_id;index"`
	Avatar        string `json:"avatar" gorm:"column:avatar"`
	Locale        string `json:"locale" gorm:"column:locale;default:'zh-CN'"`
	EmailVerified bool   `json:"email_verified" gorm:"column:email_verified;default:false"`
	LastLoginAt   int64  `json:"last_login_at" gorm:"column:last_login_at"`
	LastLoginIP   string `json:"last_login_ip" gorm:"column:last_login_ip"`
}

// GetUserByGoogleID 通过Google ID获取用户
func GetUserByGoogleID(googleID string) (*User, error) {
	if googleID == "" {
		return nil, errors.New("google ID is empty")
	}

	var user User
	err := DB.Where("google_id = ?", googleID).First(&user).Error
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// GetOrCreateUserByGoogle 通过Google信息获取或创建用户
func GetOrCreateUserByGoogle(email string, googleID string) (*User, error) {
	// 首先尝试通过Google ID查找
	user, err := GetUserByGoogleID(googleID)
	if err == nil {
		return user, nil
	}

	// 如果不存在，尝试通过邮箱查找
	user, err = GetUserByEmail(email)
	if err == nil {
		// 如果邮箱已存在但没有关联Google ID，更新Google ID
		if user.GoogleID == "" {
			user.GoogleID = googleID
			err = user.Update(false)
			if err != nil {
				return nil, err
			}
		}
		return user, nil
	}

	// 如果都不存在，返回nil（让调用方创建新用户）
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	return nil, err
}

// LinkGoogleAccount 关联Google账号
func (user *User) LinkGoogleAccount(googleID string) error {
	if googleID == "" {
		return errors.New("google ID is empty")
	}

	// 检查该Google ID是否已被其他用户使用
	var existingUser User
	err := DB.Where("google_id = ? AND id != ?", googleID, user.Id).First(&existingUser).Error
	if err == nil {
		return errors.New("this Google account is already linked to another user")
	}

	user.GoogleID = googleID
	user.EmailVerified = true
	return user.Update(false)
}

// UnlinkGoogleAccount 取消关联Google账号
func (user *User) UnlinkGoogleAccount() error {
	// 检查是否有其他登录方式
	if user.Password == "" && user.GoogleID != "" {
		return errors.New("cannot unlink Google account as it's the only login method")
	}

	user.GoogleID = ""
	return user.Update(false)
}

// HasGoogleAccount 检查是否已关联Google账号
func (user *User) HasGoogleAccount() bool {
	return user.GoogleID != ""
}

// UpdateLastLogin 更新最后登录信息
func (user *User) UpdateLastLogin(ip string) error {
	user.LastLoginAt = helper.GetTimestamp()
	user.LastLoginIP = ip
	return DB.Model(user).Updates(map[string]interface{}{
		"last_login_at": user.LastLoginAt,
		"last_login_ip": user.LastLoginIP,
	}).Error
}

// CreateUserByGoogle 通过Google信息创建用户
func CreateUserByGoogle(email, googleID, name, avatar, locale string) (*User, error) {
	user := &User{
		Username:      name,
		DisplayName:   name,
		Email:         email,
		GoogleID:      googleID,
		Avatar:        avatar,
		Locale:        locale,
		EmailVerified: true,
		Status:        UserStatusEnabled,
		Role:          RoleCommonUser,
		Quota:         config.QuotaForNewUser,
	}

	err := user.Insert(context.Background(), 0)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// GetUserSessions 获取用户的所有会话
func GetUserSessions(userId int) ([]*Token, error) {
	var tokens []*Token
	err := DB.Where("user_id = ? AND status = ?", userId, TokenStatusEnabled).
		Order("created_time desc").
		Find(&tokens).Error
	return tokens, err
}

// RevokeUserSession 撤销用户会话
func RevokeUserSession(userId int, tokenId int) error {
	return DB.Model(&Token{}).
		Where("id = ? AND user_id = ?", tokenId, userId).
		Update("status", TokenStatusDisabled).Error
}

// RevokeAllUserSessions 撤销用户的所有会话
func RevokeAllUserSessions(userId int) error {
	return DB.Model(&Token{}).
		Where("user_id = ?", userId).
		Update("status", TokenStatusDisabled).Error
}

// 用户偏好设置结构
type UserPreferences struct {
	UserId             int    `json:"user_id" gorm:"primaryKey"`
	Theme              string `json:"theme" gorm:"default:'light'"`
	Language           string `json:"language" gorm:"default:'zh-CN'"`
	Timezone           string `json:"timezone" gorm:"default:'Asia/Shanghai'"`
	EmailNotifications bool   `json:"email_notifications" gorm:"default:true"`
	WebhookURL         string `json:"webhook_url"`
	DefaultModel       string `json:"default_model"`
	DefaultParameters  string `json:"default_parameters" gorm:"type:json"`
	CreatedAt          int64  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt          int64  `json:"updated_at" gorm:"autoUpdateTime"`
}

func (UserPreferences) TableName() string {
	return "user_preferences"
}

// GetUserPreferences 获取用户偏好设置
func GetUserPreferences(userId int) (*UserPreferences, error) {
	var preferences UserPreferences
	err := DB.Where("user_id = ?", userId).First(&preferences).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// 如果不存在，创建默认设置
		preferences = UserPreferences{
			UserId:             userId,
			Theme:              "light",
			Language:           "zh-CN",
			Timezone:           "Asia/Shanghai",
			EmailNotifications: true,
		}
		err = DB.Create(&preferences).Error
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	return &preferences, nil
}

// UpdateUserPreferences 更新用户偏好设置
func UpdateUserPreferences(userId int, preferences *UserPreferences) error {
	preferences.UserId = userId
	return DB.Save(preferences).Error
}

// 用户活动日志结构
type UserActivityLog struct {
	Id          int    `json:"id" gorm:"primaryKey"`
	UserId      int    `json:"user_id" gorm:"index"`
	Action      string `json:"action"`
	Resource    string `json:"resource"`
	ResourceId  string `json:"resource_id"`
	Details     string `json:"details" gorm:"type:json"`
	IPAddress   string `json:"ip_address"`
	UserAgent   string `json:"user_agent"`
	Status      string `json:"status"`
	ErrorMsg    string `json:"error_msg"`
	RequestTime int64  `json:"request_time"`
	CreatedAt   int64  `json:"created_at" gorm:"autoCreateTime"`
}

func (UserActivityLog) TableName() string {
	return "user_activity_logs"
}

// LogUserActivity 记录用户活动
func LogUserActivity(userId int, action, resource, resourceId, details, ip, userAgent, status, errorMsg string) error {
	log := &UserActivityLog{
		UserId:      userId,
		Action:      action,
		Resource:    resource,
		ResourceId:  resourceId,
		Details:     details,
		IPAddress:   ip,
		UserAgent:   userAgent,
		Status:      status,
		ErrorMsg:    errorMsg,
		RequestTime: GetTimestamp(),
	}

	return DB.Create(log).Error
}

// GetUserActivityLogs 获取用户活动日志
func GetUserActivityLogs(userId int, offset, limit int) ([]*UserActivityLog, error) {
	var logs []*UserActivityLog
	err := DB.Where("user_id = ?", userId).
		Order("created_at desc").
		Offset(offset).
		Limit(limit).
		Find(&logs).Error
	return logs, err
}

// GetUserActivityStats 获取用户活动统计
func GetUserActivityStats(userId int) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// 获取总请求数
	var totalRequests int64
	DB.Model(&UserActivityLog{}).Where("user_id = ?", userId).Count(&totalRequests)
	stats["total_requests"] = totalRequests

	// 获取成功率
	var successCount int64
	DB.Model(&UserActivityLog{}).Where("user_id = ? AND status = ?", userId, "success").Count(&successCount)
	if totalRequests > 0 {
		stats["success_rate"] = float64(successCount) / float64(totalRequests) * 100
	} else {
		stats["success_rate"] = 0
	}

	// 获取最活跃的时间段
	type HourActivity struct {
		Hour  int `json:"hour"`
		Count int `json:"count"`
	}
	var hourlyActivity []HourActivity
	DB.Raw(`
		SELECT
			HOUR(FROM_UNIXTIME(created_at)) as hour,
			COUNT(*) as count
		FROM user_activity_logs
		WHERE user_id = ?
		GROUP BY hour
		ORDER BY count DESC
		LIMIT 1
	`, userId).Scan(&hourlyActivity)

	if len(hourlyActivity) > 0 {
		stats["most_active_hour"] = hourlyActivity[0].Hour
	}

	return stats, nil
}
