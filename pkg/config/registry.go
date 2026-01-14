package config

//命名空间注册器
import (
	"encoding/json"
	"fmt"
	"sync"
)

// 命名空间解析函数
type NamespaceParseFunc func() interface{}

var (
	namespaceRegistry = make(map[string]NamespaceParseFunc)
	registryMu        sync.RWMutex
)

// 注册命名空间
func RegisterNamespace(name string, parseFunc NamespaceParseFunc) {
	registryMu.Lock()
	defer registryMu.Unlock()

	namespaceRegistry[name] = parseFunc
}

// 获取所有已注册的命名空间
func GetRegisteredNamespaces() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()

	namespaces := make([]string, 0, len(namespaceRegistry))
	for name := range namespaceRegistry {
		namespaces = append(namespaces, name)
	}

	return namespaces
}

// 解析命名空间配置
func parseNamespaceConfig(namespace string, data []byte) (interface{}, error) {
	registryMu.RLock()
	parseFunc, exists := namespaceRegistry[namespace]
	registryMu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("namespace %s not registered", namespace)
	}

	config := parseFunc()

	// JSON 反序列化
	if err := json.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("unmarshal config failed: %w", err)
	}

	return config, nil
}

// 检查命名空间是否已注册
func isNamespaceRegistered(namespace string) bool {
	registryMu.RLock()
	defer registryMu.RUnlock()
	_, exists := namespaceRegistry[namespace]
	return exists
}
