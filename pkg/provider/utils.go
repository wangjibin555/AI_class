package provider

import (
	"encoding/json"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// ============ 键路径解析 ============

// ParseKeyPath 解析键路径
// 支持格式: "a.b.c" 或 "a.b[0].c"
// 【未来扩展】支持数组索引访问 a.b[0]
func ParseKeyPath(key string) []string {
	if key == "" {
		return nil
	}
	return strings.Split(key, ".")
}

// GetNestedValue 从嵌套 map 中获取值
// 支持 key 格式: "redis.host" -> map["redis"]["host"]
func GetNestedValue(data map[string]interface{}, key string) (interface{}, bool) {
	parts := ParseKeyPath(key)
	if len(parts) == 0 {
		return nil, false
	}

	var current interface{} = data
	for _, part := range parts {
		switch v := current.(type) {
		case map[string]interface{}:
			val, ok := v[part]
			if !ok {
				return nil, false
			}
			current = val
		default:
			return nil, false
		}
	}
	return current, true
}

// ============ 类型转换 ============

// ToString 将 interface{} 转换为 string
func ToString(val interface{}) (string, error) {
	switch v := val.(type) {
	case string:
		return v, nil
	case []byte:
		return string(v), nil
	case int, int8, int16, int32, int64:
		return strconv.FormatInt(reflect.ValueOf(v).Int(), 10), nil
	case uint, uint8, uint16, uint32, uint64:
		return strconv.FormatUint(reflect.ValueOf(v).Uint(), 10), nil
	case float32, float64:
		return strconv.FormatFloat(reflect.ValueOf(v).Float(), 'f', -1, 64), nil
	case bool:
		return strconv.FormatBool(v), nil
	default:
		// 尝试 JSON 序列化
		bytes, err := json.Marshal(v)
		if err != nil {
			return "", NewConfigError("ToString", "", ErrTypeMismatch)
		}
		return string(bytes), nil
	}
}

// ToInt 将 interface{} 转换为 int
func ToInt(val interface{}) (int, error) {
	switch v := val.(type) {
	case int:
		return v, nil
	case int8:
		return int(v), nil
	case int16:
		return int(v), nil
	case int32:
		return int(v), nil
	case int64:
		return int(v), nil
	case float32:
		return int(v), nil
	case float64:
		return int(v), nil
	case string:
		return strconv.Atoi(v)
	case json.Number:
		i, err := v.Int64()
		return int(i), err
	default:
		return 0, NewConfigError("ToInt", "", ErrTypeMismatch)
	}
}

// ToBool 将 interface{} 转换为 bool
func ToBool(val interface{}) (bool, error) {
	switch v := val.(type) {
	case bool:
		return v, nil
	case string:
		return strconv.ParseBool(v)
	case int, int8, int16, int32, int64:
		return reflect.ValueOf(v).Int() != 0, nil
	case float32, float64:
		return reflect.ValueOf(v).Float() != 0, nil
	default:
		return false, NewConfigError("ToBool", "", ErrTypeMismatch)
	}
}

// ToFloat64 将 interface{} 转换为 float64
func ToFloat64(val interface{}) (float64, error) {
	switch v := val.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case string:
		return strconv.ParseFloat(v, 64)
	case json.Number:
		return v.Float64()
	default:
		return 0, NewConfigError("ToFloat64", "", ErrTypeMismatch)
	}
}

// ToDuration 将 interface{} 转换为 time.Duration
// 支持格式: "5s", "1m30s", "2h", 或者毫秒数
func ToDuration(val interface{}) (time.Duration, error) {
	switch v := val.(type) {
	case time.Duration:
		return v, nil
	case string:
		return time.ParseDuration(v)
	case int:
		return time.Duration(v) * time.Millisecond, nil
	case int64:
		return time.Duration(v) * time.Millisecond, nil
	case float64:
		return time.Duration(v) * time.Millisecond, nil
	default:
		return 0, NewConfigError("ToDuration", "", ErrTypeMismatch)
	}
}

// ToStringSlice 将 interface{} 转换为 []string
func ToStringSlice(val interface{}) ([]string, error) {
	switch v := val.(type) {
	case []string:
		return v, nil
	case []interface{}:
		result := make([]string, len(v))
		for i, item := range v {
			s, err := ToString(item)
			if err != nil {
				return nil, err
			}
			result[i] = s
		}
		return result, nil
	default:
		return nil, NewConfigError("ToStringSlice", "", ErrTypeMismatch)
	}
}

// ToStringMap 将 interface{} 转换为 map[string]string
func ToStringMap(val interface{}) (map[string]string, error) {
	switch v := val.(type) {
	case map[string]string:
		return v, nil
	case map[string]interface{}:
		result := make(map[string]string, len(v))
		for key, item := range v {
			s, err := ToString(item)
			if err != nil {
				return nil, err
			}
			result[key] = s
		}
		return result, nil
	default:
		return nil, NewConfigError("ToStringMap", "", ErrTypeMismatch)
	}
}


// 【未来扩展】添加 YAML、TOML 支持
// func UnmarshalYAML(data []byte, v interface{}) error { ... }
// func UnmarshalTOML(data []byte, v interface{}) error { ... }