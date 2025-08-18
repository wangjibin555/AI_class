package auth

// WechatClientInterface 微信客户端接口
// 定义统一的微信客户端接口，支持真实客户端和Mock客户端
type WechatClientInterface interface {
	// GetSession 通过授权码获取微信会话信息
	GetSession(code string) (*WechatSession, error)

	// ValidateUserInfo 验证并解密微信用户信息
	ValidateUserInfo(encryptedData, iv, sessionKey string) (*WechatUserInfo, error)
}

// 确保现有实现满足接口约束
var _ WechatClientInterface = (*WechatClient)(nil)
var _ WechatClientInterface = (*MockWechatClient)(nil)
