package minio

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"
)

func TestMinioClient(t *testing.T) {
	// 创建配置
	config := &MinioConfig{
		Endpoint:        "localhost:9000",
		AccessKeyID:     "minioadmin",
		SecretAccessKey: "minioadmin",
		UseSSL:          false,
		BucketName:      "test-bucket",
		Location:        "us-east-1",
	}

	// 创建客户端
	client, err := NewMinioClient(config)
	if err != nil {
		t.Fatalf("Failed to create minio client: %v", err)
	}

	ctx := context.Background()

	// 测试桶操作
	t.Run("Bucket Operations", func(t *testing.T) {
		// 列出所有桶
		buckets, err := client.ListBuckets(ctx)
		if err != nil {
			t.Errorf("Failed to list buckets: %v", err)
		}
		fmt.Printf("Total buckets: %d\n", len(buckets))

		// 检查桶是否存在
		exists, err := client.BucketExists(ctx, config.BucketName)
		if err != nil {
			t.Errorf("Failed to check bucket existence: %v", err)
		}
		fmt.Printf("Bucket %s exists: %v\n", config.BucketName, exists)
	})

	// 测试文件上传
	t.Run("Upload File", func(t *testing.T) {
		// 创建测试文件
		testFile := "/tmp/test.txt"
		content := []byte("Hello MinIO!")
		if err := os.WriteFile(testFile, content, 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
		defer os.Remove(testFile)

		// 上传文件
		err := client.UploadFile(ctx, "", "test.txt", testFile, &UploadFileOptions{
			ContentType: "text/plain",
		})
		if err != nil {
			t.Errorf("Failed to upload file: %v", err)
		}
		fmt.Println("File uploaded successfully")
	})

	// 测试字节上传
	t.Run("Upload Bytes", func(t *testing.T) {
		data := []byte("Hello MinIO from bytes!")
		err := client.UploadBytes(ctx, "", "test-bytes.txt", data, "text/plain")
		if err != nil {
			t.Errorf("Failed to upload bytes: %v", err)
		}
		fmt.Println("Bytes uploaded successfully")
	})

	// 测试对象是否存在
	t.Run("Object Exists", func(t *testing.T) {
		exists, err := client.ObjectExists(ctx, "", "test.txt")
		if err != nil {
			t.Errorf("Failed to check object existence: %v", err)
		}
		fmt.Printf("Object test.txt exists: %v\n", exists)
	})

	// 测试获取对象信息
	t.Run("Get Object Info", func(t *testing.T) {
		info, err := client.GetObjectInfo(ctx, "", "test.txt")
		if err != nil {
			t.Errorf("Failed to get object info: %v", err)
		}
		fmt.Printf("Object info: Size=%d, ContentType=%s\n", info.Size, info.ContentType)
	})

	// TODO:测试列出对象

	// 测试下载文件
	t.Run("Download File", func(t *testing.T) {
		downloadPath := "/tmp/downloaded-test.txt"
		defer os.Remove(downloadPath)

		err := client.DownloadFile(ctx, "", "test.txt", downloadPath)
		if err != nil {
			t.Errorf("Failed to download file: %v", err)
		}

		// 验证下载的文件
		content, err := os.ReadFile(downloadPath)
		if err != nil {
			t.Errorf("Failed to read downloaded file: %v", err)
		}
		fmt.Printf("Downloaded content: %s\n", string(content))
	})

	// 测试获取字节数据
	t.Run("Get Object Bytes", func(t *testing.T) {
		data, err := client.GetObjectBytes(ctx, "", "test-bytes.txt")
		if err != nil {
			t.Errorf("Failed to get object bytes: %v", err)
		}
		fmt.Printf("Object bytes: %s\n", string(data))
	})

	// 测试预签名URL
	t.Run("Presigned URL", func(t *testing.T) {
		// 生成下载URL（有效期1小时）
		downloadURL, err := client.PresignedGetObject(ctx, "", "test.txt", time.Hour)
		if err != nil {
			t.Errorf("Failed to generate presigned download URL: %v", err)
		}
		fmt.Printf("Download URL: %s\n", downloadURL)

		// 生成上传URL（有效期1小时）
		uploadURL, err := client.PresignedPutObject(ctx, "", "new-upload.txt", time.Hour)
		if err != nil {
			t.Errorf("Failed to generate presigned upload URL: %v", err)
		}
		fmt.Printf("Upload URL: %s\n", uploadURL)
	})

	// 测试复制对象
	t.Run("Copy Object", func(t *testing.T) {
		err := client.CopyObject(ctx, config.BucketName, "test.txt", config.BucketName, "test-copy.txt")
		if err != nil {
			t.Errorf("Failed to copy object: %v", err)
		}
		fmt.Println("Object copied successfully")
	})

	// 测试删除对象
	t.Run("Delete Object", func(t *testing.T) {
		// 删除单个对象
		err := client.DeleteObject(ctx, "", "test-copy.txt")
		if err != nil {
			t.Errorf("Failed to delete object: %v", err)
		}
		fmt.Println("Object deleted successfully")

		// 批量删除
		err = client.DeleteObjects(ctx, "", []string{"test.txt", "test-bytes.txt"})
		if err != nil {
			t.Errorf("Failed to delete objects: %v", err)
		}
		fmt.Println("Multiple objects deleted successfully")
	})
}
