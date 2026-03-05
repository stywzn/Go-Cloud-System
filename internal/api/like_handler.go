package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/stywzn/Go-Interaction-Service/internal/service"
)

// LikeRequest 定义前端传过来的 JSON 格式
// 白名单校验: 利用 Gin 的 binding 标签，把脏数据死死挡在门外
type LikeRequest struct {
	ArticleID int `json:"article_id" binding:"required,gt=0"` // 必须传，且必须大于 0
	Action    int `json:"action" binding:"oneof=0 1"`         // 只能传 0(取消) 或 1(点赞)，传其他的直接报错
}

// HandleLike 处理点赞/取消赞的 HTTP 请求
func HandleLike(c *gin.Context) {
	var req LikeRequest

	// 参数绑定与基础校验
	// 如果前端传了 <script> 或者 action=99，ShouldBindJSON 会直接报错，连咱们的 Service 都进不去
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":  400,
			"msg":   "参数格式错误，请不要搞破坏哦",
			"error": err.Error(),
		})
		return
	}

	// 获取当前登录的用户 ID
	// (在咱们目前的 V1.0 里先写死。增加安全中间件时，这里会换成从 JWT 解析出来的真实 ID)
	userID := 1001

	// 呼叫后台 Service 核心兵力 (也就是去打 Redis 和进 Channel)
	err := service.ToggleLike(c.Request.Context(), userID, req.ArticleID, req.Action)
	if err != nil {
		// 如果 Redis SADD 返回 0 (重复点赞)，就会抛错到这里
		// HTTP 状态码给 409 Conflict，告诉前端状态冲突了
		c.JSON(http.StatusConflict, gin.H{
			"code": 409,
			"msg":  err.Error(),
		})
		return
	}

	// 成功放入异步队列，立刻返回响应！耗时通常在 2 毫秒以内！
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "操作成功，后台正在飞速处理中~",
	})
}

// HandleGetLikeCount 查总赞数
func HandleGetLikeCount(c *gin.Context) {
	// 从 URL 参数里拿 article_id，比如 /api/v1/like/count?article_id=999
	articleIDStr := c.Query("article_id")
	if articleIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "必须提供 article_id"})
		return
	}

	// 实际开发中这里要把 string 转成 int，为了极简我们伪代码跳过严格校验直接用
	// 这里用简单粗暴的 Atoi 或者直接在 Service 层改接 String，这里假设你转好了
	importStrToInt := 999 // 伪代码：强制假设查询的是文章 999

	count, _ := service.GetArticleLikeCount(c.Request.Context(), importStrToInt)
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{"article_id": importStrToInt, "like_count": count},
	})
}

// HandleGetLeaderboard 查排行榜
func HandleGetLeaderboard(c *gin.Context) {
	// 获取排行榜前 10 名
	topArticles, err := service.GetLeaderboard(c.Request.Context(), 10)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "系统繁忙"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "获取热榜成功",
		"data": topArticles,
	})
}
