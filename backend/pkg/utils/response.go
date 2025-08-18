package utils

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Response 统一响应结构
type Response struct {
	Code      int         `json:"code"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
	Warnings  []string    `json:"warnings,omitempty"`
	Timestamp int64       `json:"timestamp"`
	RequestID string      `json:"request_id,omitempty"`
}

// SuccessResponse 成功响应
func SuccessResponse(c *gin.Context, message string, data interface{}) {
	response := Response{
		Code:      http.StatusOK,
		Message:   message,
		Data:      data,
		Timestamp: time.Now().Unix(),
		RequestID: getRequestID(c),
	}
	c.JSON(http.StatusOK, response)
}

// SuccessResponseWithWarnings 带警告的成功响应
func SuccessResponseWithWarnings(c *gin.Context, message string, data interface{}, warnings []string) {
	response := Response{
		Code:      http.StatusOK,
		Message:   message,
		Data:      data,
		Warnings:  warnings,
		Timestamp: time.Now().Unix(),
		RequestID: getRequestID(c),
	}
	c.JSON(http.StatusOK, response)
}

// ErrorResponse 错误响应（支持可选的error参数）
func ErrorResponse(c *gin.Context, statusCode int, message string, errorDetails ...string) {
	var errorDetail string
	if len(errorDetails) > 0 {
		errorDetail = errorDetails[0]
	}

	response := Response{
		Code:      statusCode,
		Message:   message,
		Error:     errorDetail,
		Timestamp: time.Now().Unix(),
		RequestID: getRequestID(c),
	}
	c.JSON(statusCode, response)
}

// ErrorResponseWithData 带数据的错误响应
func ErrorResponseWithData(c *gin.Context, statusCode int, message string, errorDetail string, data interface{}) {
	response := Response{
		Code:      statusCode,
		Message:   message,
		Error:     errorDetail,
		Data:      data,
		Timestamp: time.Now().Unix(),
		RequestID: getRequestID(c),
	}
	c.JSON(statusCode, response)
}

// ValidationErrorResponse 验证错误响应
func ValidationErrorResponse(c *gin.Context, message string, errors map[string]string) {
	response := Response{
		Code:      http.StatusBadRequest,
		Message:   message,
		Data:      errors,
		Timestamp: time.Now().Unix(),
		RequestID: getRequestID(c),
	}
	c.JSON(http.StatusBadRequest, response)
}

// PaginationResponse 分页响应
type PaginationResponse struct {
	Items       interface{} `json:"items"`
	Total       int64       `json:"total"`
	Page        int         `json:"page"`
	PageSize    int         `json:"page_size"`
	TotalPages  int         `json:"total_pages"`
	HasNext     bool        `json:"has_next"`
	HasPrevious bool        `json:"has_previous"`
}

// PaginationSuccessResponse 分页成功响应
func PaginationSuccessResponse(c *gin.Context, message string, items interface{}, total int64, page, pageSize int) {
	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))

	paginationData := PaginationResponse{
		Items:       items,
		Total:       total,
		Page:        page,
		PageSize:    pageSize,
		TotalPages:  totalPages,
		HasNext:     page < totalPages,
		HasPrevious: page > 1,
	}

	response := Response{
		Code:      http.StatusOK,
		Message:   message,
		Data:      paginationData,
		Timestamp: time.Now().Unix(),
		RequestID: getRequestID(c),
	}
	c.JSON(http.StatusOK, response)
}

// getRequestID 获取请求ID
func getRequestID(c *gin.Context) string {
	requestID := c.GetHeader("X-Request-ID")
	if requestID == "" {
		requestID = c.GetString("request_id")
	}
	return requestID
}

// LogError 记录错误日志
func LogError(message string, err error) {
	// 这里可以集成日志库，暂时使用标准输出
	if err != nil {
		fmt.Printf("[ERROR] %s: %v\n", message, err)
	} else {
		fmt.Printf("[ERROR] %s\n", message)
	}
}