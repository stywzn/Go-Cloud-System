package storage

import (
	"context"
	"fmt"
	"io"
	"path/filepath"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinIOStorage struct {
	client *minio.Client
	bucket string
}

// NewMinIOStorage 初始化MinIO存储引擎
func NewMinIOStorage(endpoint, accessKey, secretKey, bucket string, useSSL bool) (*MinIOStorage, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	// 检查bucket是否存在，不存在则创建
	ctx := context.Background()
	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to check bucket: %w", err)
	}

	if !exists {
		err = client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{Region: "us-east-1"})
		if err != nil {
			return nil, fmt.Errorf("failed to create bucket: %w", err)
		}
	}

	return &MinIOStorage{
		client: client,
		bucket: bucket,
	}, nil
}

// Put 上传单个文件
func (m *MinIOStorage) Put(key string, r io.Reader, size int64) error {
	ctx := context.Background()
	_, err := m.client.PutObject(ctx, m.bucket, key, r, size, minio.PutObjectOptions{
		ContentType: "application/octet-stream",
	})
	return err
}

// InitUpload 初始化分片上传
func (m *MinIOStorage) InitUpload(uploadID string) error {
	// MinIO提供类似S3的分片上传，但不需要主动初始化
	// 这里我们可以创建一个标记文件或记录在数据库中
	// 跳过即可，真实场景中应该记录uploadID到数据库
	return nil
}

// UploadPart 上传单个分片
// partNumber从1开始
func (m *MinIOStorage) UploadPart(uploadID string, partNumber int, r io.Reader, size int64) error {
	ctx := context.Background()
	partKey := filepath.Join("temp", uploadID, fmt.Sprintf("part-%d", partNumber))
	_, err := m.client.PutObject(ctx, m.bucket, partKey, r, size, minio.PutObjectOptions{
		ContentType: "application/octet-stream",
	})
	return err
}

// 上面的初始化和 Put 保持不变...
// CompleteUpload 生产级修复：使用 ComposeObject 服务端合并，彻底解决 OOM
func (m *MinIOStorage) CompleteUpload(uploadID string, parts []Part) (string, error) {
	ctx := context.Background()
	// 真实场景下，最终的文件名(Key)通常是由调用方传入的(比如文件的MD5)，这里先按你的逻辑用 .merged
	finalKey := uploadID + ".merged"

	// 1. 构造合并源对象列表
	var srcOpts []minio.CopySrcOptions
	for _, part := range parts {
		partKey := filepath.Join("temp", uploadID, fmt.Sprintf("part-%d", part.PartNumber))
		srcOpts = append(srcOpts, minio.CopySrcOptions{
			Bucket: m.bucket,
			Object: partKey,
		})
	}

	// 2. 构造目标对象
	dstOpt := minio.CopyDestOptions{
		Bucket: m.bucket,
		Object: finalKey,
	}

	// 3. 核心调用：命令 MinIO 在服务端进行无内存消耗的拼接
	// 注意：如果是超大文件(超过 10000 个分片)，需要分批 Compose，一般情况足够了
	_, err := m.client.ComposeObject(ctx, dstOpt, srcOpts...)
	if err != nil {
		return "", fmt.Errorf("failed to compose object on server side: %w", err)
	}

	// 4. 异步清理临时分片 (不阻塞主流程返回)
	go func() {
		cleanCtx := context.Background()
		for _, part := range parts {
			partKey := filepath.Join("temp", uploadID, fmt.Sprintf("part-%d", part.PartNumber))
			if err := m.client.RemoveObject(cleanCtx, m.bucket, partKey, minio.RemoveObjectOptions{}); err != nil {
				fmt.Printf("[Async Cleanup] warning: failed to remove part %s: %v\n", partKey, err)
			}
		}
	}()

	return finalKey, nil
}

// AbortUpload 中止分片上传
func (m *MinIOStorage) AbortUpload(uploadID string) error {
	ctx := context.Background()
	tempDir := filepath.Join("temp", uploadID)

	// 列出临时目录下的所有文件并删除
	objectsChan := m.client.ListObjects(ctx, m.bucket, minio.ListObjectsOptions{
		Prefix:    tempDir,
		Recursive: true,
	})

	for object := range objectsChan {
		if object.Err != nil {
			return object.Err
		}
		if err := m.client.RemoveObject(ctx, m.bucket, object.Key, minio.RemoveObjectOptions{}); err != nil {
			return err
		}
	}

	return nil
}
