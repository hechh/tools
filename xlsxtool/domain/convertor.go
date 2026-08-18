package domain

import (
	"errors"
	"sync"
)

var (
	converters   = make(map[string]ConvertFunc)
	protoTypes   = make(map[string]string) // origin → proto 类型名（供 proto/code 生成）
	convertersMu sync.RWMutex
)

// ConvertFunc 类型转换函数：字符串 → Go 原生值（供 JSON 序列化）
type ConvertFunc func(val string) (any, error)

// RegisterConvertor 注册类型转换器
// protoType 为 proto 类型名（供 proto/code 生成），origins 为 Excel 第 2 行可用的类型名
func RegisterConvertor(protoType string, fn ConvertFunc, origins ...string) {
	convertersMu.Lock()
	defer convertersMu.Unlock()
	for _, origin := range origins {
		converters[origin] = fn
		protoTypes[origin] = protoType
	}
}

// Convert 执行类型转换
func Convert(origin, val string) (any, error) {
	convertersMu.RLock()
	fn, ok := converters[origin]
	convertersMu.RUnlock()
	if !ok {
		return nil, errors.New("未注册的类型: " + origin)
	}
	return fn(val)
}

// GetProtoType 返回 origin 对应的 proto 类型名（未注册时原样返回）
func GetProtoType(origin string) string {
	convertersMu.RLock()
	defer convertersMu.RUnlock()
	if t, ok := protoTypes[origin]; ok {
		return t
	}
	return origin
}
