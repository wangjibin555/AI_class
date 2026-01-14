# MinIO 客户端使用指南

## 📖 目录
- [简介](#简介)
- [安装依赖](#安装依赖)
- [快速开始](#快速开始)
- [配置说明](#配置说明)
- [桶管理](#桶管理)
- [对象操作](#对象操作)
- [高级功能](#高级功能)
- [最佳实践](#最佳实践)
- [完整示例](#完整示例)

---

## 📚 简介

MinIO是一个高性能的对象存储服务，兼容Amazon S3 API。本客户端封装了MinIO的常用操作，提供简洁易用的接口。

### 核心功能
- ✅ 桶管理（创建、删除、列表）
- ✅ 文件上传（文件、字节、流）
- ✅ 文件下载
- ✅ 对象删除（单个、批量）
- ✅ 对象列表和信息查询
- ✅ 预签名URL（临时访问）
- ✅ 对象复制
- ✅ 桶策略管理

---

## 🔧 安装依赖

```bash
go get github.com/minio/minio-go/v7
```

或在项目根目录执行：
```bash
cd /Users/wepie/Desktop/AI-Class/doc-to-ppt-system
go get github.com/minio/minio-go/v7
```

---

## 🚀 快速开始

### 1. 创建客户端

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "doc-to-ppt-system/infrastructure/minio"
)

func main() {
    // 配置MinIO连接
    config := &minio.MinioConfig{
        Endpoint:        "localhost:9000",      // MinIO服务地址
        AccessKeyID:     "minioadmin",          // 访问密钥
        SecretAccessKey: "minioadmin",          // 密钥
        UseSSL:          false,                 // 是否使用HTTPS
        BucketName:      "my-bucket",          // 默认桶名
        Location:        "us-east-1",          // 区域
    }
    
    // 创建客户端
    client, err := minio.NewMinioClient(config)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("MinIO client created successfully!")
}
```

### 2. 上传文件

```go
ctx := context.Background()

// 上传本地文件
err := client.UploadFile(ctx, "", "document.pdf", "/path/to/file.pdf", &minio.UploadFileOptions{
    ContentType: "application/pdf",
})
if err != nil {
    log.Fatal(err)
}

fmt.Println("File uploaded successfully!")
```

### 3. 下载文件

```go
// 下载文件到本地
err := client.DownloadFile(ctx, "", "document.pdf", "/path/to/download.pdf")
if err != nil {
    log.Fatal(err)
}

fmt.Println("File downloaded successfully!")
```

---

## ⚙️ 配置说明

### MinioConfig 结构体

```go
type MinioConfig struct {
    // 基础配置
    Endpoint        string        // MinIO服务地址（必需）
    AccessKeyID     string        // 访问密钥ID（必需）
    SecretAccessKey string        // 访问密钥（必需）
    UseSSL          bool          // 是否使用SSL（默认false）
    
    // 桶配置
    BucketName string             // 默认桶名称
    Location   string             // 桶所在区域（默认us-east-1）
    
    // 超时配置
    ConnectTimeout time.Duration  // 连接超时（默认10秒）
    RequestTimeout time.Duration  // 请求超时（默认30秒）
}
```

### 配置示例

```go
// 本地开发环境
devConfig := &minio.MinioConfig{
    Endpoint:        "localhost:9000",
    AccessKeyID:     "minioadmin",
    SecretAccessKey: "minioadmin",
    UseSSL:          false,
    BucketName:      "dev-bucket",
}

// 生产环境
prodConfig := &minio.MinioConfig{
    Endpoint:        "minio.example.com:9000",
    AccessKeyID:     os.Getenv("MINIO_ACCESS_KEY"),
    SecretAccessKey: os.Getenv("MINIO_SECRET_KEY"),
    UseSSL:          true,
    BucketName:      "prod-bucket",
    Location:        "us-east-1",
    ConnectTimeout:  10 * time.Second,
    RequestTimeout:  30 * time.Second,
}
```

---

## 🗂️ 桶管理

### 创建桶

```go
// 创建新桶
err := client.CreateBucket(ctx, "new-bucket", "us-east-1")
if err != nil {
    log.Fatal(err)
}
```

### 检查桶是否存在

```go
exists, err := client.BucketExists(ctx, "my-bucket")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Bucket exists: %v\n", exists)
```

### 列出所有桶

```go
buckets, err := client.ListBuckets(ctx)
if err != nil {
    log.Fatal(err)
}

for _, bucket := range buckets {
    fmt.Printf("Bucket: %s, Created: %s\n", bucket.Name, bucket.CreationDate)
}
```

### 删除桶

```go
// 注意：桶必须为空才能删除
err := client.DeleteBucket(ctx, "old-bucket")
if err != nil {
    log.Fatal(err)
}
```

### 设置桶为公开访问

```go
// 设置桶策略为公开读
err := client.SetBucketPublic(ctx, "public-bucket")
if err != nil {
    log.Fatal(err)
}
```

---

## 📄 对象操作

### 文件上传

#### 1. 上传本地文件

```go
// 基本上传
err := client.UploadFile(ctx, "my-bucket", "docs/file.pdf", "/local/path/file.pdf", nil)

// 带选项的上传
err := client.UploadFile(ctx, "my-bucket", "docs/file.pdf", "/local/path/file.pdf", 
    &minio.UploadFileOptions{
        ContentType: "application/pdf",
        Metadata: map[string]string{
            "uploaded-by": "user123",
            "department":  "sales",
        },
    })
```

#### 2. 上传字节数据

```go
data := []byte("Hello MinIO!")
err := client.UploadBytes(ctx, "my-bucket", "hello.txt", data, "text/plain")
```

#### 3. 上传数据流

```go
file, err := os.Open("large-file.dat")
if err != nil {
    log.Fatal(err)
}
defer file.Close()

fileInfo, _ := file.Stat()
err = client.UploadStream(ctx, "my-bucket", "large-file.dat", file, fileInfo.Size(), "application/octet-stream")
```

### 文件下载

#### 1. 下载到本地文件

```go
err := client.DownloadFile(ctx, "my-bucket", "docs/file.pdf", "/local/download/file.pdf")
```

#### 2. 获取对象数据流

```go
reader, err := client.GetObject(ctx, "my-bucket", "data.json")
if err != nil {
    log.Fatal(err)
}
defer reader.Close()

// 处理数据流
data, err := io.ReadAll(reader)
```

#### 3. 直接获取字节数据

```go
data, err := client.GetObjectBytes(ctx, "my-bucket", "config.json")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Content: %s\n", string(data))
```

### 对象信息

#### 检查对象是否存在

```go
exists, err := client.ObjectExists(ctx, "my-bucket", "file.txt")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Object exists: %v\n", exists)
```

#### 获取对象详细信息

```go
info, err := client.GetObjectInfo(ctx, "my-bucket", "file.txt")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Size: %d bytes\n", info.Size)
fmt.Printf("ContentType: %s\n", info.ContentType)
fmt.Printf("LastModified: %s\n", info.LastModified)
fmt.Printf("ETag: %s\n", info.ETag)
```

### 对象列表

#### 列出桶中所有对象

```go
// 递归列出所有对象
objects, err := client.ListObjects(ctx, "my-bucket", "", true)
if err != nil {
    log.Fatal(err)
}

for _, obj := range objects {
    fmt.Printf("Object: %s, Size: %d, Modified: %s\n", 
        obj.Key, obj.Size, obj.LastModified)
}
```

#### 列出指定前缀的对象

```go
// 列出 "images/" 目录下的文件
objects, err := client.ListObjects(ctx, "my-bucket", "images/", false)
```

### 对象删除

#### 删除单个对象

```go
err := client.DeleteObject(ctx, "my-bucket", "old-file.txt")
if err != nil {
    log.Fatal(err)
}
```

#### 批量删除对象

```go
objectsToDelete := []string{
    "file1.txt",
    "file2.txt",
    "file3.txt",
}

err := client.DeleteObjects(ctx, "my-bucket", objectsToDelete)
if err != nil {
    log.Fatal(err)
}
```

### 对象复制

```go
// 在同一桶内复制
err := client.CopyObject(ctx, 
    "my-bucket", "original.txt",   // 源
    "my-bucket", "backup.txt")     // 目标

// 跨桶复制
err := client.CopyObject(ctx, 
    "source-bucket", "file.txt",   // 源
    "dest-bucket", "file.txt")     // 目标
```

---

## 🔐 高级功能

### 预签名URL（临时访问）

#### 生成下载URL

```go
// 生成1小时有效的下载链接
downloadURL, err := client.PresignedGetObject(ctx, "my-bucket", "private-file.pdf", time.Hour)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("临时下载链接: %s\n", downloadURL)
// 用户可以在1小时内通过这个URL直接下载文件，无需认证
```

#### 生成上传URL

```go
// 生成1小时有效的上传链接
uploadURL, err := client.PresignedPutObject(ctx, "my-bucket", "user-upload.jpg", time.Hour)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("临时上传链接: %s\n", uploadURL)
// 用户可以通过HTTP PUT请求直接上传到这个URL
```

### 使用上下文

#### 超时控制

```go
// 设置5秒超时
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

err := client.UploadFile(ctx, "", "file.txt", "/path/file.txt", nil)
if err != nil {
    if ctx.Err() == context.DeadlineExceeded {
        fmt.Println("上传超时")
    }
}
```

#### 请求追踪

```go
// 带trace ID的上下文
ctx := context.WithValue(context.Background(), "request_id", "req-12345")
client.WithContext(ctx).UploadFile(ctx, "", "file.txt", "/path/file.txt", nil)
```

---

## 💡 最佳实践

### 1. 使用默认桶

```go
// 在配置中设置默认桶
config := &minio.MinioConfig{
    BucketName: "app-files",
    // ... 其他配置
}

// 使用时传空字符串会自动使用默认桶
client.UploadFile(ctx, "", "file.txt", "/path/file.txt", nil)
```

### 2. 错误处理

```go
if err := client.UploadFile(ctx, "", "file.txt", "/path/file.txt", nil); err != nil {
    // 检查特定错误类型
    if minio.ToErrorResponse(err).Code == "NoSuchBucket" {
        fmt.Println("桶不存在")
    } else {
        log.Printf("上传失败: %v", err)
    }
    return err
}
```

### 3. 资源清理

```go
// 使用defer确保资源被释放
reader, err := client.GetObject(ctx, "", "file.txt")
if err != nil {
    return err
}
defer reader.Close() // 重要！

// 处理数据...
```

### 4. 批量操作

```go
// 使用批量删除而不是循环单个删除
objectsToDelete := []string{"file1.txt", "file2.txt", "file3.txt"}
err := client.DeleteObjects(ctx, "", objectsToDelete)

// 而不是：
// for _, obj := range objects {
//     client.DeleteObject(ctx, "", obj) // 效率低
// }
```

### 5. 大文件处理

```go
// 对于大文件，使用流式上传
file, _ := os.Open("large-file.dat")
defer file.Close()

fileInfo, _ := file.Stat()
client.UploadStream(ctx, "", "large-file.dat", file, fileInfo.Size(), "application/octet-stream")
```

---

## 📋 完整示例

### 文档管理系统示例

```go
package main

import (
    "context"
    "fmt"
    "log"
    "path/filepath"
    "time"
    
    "doc-to-ppt-system/infrastructure/minio"
)

type DocumentManager struct {
    client *minio.MinioClient
}

func NewDocumentManager() (*DocumentManager, error) {
    config := &minio.MinioConfig{
        Endpoint:        "localhost:9000",
        AccessKeyID:     "minioadmin",
        SecretAccessKey: "minioadmin",
        UseSSL:          false,
        BucketName:      "documents",
    }
    
    client, err := minio.NewMinioClient(config)
    if err != nil {
        return nil, err
    }
    
    return &DocumentManager{client: client}, nil
}

// 上传文档
func (dm *DocumentManager) UploadDocument(ctx context.Context, localPath, remoteName string) error {
    ext := filepath.Ext(localPath)
    contentType := getContentType(ext)
    
    return dm.client.UploadFile(ctx, "", remoteName, localPath, &minio.UploadFileOptions{
        ContentType: contentType,
        Metadata: map[string]string{
            "uploaded-at": time.Now().Format(time.RFC3339),
        },
    })
}

// 生成临时下载链接
func (dm *DocumentManager) GenerateDownloadLink(ctx context.Context, fileName string, expiry time.Duration) (string, error) {
    return dm.client.PresignedGetObject(ctx, "", fileName, expiry)
}

// 列出所有文档
func (dm *DocumentManager) ListDocuments(ctx context.Context) ([]string, error) {
    objects, err := dm.client.ListObjects(ctx, "", "", false)
    if err != nil {
        return nil, err
    }
    
    var documents []string
    for _, obj := range objects {
        documents = append(documents, obj.Key)
    }
    return documents, nil
}

// 删除过期文档
func (dm *DocumentManager) CleanupOldDocuments(ctx context.Context, olderThan time.Duration) error {
    objects, err := dm.client.ListObjects(ctx, "", "", true)
    if err != nil {
        return err
    }
    
    var toDelete []string
    cutoff := time.Now().Add(-olderThan)
    
    for _, obj := range objects {
        if obj.LastModified.Before(cutoff) {
            toDelete = append(toDelete, obj.Key)
        }
    }
    
    if len(toDelete) > 0 {
        return dm.client.DeleteObjects(ctx, "", toDelete)
    }
    
    return nil
}

func getContentType(ext string) string {
    types := map[string]string{
        ".pdf":  "application/pdf",
        ".doc":  "application/msword",
        ".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
        ".txt":  "text/plain",
    }
    if ct, ok := types[ext]; ok {
        return ct
    }
    return "application/octet-stream"
}

func main() {
    dm, err := NewDocumentManager()
    if err != nil {
        log.Fatal(err)
    }
    
    ctx := context.Background()
    
    // 上传文档
    err = dm.UploadDocument(ctx, "/path/to/document.pdf", "reports/2024-q1.pdf")
    if err != nil {
        log.Printf("上传失败: %v", err)
    }
    
    // 生成24小时有效的下载链接
    link, err := dm.GenerateDownloadLink(ctx, "reports/2024-q1.pdf", 24*time.Hour)
    if err != nil {
        log.Printf("生成链接失败: %v", err)
    }
    fmt.Printf("下载链接: %s\n", link)
    
    // 列出所有文档
    docs, err := dm.ListDocuments(ctx)
    if err != nil {
        log.Printf("列出文档失败: %v", err)
    }
    fmt.Printf("文档列表: %v\n", docs)
    
    // 清理30天前的文档
    err = dm.CleanupOldDocuments(ctx, 30*24*time.Hour)
    if err != nil {
        log.Printf("清理失败: %v", err)
    }
}
```

---

## 🔗 相关资源

- [MinIO官方文档](https://min.io/docs/minio/linux/index.html)
- [MinIO Go SDK](https://github.com/minio/minio-go)
- [本项目实现](./minio.go)

---

**最后更新**: 2025-11-24

