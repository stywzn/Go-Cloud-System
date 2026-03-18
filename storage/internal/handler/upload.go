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

func (h *FileHandler) InitUpload(c *gin.Context) {
	userID := middleware.GetUserID(c)
	fileName := c.PostForm("file_name")
	fileHash := c.PostForm("file_hash") // MUST HAVE: File fingerprint
	totalSizeStr := c.PostForm("total_size")

	totalSize, err := strconv.ParseInt(totalSizeStr, 10, 64)
	if fileName == "" || fileHash == "" || err != nil || totalSize <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file_name, file_hash or total_size"})
		return
	}

	// Default chunk size is 5MB
	chunkSize := int64(5 * 1024 * 1024)
	if csStr := c.PostForm("chunk_size"); csStr != "" {
		if cs, err := strconv.ParseInt(csStr, 10, 64); err == nil && cs > 0 {
			chunkSize = cs
		}
	}

	// Call service layer to determine upload status based on fileHash
	// Expected to return: status (int), uploadID (string), uploadedParts ([]int)
	// status: 1 = Fast Resume (秒传成功), 2 = Resume (断点续传), 3 = New Upload (全新上传)
	status, uploadID, uploadedParts, err := h.svc.InitUpload(c.Request.Context(), userID, fileName, fileHash, totalSize, chunkSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Scenario 1: Instant Upload successful
	if status == 1 {
		c.JSON(http.StatusOK, gin.H{
			"code": 200,
			"msg":  "instant upload success",
			"data": gin.H{"status": "completed"},
		})
		return
	}

	// Scenario 2 & 3: Resumable or New Upload
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "upload initialized",
		"data": gin.H{
			"upload_id":      uploadID,
			"chunk_size":     chunkSize,
			"uploaded_parts": uploadedParts, // Array of part numbers already on server
		},
	})
}

// UploadPart Uploads a specific chunk
// Route Params: upload_id, part_number
// Form Field: part (binary file), part_hash (optional but recommended for data integrity)
func (h *FileHandler) UploadPart(c *gin.Context) {
	uploadID := c.Param("upload_id")
	partNumber, err := strconv.Atoi(c.Param("part_number"))
	if err != nil || partNumber <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid part_number"})
		return
	}

	file, err := c.FormFile("part")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid part file"})
		return
	}

	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to open part file"})
		return
	}
	defer src.Close()

	// Pass down to service layer
	err = h.svc.UploadPart(c.Request.Context(), uploadID, partNumber, src, file.Size)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "part uploaded successfully"})
}

// CompleteUpload and GetUploadStatus remain mostly the same structurally.

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
