package minio

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/minio/minio-go/v7"
)

// ==================== 对象操作 ====================

// UploadFileOptions 上传文件选项
type UploadFileOptions struct {
	ContentType string            // 文件MIME类型
	Metadata    map[string]string // 自定义元数据
}

// UploadFile 上传文件
func (c *MinioClient) UploadFile(ctx context.Context, bucketName, objectName, filePath string, opts *UploadFileOptions) error {
	if ctx == nil {
		ctx = c.ctx
	}
	if bucketName == "" {
		bucketName = c.config.BucketName
	}

	// 打开文件
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// 获取文件信息
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	// 设置上传选项
	uploadOpts := minio.PutObjectOptions{}
	if opts != nil {
		uploadOpts.ContentType = opts.ContentType
		uploadOpts.UserMetadata = opts.Metadata
	}

	// 如果未指定ContentType，尝试从文件扩展名推断
	if uploadOpts.ContentType == "" {
		uploadOpts.ContentType = getContentType(filePath)
	}

	// 上传文件
	_, err = c.client.PutObject(ctx, bucketName, objectName, file, fileInfo.Size(), uploadOpts)
	if err != nil {
		return fmt.Errorf("failed to upload file: %w", err)
	}

	return nil
}

// UploadBytes 上传字节数据
func (c *MinioClient) UploadBytes(ctx context.Context, bucketName, objectName string, data []byte, contentType string) error {
	if ctx == nil {
		ctx = c.ctx
	}
	if bucketName == "" {
		bucketName = c.config.BucketName
	}

	opts := minio.PutObjectOptions{
		ContentType: contentType,
	}

	// 使用bytes.NewReader创建io.Reader
	reader := bytes.NewReader(data)
	_, err := c.client.PutObject(ctx, bucketName, objectName, reader, int64(len(data)), opts)
	if err != nil {
		return fmt.Errorf("failed to upload bytes: %w", err)
	}

	return nil
}

// UploadStream 上传数据流
func (c *MinioClient) UploadStream(ctx context.Context, bucketName, objectName string, reader io.Reader, size int64, contentType string) error {
	if ctx == nil {
		ctx = c.ctx
	}
	if bucketName == "" {
		bucketName = c.config.BucketName
	}

	opts := minio.PutObjectOptions{
		ContentType: contentType,
	}

	_, err := c.client.PutObject(ctx, bucketName, objectName, reader, size, opts)
	if err != nil {
		return fmt.Errorf("failed to upload stream: %w", err)
	}

	return nil
}

// DownloadFile 下载文件到本地
func (c *MinioClient) DownloadFile(ctx context.Context, bucketName, objectName, filePath string) error {
	if ctx == nil {
		ctx = c.ctx
	}
	if bucketName == "" {
		bucketName = c.config.BucketName
	}

	// 确保目标目录存在
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// 下载文件
	err := c.client.FGetObject(ctx, bucketName, objectName, filePath, minio.GetObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}

	return nil
}

// GetObject 获取对象（返回io.ReadCloser）
func (c *MinioClient) GetObject(ctx context.Context, bucketName, objectName string) (io.ReadCloser, error) {
	if ctx == nil {
		ctx = c.ctx
	}
	if bucketName == "" {
		bucketName = c.config.BucketName
	}

	object, err := c.client.GetObject(ctx, bucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get object: %w", err)
	}

	return object, nil
}

// GetObjectBytes 获取对象字节数据
func (c *MinioClient) GetObjectBytes(ctx context.Context, bucketName, objectName string) ([]byte, error) {
	if ctx == nil {
		ctx = c.ctx
	}

	reader, err := c.GetObject(ctx, bucketName, objectName)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read object data: %w", err)
	}

	return data, nil
}

// DeleteObject 删除对象
func (c *MinioClient) DeleteObject(ctx context.Context, bucketName, objectName string) error {
	if ctx == nil {
		ctx = c.ctx
	}
	if bucketName == "" {
		bucketName = c.config.BucketName
	}

	err := c.client.RemoveObject(ctx, bucketName, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}

	return nil
}

// DeleteObjects 批量删除对象
func (c *MinioClient) DeleteObjects(ctx context.Context, bucketName string, objectNames []string) error {
	if ctx == nil {
		ctx = c.ctx
	}
	if bucketName == "" {
		bucketName = c.config.BucketName
	}

	// 创建对象删除通道
	objectsCh := make(chan minio.ObjectInfo)

	// 发送要删除的对象
	go func() {
		defer close(objectsCh)
		for _, name := range objectNames {
			objectsCh <- minio.ObjectInfo{Key: name}
		}
	}()

	// 删除对象
	opts := minio.RemoveObjectsOptions{
		GovernanceBypass: true,
	}
	errorCh := c.client.RemoveObjects(ctx, bucketName, objectsCh, opts)

	// 检查错误
	for e := range errorCh {
		if e.Err != nil {
			return fmt.Errorf("failed to delete object %s: %w", e.ObjectName, e.Err)
		}
	}

	return nil
}

