package domain

import (
	"errors"
	"sync"

	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

var (
	converters   = make(map[string]*TypeConverter)
	convertersMu sync.RWMutex
)

// RegisterConvertor 注册类型转换器
func RegisterConvertor(target string, fn ConvertFunc, protoType string, origins ...string) {
	convertersMu.Lock()
	defer convertersMu.Unlock()
	for _, origin := range origins {
		converters[origin] = &TypeConverter{
			Target: target,
			Origin: origin,
			Proto:  protoType,
			Conv:   fn,
		}
	}
}

// Target 获取目标类型名
func Target(origin string) string {
	convertersMu.RLock()
	defer convertersMu.RUnlock()
	if c, ok := converters[origin]; ok {
		return c.Target
	}
	return origin
}

func GetProtoType(origin string) string {
	convertersMu.RLock()
	defer convertersMu.RUnlock()
	if c, ok := converters[origin]; ok {
		return c.Proto
	}
	return origin
}

// Convert 执行类型转换
func Convert(origin, val string, field protoreflect.FieldDescriptor, msg *dynamicpb.Message) error {
	convertersMu.RLock()
	c, ok := converters[origin]
	convertersMu.RUnlock()
	if !ok {
		return errors.New("未注册的类型: " + origin)
	}
	return c.Conv(val, field, msg)
}
