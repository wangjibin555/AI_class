package server

import "AI_class/pkg/config"

const NamespaceServer = "server"

type ServerConfig struct {
	HTTP HTTPConfig `json:"http"`
	GRPC GRPCConfig `json:"rpc"`
	Name string     `json:"name"`
	Env  string     `json:"env"`
}

type GRPCConfig struct {
	Port   int  `json:"port"`
	Enable bool `json:"enable"`
}

func init() {
	config.RegisterNamespace(NamespaceServer, func() interface{} {
		return &ServerConfig{}
	})
}

func (c *ServerConfig) GetHTTPPort() int {
	return c.HTTP.Port
}

func (c *ServerConfig) GetGRPCPort() int {
	return c.GRPC.Port
}
