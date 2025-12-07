package controller

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/random"
	"github.com/songquanpeng/one-api/model"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// Google OAuth2 配置
var googleOauthConfig = &oauth2.Config{
	ClientID:     "", // 从环境变量加载
	ClientSecret: "", // 从环境变量加载
	RedirectURL:  "", // 从环境变量加载
	Scopes: []string{
		"https://www.googleapis.com/auth/userinfo.email",
		"https://www.googleapis.com/auth/userinfo.profile",
	},
	Endpoint: google.Endpoint,
}

// Google 用户信息结构
type GoogleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	VerifiedEmail bool   `json:"verified_email"`
	Locale        string `json:"locale"`
}

func init() {
	// 从环境变量加载配置
	if clientID := config.OptionMap["GOOGLE_CLIENT_ID"]; clientID != "" {
		googleOauthConfig.ClientID = clientID
	}
	if clientSecret := config.OptionMap["GOOGLE_CLIENT_SECRET"]; clientSecret != "" {
		googleOauthConfig.ClientSecret = clientSecret
	}
	if redirectURL := config.OptionMap["GOOGLE_REDIRECT_URL"]; redirectURL != "" {
		googleOauthConfig.RedirectURL = redirectURL
	}
}

// GoogleLoginHandler 处理Google登录重定向
func GoogleLoginHandler(c *gin.Context) {
	// 生成随机state防止CSRF攻击
	state := generateStateToken()

	// 将state存储在session中
	session := c.MustGet("session").(map[string]interface{})
	session["oauth_state"] = state

	// 获取授权URL
	url := googleOauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)

	c.JSON(200, gin.H{
		"success":  true,
		"auth_url": url,
		"state":    state,
	})
}

// GoogleCallbackHandler 处理Google OAuth回调
func GoogleCallbackHandler(c *gin.Context) {
	// 验证state
	state := c.Query("state")
	session := c.MustGet("session").(map[string]interface{})
	savedState, ok := session["oauth_state"].(string)

	if !ok || state != savedState {
		c.JSON(400, gin.H{
			"success": false,
			"message": "Invalid state parameter",
		})
		return
	}

	// 获取授权码
	code := c.Query("code")
	if code == "" {
		c.JSON(400, gin.H{
			"success": false,
			"message": "No authorization code provided",
		})
		return
	}

	// 交换访问令牌
	ctx := context.Background()
	token, err := googleOauthConfig.Exchange(ctx, code)
	if err != nil {
		c.JSON(500, gin.H{
			"success": false,
			"message": "Failed to exchange token: " + err.Error(),
		})
		return
	}

	// 获取用户信息
	userInfo, err := fetchGoogleUserInfo(token.AccessToken)
	if err != nil {
		c.JSON(500, gin.H{
			"success": false,
			"message": "Failed to fetch user info: " + err.Error(),
		})
		return
	}

	// 查找或创建用户
	user, err := model.GetUserByGoogleID(userInfo.ID)
	if err != nil {
		// 用户不存在，创建新用户
		user = &model.User{
			Username: userInfo.Name,
			Email:    userInfo.Email,
			GoogleID: userInfo.ID,
			Avatar:   userInfo.Picture,
			Status:   model.UserStatusEnabled,
			Role:     model.RoleCommonUser,
		}

		if err := user.Insert(context.Background(), 0); err != nil {
			c.JSON(500, gin.H{
				"success": false,
				"message": "Failed to create user: " + err.Error(),
			})
			return
		}
	}

	// 更新用户信息
	user.Avatar = userInfo.Picture
	user.LastLoginAt = time.Now().Unix()
	user.Update(false)

	// 生成JWT令牌
	accessToken, refreshToken, err := generateTokenPair(user)
	if err != nil {
		c.JSON(500, gin.H{
			"success": false,
			"message": "Failed to generate tokens",
		})
		return
	}

	// 清除OAuth state
	delete(session, "oauth_state")

	c.JSON(200, gin.H{
		"success":       true,
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"user": gin.H{
			"id":       user.Id,
			"username": user.Username,
			"email":    user.Email,
			"avatar":   user.Avatar,
			"role":     user.Role,
		},
	})
}

