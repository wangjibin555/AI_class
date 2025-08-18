package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

// ConfigHandler 配置处理器
type ConfigHandler struct{}

// NewConfigHandler 创建配置处理器
func NewConfigHandler() *ConfigHandler {
	return &ConfigHandler{}
}

// GetClientConfig 获取客户端配置
func (h *ConfigHandler) GetClientConfig(c *gin.Context) {
	// 获取服务器配置
	externalHost := viper.GetString("server.external_host")
	externalPort := viper.GetString("server.external_port")
	wsHost := viper.GetString("server.ws_host")
	wsPort := viper.GetString("server.ws_port")

	// 设置默认值
	if externalHost == "" {
		externalHost = "wangjibin-sc.wepie.com"
	}
	if externalPort == "" {
		externalPort = "9000"
	}
	if wsHost == "" {
		wsHost = externalHost
	}
	if wsPort == "" {
		wsPort = externalPort
	}

	// 构建配置响应
	config := gin.H{
		"server": gin.H{
			"base_url": "https://" + externalHost + ":" + externalPort + "/api/v1",
			"ws_url":   "wss://" + wsHost + ":" + wsPort + "/ws",
			"file_url": "https://" + externalHost + ":" + externalPort,
		},
		"app": gin.H{
			"name":    viper.GetString("app.name"),
			"version": viper.GetString("app.version"),
			"mode":    viper.GetString("app.mode"),
		},
		"features": gin.H{
			"tts_enabled":    viper.IsSet("aliyun.tts.app_key"),
			"ai_enabled":     viper.IsSet("aliyun.dashscope.api_key"),
			"coze_enabled":   viper.IsSet("coze.token"),
			"wechat_enabled": viper.IsSet("wechat.app_id"),
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "配置获取成功",
		"data":    config,
	})
}
