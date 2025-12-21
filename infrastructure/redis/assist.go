package redis

import "github.com/gomodule/redigo/redis"

type SendCommand struct {
	CommandName string
	Args        []interface{}
	Idx         int //每个命令的唯一标号，集群模式下需要依据这个进行确认顺序
}

// 用于存储每个Pipeline的执行结果与错误信息
type PipeApplyPair struct {
	val interface{}
	err error
}

// 辅助方法
// String 将结果转换为字符串
func (p *PipeApplyPair) String() (string, error) {
	if p.err != nil {
		return "", p.err
	}
	return redis.String(p.val, nil)
}

// Int 将结果转换为整数
func (p *PipeApplyPair) Int() (int, error) {
	if p.err != nil {
		return 0, p.err
	}
	return redis.Int(p.val, nil)
}

// Int64 将结果转换为int64
func (p *PipeApplyPair) Int64() (int64, error) {
	if p.err != nil {
		return 0, p.err
	}
	return redis.Int64(p.val, nil)
}

// Bool 将结果转换为布尔值
func (p *PipeApplyPair) Bool() (bool, error) {
	if p.err != nil {
		return false, p.err
	}
	return redis.Bool(p.val, nil)
}

// Bytes 将结果转换为字节数组
func (p *PipeApplyPair) Bytes() ([]byte, error) {
	if p.err != nil {
		return nil, p.err
	}
	return redis.Bytes(p.val, nil)
}

// Strings 将结果转换为字符串数组
func (p *PipeApplyPair) Strings() ([]string, error) {
	if p.err != nil {
		return nil, p.err
	}
	return redis.Strings(p.val, nil)
}

// IsOK 检查结果是否为"OK"
func (p *PipeApplyPair) IsOK() bool {
	if p.err != nil {
		return false
	}
	str, err := redis.String(p.val, nil)
	return err == nil && str == "OK"
}

// Value 获取原始值
func (p *PipeApplyPair) Value() (interface{}, error) {
	return p.val, p.err
}