// GoogleTokenLoginHandler 处理前端直接传递的Google ID Token
func GoogleTokenLoginHandler(c *gin.Context) {
	var req struct {
		Token string `json:"token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{
			"success": false,
			"message": "Invalid request",
		})
		return
	}

	// 验证Google ID Token
	userInfo, err := verifyGoogleIDToken(req.Token)
	if err != nil {
		c.JSON(401, gin.H{
			"success": false,
			"message": "Invalid token: " + err.Error(),
		})
		return
	}

	// 查找或创建用户
	user, err := model.GetUserByGoogleID(userInfo.ID)
	if err != nil {
		// 创建新用户
		user = &model.User{
			Username: userInfo.Name,
			Email:    userInfo.Email,
			GoogleID: userInfo.ID,
			Avatar:   userInfo.Picture,
			Status:   model.UserStatusEnabled,
			Role:     model.RoleCommonUser,
		}

		if err := user.Insert(context.Background(), 0); err != nil {
			c.JSON(500, gin.H{
				"success": false,
				"message": "Failed to create user",
			})
			return
		}
	}

	// 更新最后登录时间
	user.LastLoginAt = time.Now().Unix()
	user.Update(false)

	// 生成JWT令牌
	accessToken, refreshToken, err := generateTokenPair(user)
	if err != nil {
		c.JSON(500, gin.H{
			"success": false,
			"message": "Failed to generate tokens",
		})
		return
	}

	c.JSON(200, gin.H{
		"success":       true,
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"user": gin.H{
			"id":       user.Id,
			"username": user.Username,
			"email":    user.Email,
			"avatar":   user.Avatar,
			"role":     user.Role,
		},
	})
}

// 生成随机state令牌
func generateStateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

// 获取Google用户信息
func fetchGoogleUserInfo(accessToken string) (*GoogleUserInfo, error) {
	resp, err := http.Get(fmt.Sprintf(
		"https://www.googleapis.com/oauth2/v2/userinfo?access_token=%s",
		accessToken,
	))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var userInfo GoogleUserInfo
	if err := json.Unmarshal(data, &userInfo); err != nil {
		return nil, err
	}

	return &userInfo, nil
}

// 验证Google ID Token
func verifyGoogleIDToken(idToken string) (*GoogleUserInfo, error) {
	// 调用Google的tokeninfo端点验证token
	resp, err := http.Get(fmt.Sprintf(
		"https://oauth2.googleapis.com/tokeninfo?id_token=%s",
		idToken,
	))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("invalid token")
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var tokenInfo struct {
		Aud     string `json:"aud"`
		Sub     string `json:"sub"`
		Email   string `json:"email"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
	}

	if err := json.Unmarshal(data, &tokenInfo); err != nil {
		return nil, err
	}

	// 验证audience是否匹配我们的Client ID
	if tokenInfo.Aud != googleOauthConfig.ClientID {
		return nil, fmt.Errorf("invalid audience")
	}

	return &GoogleUserInfo{
		ID:            tokenInfo.Sub,
		Email:         tokenInfo.Email,
		Name:          tokenInfo.Name,
		Picture:       tokenInfo.Picture,
		VerifiedEmail: true,
	}, nil
}

// 生成JWT令牌对
func generateTokenPair(user *model.User) (accessToken, refreshToken string, err error) {
	// 简化实现：使用随机 UUID 作为访问/刷新令牌占位
	return random.GetUUID(), random.GetUUID(), nil
}

// LinkGoogleAccount 关联Google账号到现有账号
func LinkGoogleAccount(c *gin.Context) {
	userId := c.GetInt("id")

	var req struct {
		Token string `json:"token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{
			"success": false,
			"message": "Invalid request",
		})
		return
	}

	// 验证Google ID Token
	userInfo, err := verifyGoogleIDToken(req.Token)
	if err != nil {
		c.JSON(401, gin.H{
			"success": false,
			"message": "Invalid token",
		})
		return
	}

	// 检查Google ID是否已被其他账号使用
	existingUser, _ := model.GetUserByGoogleID(userInfo.ID)
	if existingUser != nil && existingUser.Id != userId {
		c.JSON(400, gin.H{
			"success": false,
			"message": "This Google account is already linked to another user",
		})
		return
	}

	// 更新用户的Google ID
	user, err := model.GetUserById(userId, false)
	if err != nil {
		c.JSON(500, gin.H{
			"success": false,
			"message": "Failed to get user",
		})
		return
	}

	user.GoogleID = userInfo.ID
	if user.Avatar == "" {
		user.Avatar = userInfo.Picture
	}

	if err := user.Update(false); err != nil {
		c.JSON(500, gin.H{
			"success": false,
			"message": "Failed to link Google account",
		})
		return
	}

	c.JSON(200, gin.H{
		"success": true,
		"message": "Google account linked successfully",
	})
}

// UnlinkGoogleAccount 取消关联Google账号
func UnlinkGoogleAccount(c *gin.Context) {
	userId := c.GetInt("id")

	user, err := model.GetUserById(userId, false)
	if err != nil {
		c.JSON(500, gin.H{
			"success": false,
			"message": "Failed to get user",
		})
		return
	}

	// 检查用户是否有其他登录方式
	if user.Password == "" && user.GoogleID != "" {
		c.JSON(400, gin.H{
			"success": false,
			"message": "Cannot unlink Google account as it's the only login method. Please set a password first.",
		})
		return
	}

	user.GoogleID = ""
	if err := user.Update(false); err != nil {
		c.JSON(500, gin.H{
			"success": false,
			"message": "Failed to unlink Google account",
		})
		return
	}

	c.JSON(200, gin.H{
		"success": true,
		"message": "Google account unlinked successfully",
	})
}
