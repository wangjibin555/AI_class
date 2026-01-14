package minio

import (
	"context"
	"fmt"

	"github.com/minio/minio-go/v7"
)

// ==================== 桶管理操作 ====================

// BucketExists 检查桶是否存在
func (c *MinioClient) BucketExists(ctx context.Context, bucketName string) (bool, error) {
	if ctx == nil {
		ctx = c.ctx
	}
	exists, err := c.client.BucketExists(ctx, bucketName)
	if err != nil {
		return false, fmt.Errorf("failed to check bucket existence: %w", err)
	}
	return exists, nil
}

// CreateBucket 创建桶
func (c *MinioClient) CreateBucket(ctx context.Context, bucketName, location string) error {
	if ctx == nil {
		ctx = c.ctx
	}
	if location == "" {
		location = c.config.Location
	}

	// 检查桶是否已存在
	exists, err := c.BucketExists(ctx, bucketName)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("bucket %s already exists", bucketName)
	}

	// 创建桶
	err = c.client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{
		Region: location,
	})
	if err != nil {
		return fmt.Errorf("failed to create bucket: %w", err)
	}

	return nil
}

// DeleteBucket 删除桶（桶必须为空）
func (c *MinioClient) DeleteBucket(ctx context.Context, bucketName string) error {
	if ctx == nil {
		ctx = c.ctx
	}

	err := c.client.RemoveBucket(ctx, bucketName)
	if err != nil {
		return fmt.Errorf("failed to delete bucket: %w", err)
	}
	return nil
}

// ListBuckets 列出所有桶
func (c *MinioClient) ListBuckets(ctx context.Context) ([]minio.BucketInfo, error) {
	if ctx == nil {
		ctx = c.ctx
	}

	buckets, err := c.client.ListBuckets(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list buckets: %w", err)
	}
	return buckets, nil
}

// SetBucketPolicy 设置桶策略（公开/私有）
func (c *MinioClient) SetBucketPolicy(ctx context.Context, bucketName, policy string) error {
	if ctx == nil {
		ctx = c.ctx
	}

	err := c.client.SetBucketPolicy(ctx, bucketName, policy)
	if err != nil {
		return fmt.Errorf("failed to set bucket policy: %w", err)
	}
	return nil
}

// GetBucketPolicy 获取桶策略
func (c *MinioClient) GetBucketPolicy(ctx context.Context, bucketName string) (string, error) {
	if ctx == nil {
		ctx = c.ctx
	}

	policy, err := c.client.GetBucketPolicy(ctx, bucketName)
	if err != nil {
		return "", fmt.Errorf("failed to get bucket policy: %w", err)
	}
	return policy, nil
}

// SetBucketPublic 设置桶为公开（所有人可读）
func (c *MinioClient) SetBucketPublic(ctx context.Context, bucketName string) error {
	if ctx == nil {
		ctx = c.ctx
	}

	// 公开读策略
	policy := fmt.Sprintf(`{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Effect": "Allow",
				"Principal": {"AWS": ["*"]},
				"Action": ["s3:GetObject"],
				"Resource": ["arn:aws:s3:::%s/*"]
			}
		]
	}`, bucketName)

	return c.SetBucketPolicy(ctx, bucketName, policy)
}
