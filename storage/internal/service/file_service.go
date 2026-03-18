package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"sync"
	"time"

	"github.com/stywzn/Go-Cloud-System/storage/internal/metrics"
	"github.com/stywzn/Go-Cloud-System/storage/internal/model"
	"github.com/stywzn/Go-Cloud-System/storage/internal/repository"
	"github.com/stywzn/Go-Cloud-System/storage/internal/storage"
)

type FileService interface {
	UploadFile(ctx context.Context, file *multipart.FileHeader, userID uint) (*model.File, error)
	// Updated signature to match Handler expectations
	InitUpload(ctx context.Context, userID uint, fileName string, fileHash string, totalSize int64, chunkSize int64) (int, string, []int, error)
	UploadPart(ctx context.Context, uploadID string, partNumber int, r io.Reader, size int64) error
	CompleteUpload(ctx context.Context, uploadID string, userID uint) (*model.File, error)
	GetUploadStatus(ctx context.Context, uploadID string) (*model.UploadTask, error)
}

type fileService struct {
	repo         repository.FileRepository
	userRepo     repository.UserRepository
	taskRepo     repository.UploadTaskRepository
	store        storage.StorageEngine
	defaultQuota int64 // 默认配额 5GB
}

func NewFileService(
	repo repository.FileRepository,
	userRepo repository.UserRepository,
	taskRepo repository.UploadTaskRepository,
	store storage.StorageEngine,
) FileService {
	return &fileService{
		repo:         repo,
		userRepo:     userRepo,
		taskRepo:     taskRepo,
		store:        store,
		defaultQuota: 5 * 1024 * 1024 * 1024, // 5GB
	}
}

var partMu sync.Mutex

func (s *fileService) UploadFile(ctx context.Context, fileHeader *multipart.FileHeader, userID uint) (*model.File, error) {
	// 检查用户是否存在（如果不存在，创建默认用户）
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		// 简化：如果用户不存在，返回错误
		return nil, errors.New("user not found")
	}

	// 检查配额
	if user.Quota < fileHeader.Size {
		return nil, errors.New("quota exceeded")
	}

	// 打开文件
	src, err := fileHeader.Open()
	if err != nil {
		return nil, errors.New("cannot open file")
	}
	defer src.Close()

	// 计算 SHA256 哈希
	hasher := sha256.New()
	io.Copy(hasher, src)
	hash := hex.EncodeToString(hasher.Sum(nil))

	// 检查文件是否已存在（秒传）
	existingFile, err := s.repo.GetByHash(ctx, hash)
	if err == nil && existingFile != nil {
		// 文件已存在，直接关联
		return existingFile, nil
	}

	// 重新打开文件用于存储
	src, err = fileHeader.Open()
	if err != nil {
		return nil, errors.New("cannot re-open file")
	}
	defer src.Close()

	// 存储文件
	storedName := hash + ".bin"
	if err := s.store.Put(storedName, src, fileHeader.Size); err != nil {
		return nil, errors.New("failed to save file")
	}

	// 创建文件记录
	file := &model.File{
		OriginalName: fileHeader.Filename,
		StoredName:   storedName,
		Hash:         hash,
		Size:         fileHeader.Size,
		FilePath:     storedName,
	}
	if err := s.repo.Create(ctx, file); err != nil {
		return nil, err
	}

	// 扣减配额
	user.Quota -= fileHeader.Size
	s.userRepo.UpdateUser(ctx, user)

	return file, nil
}

