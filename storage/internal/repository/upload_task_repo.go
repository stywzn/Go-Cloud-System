package repository

import (
	"context"

	"github.com/stywzn/Go-Cloud-System/storage/internal/model"
	"gorm.io/gorm"
)

type UploadTaskRepository interface {
	CreateTask(ctx context.Context, task *model.UploadTask) error
	GetTask(ctx context.Context, uploadID string) (*model.UploadTask, error)
	UpdateTask(ctx context.Context, task *model.UploadTask) error
	DeleteTask(ctx context.Context, uploadID string) error
}

type uploadTaskRepository struct {
	db *gorm.DB
}

func NewUploadTaskRepository(db *gorm.DB) UploadTaskRepository {
	return &uploadTaskRepository{db: db}
}

func (r *uploadTaskRepository) CreateTask(ctx context.Context, task *model.UploadTask) error {
	return r.db.WithContext(ctx).Create(task).Error
}

func (r *uploadTaskRepository) GetTask(ctx context.Context, uploadID string) (*model.UploadTask, error) {
	var task model.UploadTask
	err := r.db.WithContext(ctx).Where("upload_id = ?", uploadID).First(&task).Error
	return &task, err
}

func (r *uploadTaskRepository) UpdateTask(ctx context.Context, task *model.UploadTask) error {
	return r.db.WithContext(ctx).Save(task).Error
}

func (r *uploadTaskRepository) DeleteTask(ctx context.Context, uploadID string) error {
	return r.db.WithContext(ctx).Where("upload_id = ?", uploadID).Delete(&model.UploadTask{}).Error
}