// ObjectExists 检查对象是否存在
func (c *MinioClient) ObjectExists(ctx context.Context, bucketName, objectName string) (bool, error) {
	if ctx == nil {
		ctx = c.ctx
	}
	if bucketName == "" {
		bucketName = c.config.BucketName
	}

	_, err := c.client.StatObject(ctx, bucketName, objectName, minio.StatObjectOptions{})
	if err != nil {
		// 检查是否是"对象不存在"错误
		errResponse := minio.ToErrorResponse(err)
		if errResponse.Code == "NoSuchKey" {
			return false, nil
		}
		return false, fmt.Errorf("failed to check object existence: %w", err)
	}

	return true, nil
}

// GetObjectInfo 获取对象信息
func (c *MinioClient) GetObjectInfo(ctx context.Context, bucketName, objectName string) (minio.ObjectInfo, error) {
	if ctx == nil {
		ctx = c.ctx
	}
	if bucketName == "" {
		bucketName = c.config.BucketName
	}

	info, err := c.client.StatObject(ctx, bucketName, objectName, minio.StatObjectOptions{})
	if err != nil {
		return minio.ObjectInfo{}, fmt.Errorf("failed to get object info: %w", err)
	}

	return info, nil
}

// ListObjectsWithCallback 分批列出对象（推荐用于大量对象）
func (c *MinioClient) ListObjectsWithCallback(ctx context.Context, bucketName, prefix string, recursive bool, callback func(minio.ObjectInfo) error) error {
	if ctx == nil {
		ctx = c.ctx
	}
	if bucketName == "" {
		bucketName = c.config.BucketName
	}

	opts := minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: recursive,
	}

	for object := range c.client.ListObjects(ctx, bucketName, opts) {
		if object.Err != nil {
			return fmt.Errorf("failed to list objects: %w", object.Err)
		}

		// 每个对象调用回调函数
		if err := callback(object); err != nil {
			return err
		}
	}

	return nil
}

// CopyObject 复制对象
func (c *MinioClient) CopyObject(ctx context.Context, srcBucket, srcObject, destBucket, destObject string) error {
	if ctx == nil {
		ctx = c.ctx
	}

	src := minio.CopySrcOptions{
		Bucket: srcBucket,
		Object: srcObject,
	}

	dest := minio.CopyDestOptions{
		Bucket: destBucket,
		Object: destObject,
	}

	_, err := c.client.CopyObject(ctx, dest, src)
	if err != nil {
		return fmt.Errorf("failed to copy object: %w", err)
	}

	return nil
}

// PresignedGetObject 生成预签名的下载URL（临时访问）
func (c *MinioClient) PresignedGetObject(ctx context.Context, bucketName, objectName string, expiry time.Duration) (string, error) {
	if ctx == nil {
		ctx = c.ctx
	}
	if bucketName == "" {
		bucketName = c.config.BucketName
	}

	reqParams := make(url.Values)
	presignedURL, err := c.client.PresignedGetObject(ctx, bucketName, objectName, expiry, reqParams)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return presignedURL.String(), nil
}

// PresignedPutObject 生成预签名的上传URL（临时上传）
func (c *MinioClient) PresignedPutObject(ctx context.Context, bucketName, objectName string, expiry time.Duration) (string, error) {
	if ctx == nil {
		ctx = c.ctx
	}
	if bucketName == "" {
		bucketName = c.config.BucketName
	}

	presignedURL, err := c.client.PresignedPutObject(ctx, bucketName, objectName, expiry)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned upload URL: %w", err)
	}

	return presignedURL.String(), nil
}

// ==================== 辅助方法 ====================

// getContentType 根据文件扩展名获取MIME类型
func getContentType(filename string) string {
	ext := filepath.Ext(filename)
	contentTypes := map[string]string{
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".pdf":  "application/pdf",
		".txt":  "text/plain",
		".html": "text/html",
		".json": "application/json",
		".xml":  "application/xml",
		".zip":  "application/zip",
		".doc":  "application/msword",
		".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".xls":  "application/vnd.ms-excel",
		".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		".ppt":  "application/vnd.ms-powerpoint",
		".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
	}

	if contentType, ok := contentTypes[ext]; ok {
		return contentType
	}

	return "application/octet-stream" // 默认二进制类型
}
