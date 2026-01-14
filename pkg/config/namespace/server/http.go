package server

// HTTPConfig HTTP 配置
type HTTPConfig struct {
	Host            string `json:"host"`
	Port            int    `json:"port"`
	ReadTimeout     int    `json:"read_timeout"`
	WriteTimeout    int    `json:"write_timeout"`
	MaxHeaderBytes  int    `json:"max_header_bytes"`
	EnableCORS      bool   `json:"enable_cors"`
	EnableRateLimit bool   `json:"enable_rate_limit"`
}
