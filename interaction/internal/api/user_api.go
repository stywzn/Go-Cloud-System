package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/stywzn/Go-Cloud-System/interaction/config"
	"github.com/stywzn/Go-Cloud-System/pkg/jwt" // Add JWT package
	"github.com/stywzn/Go-Cloud-System/pkg/trace"
)

// RegisterHandler 用户注册
func RegisterHandler(c *gin.Context) {
	traceID := trace.GetTraceID(c)

	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	// 检查用户名是否已存在
	var count int64
	err := config.DB.Raw("SELECT COUNT(*) FROM users WHERE username = ?", req.Username).Scan(&count).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "数据库错误"})
		return
	}

	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "用户名已存在"})
		return
	}

	// 创建用户
	err = config.DB.Exec("INSERT INTO users (username, password, quota) VALUES (?, ?, ?)",
		req.Username, req.Password, 5368709120).Error // 5GB默认配额
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "注册失败"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":  "注册成功",
		"trace_id": traceID,
	})
}

// LoginHandler handles user login and issues a JWT token
func LoginHandler(c *gin.Context) {
	traceID := trace.GetTraceID(c)

	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	// Query user data from the database
	var user struct {
		ID       int    `json:"id"`
		Username string `json:"username"`
		Password string `json:"password"`
		Quota    int64  `json:"quota"`
	}

	err := config.DB.Raw("SELECT id, username, password, quota FROM users WHERE username = ?", req.Username).Scan(&user).Error
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}

	// Verify password (currently plain text, ideally bcrypt should be used)
	if user.Password != req.Password {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}

	// Generate standard JWT token using the imported package
	token, err := jwt.GenerateToken(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "系统内部错误"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "登录成功",
		"user_id":  user.ID,
		"username": user.Username,
		"quota":    user.Quota,
		"token":    token,
		"trace_id": traceID,
	})
}

// GetUserInfoHandler 获取用户信息
func GetUserInfoHandler(c *gin.Context) {
	traceID := trace.GetTraceID(c)

	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	userID := userIDVal.(int)

	var user struct {
		ID       int    `json:"id"`
		Username string `json:"username"`
		Quota    int64  `json:"quota"`
	}

	err := config.DB.Raw("SELECT id, username, quota FROM users WHERE id = ?", userID).Scan(&user).Error
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user":     user,
		"trace_id": traceID,
	})
}
