package database

import "AI_class/pkg/config"

const NamespaceDatabase = "database"

type DataBaseConfig struct {
	Mysql MysqlConfig `json:"mysql"`
	Redis RedisConfig `json:"redis"`
}

func init() {
	config.RegisterNamespace(NamespaceDatabase, func() interface{} {
		return &DataBaseConfig{}
	})
}
