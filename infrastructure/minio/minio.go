package minio

import (
	"context"
	"fmt"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinioConfig MinIO配置
type MinioConfig struct {
	// 基础配置
	Endpoint        string // MinIO服务地址 例如: localhost:9000
	AccessKeyID     string // 访问密钥ID
	SecretAccessKey string // 访问密钥
	UseSSL          bool   // 是否使用SSL

	// 桶配置
	BucketName string // 默认桶名称
	Location   string // 桶所在区域，默认 us-east-1

	// 超时配置
	ConnectTimeout time.Duration // 连接超时
	RequestTimeout time.Duration // 请求超时
}

// MinioClient MinIO客户端
type MinioClient struct {
	client *minio.Client
	ctx    context.Context
	config *MinioConfig
}

// NewMinioClient 创建MinIO客户端
func NewMinioClient(config *MinioConfig) (*MinioClient, error) {
	// 设置默认值
	if config.Location == "" {
		config.Location = "us-east-1"
	}
	if config.ConnectTimeout == 0 {
		config.ConnectTimeout = 10 * time.Second
	}
	if config.RequestTimeout == 0 {
		config.RequestTimeout = 30 * time.Second
	}

	// 初始化MinIO客户端
	minioClient, err := minio.New(config.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(config.AccessKeyID, config.SecretAccessKey, ""),
		Secure: config.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	client := &MinioClient{
		client: minioClient,
		ctx:    context.Background(),
		config: config,
	}

	// 健康检查：确保可以连接到MinIO
	ctx, cancel := context.WithTimeout(context.Background(), config.ConnectTimeout)
	defer cancel()

	// 检查默认桶是否存在，如果不存在则创建
	if config.BucketName != "" {
		exists, err := client.BucketExists(ctx, config.BucketName)
		if err != nil {
			return nil, fmt.Errorf("failed to check bucket existence: %w", err)
		}
		if !exists {
			if err := client.CreateBucket(ctx, config.BucketName, config.Location); err != nil {
				return nil, fmt.Errorf("failed to create default bucket: %w", err)
			}
		}
	}

	return client, nil
}

// GetClient 获取原始MinIO客户端（用于高级操作）
func (c *MinioClient) GetClient() *minio.Client {
	return c.client
}

// WithContext 使用特定上下文
func (c *MinioClient) WithContext(ctx context.Context) *MinioClient {
	return &MinioClient{
		client: c.client,
		ctx:    ctx,
		config: c.config,
	}
}

// GetDefaultBucket 获取默认桶名称
func (c *MinioClient) GetDefaultBucket() string {
	return c.config.BucketName
}

// GetConfig 获取配置
func (c *MinioClient) GetConfig() *MinioConfig {
	return c.config
}
