package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/stywzn/Go-Cloud-System/storage/internal/middleware"
	"github.com/stywzn/Go-Cloud-System/storage/internal/service"
)

type FileHandler struct {
	svc service.FileService
}

func NewFileHandler(svc service.FileService) *FileHandler {
	return &FileHandler{svc: svc}
}

// UploadHandler 基础单文件上传
func (h *FileHandler) UploadHandler(c *gin.Context) {
	userID := middleware.GetUserID(c)
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file"})
		return
	}
	res, err := h.svc.UploadFile(c.Request.Context(), file, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "upload success", "data": res})
}

// InitUpload 初始化分片上传
// 请求参数：file_name, total_size, chunk_size（可选）
// 返回：upload_id, chunk_size
func (h *FileHandler) InitUpload(c *gin.Context) {
	userID := middleware.GetUserID(c)
	fileName := c.PostForm("file_name")
	totalSize, _ := strconv.ParseInt(c.PostForm("total_size"), 10, 64)

	if fileName == "" || totalSize <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file_name or total_size"})
		return
	}

	// 默认分片大小 5MB
	chunkSize := int64(5 * 1024 * 1024)
	chunkSizeStr := c.PostForm("chunk_size")
	if chunkSizeStr != "" {
		if cs, err := strconv.ParseInt(chunkSizeStr, 10, 64); err == nil && cs > 0 {
			chunkSize = cs
		}
	}

	uploadID, respChunkSize, err := h.svc.InitUpload(c.Request.Context(), userID, fileName, totalSize, chunkSize)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"upload_id":  uploadID,
		"chunk_size": respChunkSize,
	})
}

// UploadPart 上传单个分片
// 路由参数：upload_id, part_number
// 表单字段：part（文件）
func (h *FileHandler) UploadPart(c *gin.Context) {
	uploadID := c.Param("upload_id")
	partNumberStr := c.Param("part_number")
	partNumber, err := strconv.Atoi(partNumberStr)
	if err != nil || partNumber <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid part_number"})
		return
	}

	file, err := c.FormFile("part")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid part file"})
		return
	}

	src, _ := file.Open()
	defer src.Close()

	err = h.svc.UploadPart(c.Request.Context(), uploadID, partNumber, src, file.Size)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"msg": "part uploaded successfully"})
}

// CompleteUpload 完成分片上传，触发合并
// 路由参数：upload_id
func (h *FileHandler) CompleteUpload(c *gin.Context) {
	userID := middleware.GetUserID(c)
	uploadID := c.Param("upload_id")

	res, err := h.svc.CompleteUpload(c.Request.Context(), uploadID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": res})
}

// GetUploadStatus 获取上传状态（已完成的分片列表）
// 路由参数：upload_id
func (h *FileHandler) GetUploadStatus(c *gin.Context) {
	uploadID := c.Param("upload_id")
	status, err := h.svc.GetUploadStatus(c.Request.Context(), uploadID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": status})
}