func (s *fileService) InitUpload(ctx context.Context, userID uint, fileName string, fileHash string, totalSize int64, chunkSize int64) (int, string, []int, error) {
	// 1. Check User and Quota
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return 0, "", nil, errors.New("user not found")
	}
	if user.Quota < totalSize {
		metrics.RecordUploadPartFailure("quota_exceeded")
		return 0, "", nil, errors.New("quota exceeded")
	}

	// 2. Scenario 1: Fast Resume (秒传)
	// Check if this exact file (by Hash) already exists in the system
	existingFile, err := s.repo.GetByHash(ctx, fileHash)
	if err == nil && existingFile != nil {
		// Just create a new reference in user's folder (logic omitted for brevity, assuming global uniqueness)
		return 1, "", nil, nil
	}

	// 3. Scenario 2: Resumable Upload (断点续传)
	// Check if an incomplete task exists for this user and this fileHash
	// Note: You need a GetTaskByHash method in your taskRepo. Assuming we use fileHash as UploadID for simplicity.
	uploadID := fileHash // Using hash as uploadID makes tracking state easier

	existingTask, err := s.taskRepo.GetTask(ctx, uploadID)
	if err == nil && existingTask != nil && existingTask.Status == 0 {
		var completed []int
		_ = json.Unmarshal([]byte(existingTask.CompletedChunks), &completed)
		return 2, uploadID, completed, nil
	}

	// 4. Scenario 3: Brand New Upload (全新上传)
	totalChunks := (int(totalSize) + int(chunkSize) - 1) / int(chunkSize)

	task := &model.UploadTask{
		UserID:          userID,
		UploadID:        uploadID, // Use fileHash instead of random MD5
		FileName:        fileName,
		FileSize:        totalSize,
		ChunkSize:       chunkSize,
		TotalChunks:     totalChunks,
		CompletedChunks: "[]",
		Status:          0,
	}

	if err := s.taskRepo.CreateTask(ctx, task); err != nil {
		metrics.RecordUploadPartFailure("create_task_failed")
		return 0, "", nil, err
	}

	if err := s.store.InitUpload(uploadID); err != nil {
		metrics.RecordUploadPartFailure("init_storage_failed")
		s.taskRepo.DeleteTask(ctx, uploadID)
		return 0, "", nil, err
	}

	return 3, uploadID, []int{}, nil
}

func (s *fileService) UploadPart(ctx context.Context, uploadID string, partNumber int, r io.Reader, size int64) error {
	task, err := s.taskRepo.GetTask(ctx, uploadID)
	if err != nil {
		metrics.RecordUploadPartFailure("task_not_found")
		return errors.New("upload task not found")
	}

	if task.Status != 0 {
		return errors.New("upload task not in uploading status")
	}

	start := time.Now()
	// Upload to MinIO
	if err := s.store.UploadPart(uploadID, partNumber, r, size); err != nil {
		metrics.RecordUploadPartFailure("upload_failed")
		return err
	}

	metrics.UploadPartDuration.WithLabelValues("dynamic").Observe(time.Since(start).Seconds())
	metrics.RecordUploadPartSuccess()

	// CRITICAL FIX: Lock before modifying the JSON slice to prevent Lost Update in concurrent uploads
	partMu.Lock()
	defer partMu.Unlock()

	// Re-fetch task to get the absolutely latest CompletedChunks state before modifying
	latestTask, _ := s.taskRepo.GetTask(ctx, uploadID)

	var completed []int
	_ = json.Unmarshal([]byte(latestTask.CompletedChunks), &completed)

	exists := false
	for _, v := range completed {
		if v == partNumber {
			exists = true
			break
		}
	}
	if !exists {
		completed = append(completed, partNumber)
	}

	completedJSON, _ := json.Marshal(completed)
	latestTask.CompletedChunks = string(completedJSON)

	return s.taskRepo.UpdateTask(ctx, latestTask)
}

func (s *fileService) CompleteUpload(ctx context.Context, uploadID string, userID uint) (*model.File, error) {
	task, err := s.taskRepo.GetTask(ctx, uploadID)
	if err != nil || task.Status != 0 {
		return nil, errors.New("invalid upload task")
	}

	var completed []int
	_ = json.Unmarshal([]byte(task.CompletedChunks), &completed)

	if len(completed) != task.TotalChunks {
		return nil, fmt.Errorf("incomplete chunks: %d/%d", len(completed), task.TotalChunks)
	}

	parts := make([]storage.Part, len(completed))
	for i, partNum := range completed {
		parts[i] = storage.Part{PartNumber: partNum}
	}

	// Trigger MinIO to compose the file
	finalKey, err := s.store.CompleteUpload(uploadID, parts)
	if err != nil {
		return nil, err
	}

	// Deduct Quota
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err == nil {
		user.Quota -= task.FileSize
		s.userRepo.UpdateUser(ctx, user)
	}

	// Create Final File Record
	file := &model.File{
		OriginalName: task.FileName,
		StoredName:   finalKey,
		Hash:         uploadID, // We used hash as uploadID
		Size:         task.FileSize,
		FilePath:     finalKey,
	}
	if err := s.repo.Create(ctx, file); err != nil {
		return nil, err
	}

	task.Status = 1
	s.taskRepo.UpdateTask(ctx, task)

	return file, nil
}

func (s *fileService) GetUploadStatus(ctx context.Context, uploadID string) (*model.UploadTask, error) {
	return s.taskRepo.GetTask(ctx, uploadID)
}
