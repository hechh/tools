package infra

import (
	"strings"
	"time"

	"github.com/hechh/tools/jsontool/domain"

	"github.com/spf13/cast"
)

func init() {
	// 整数类型
	domain.RegisterConvertor("int32", func(val string) (any, error) {
		return cast.ToInt32(val), nil
	}, "int32", "int", "int8", "int16")

	domain.RegisterConvertor("int64", func(val string) (any, error) {
		return cast.ToInt64(val), nil
	}, "int64")

	// 无符号整数类型
	domain.RegisterConvertor("uint32", func(val string) (any, error) {
		return cast.ToUint32(val), nil
	}, "uint32", "uint", "uint8", "uint16")

	domain.RegisterConvertor("uint64", func(val string) (any, error) {
		return cast.ToUint64(val), nil
	}, "uint64")

	// 浮点类型
	domain.RegisterConvertor("float", func(val string) (any, error) {
		return cast.ToFloat32(val), nil
	}, "float", "float32")

	domain.RegisterConvertor("double", func(val string) (any, error) {
		return cast.ToFloat64(val), nil
	}, "double", "float64")

	// 布尔和字符串
	domain.RegisterConvertor("bool", func(val string) (any, error) {
		return cast.ToBool(val), nil
	}, "bool")

	domain.RegisterConvertor("string", func(val string) (any, error) {
		return val, nil
	}, "string")

	// timestamp 日期时间字符串 → Unix 秒（Asia/Shanghai），proto 类型为 int64
	domain.RegisterConvertor("int64", func(val string) (any, error) {
		loc, err := time.LoadLocation("Asia/Shanghai")
		if err != nil {
			return nil, err
		}
		t, err := time.ParseInLocation("2006-01-02 15:04:05", val, loc)
		if err != nil {
			return nil, err
		}
		return t.Unix(), nil
	}, "timestamp")

	// Range64 区间 {Min, Max}，支持 "," 或 "|" 分隔
	domain.RegisterConvertor("Range64", func(val string) (any, error) {
		return rangeValue(val, func(s string) any { return cast.ToInt64(s) })
	}, "Range64")

	// Range32 区间 {Min, Max}，支持 "," 或 "|" 分隔
	domain.RegisterConvertor("Range32", func(val string) (any, error) {
		return rangeValue(val, func(s string) any { return cast.ToInt32(s) })
	}, "Range32")

	// Reward 奖励 {PropType, PropId, Incr}，PropType 为枚举，需在枚举转换器注册后执行
	domain.RegisterConvertor("Reward", func(val string) (any, error) {
		vals := strings.Split(val, ",")
		if len(vals) < 2 {
			return nil, nil
		}
		item := make(map[string]any, 3)
		propType, err := domain.Convert("PropType", vals[0])
		if err != nil {
			return nil, err
		}
		item["PropType"] = propType
		switch {
		case len(vals) >= 3:
			// 3 参数模式: PropType, PropId, Incr（宝箱等需要指定具体道具 ID）
			item["PropId"] = cast.ToUint32(vals[1])
			item["Incr"] = cast.ToInt64(vals[2])
		default:
			// 2 参数模式: PropType, Incr
			item["Incr"] = cast.ToInt64(vals[1])
		}
		return item, nil
	}, "Reward")
}

// rangeValue 解析区间字符串（"," 或 "|" 分隔）为 {Min, Max} 对象
func rangeValue(val string, conv func(string) any) (any, error) {
	splitChar := ","
	if pos := strings.Index(val, "|"); pos != -1 {
		splitChar = "|"
	}
	vals := strings.Split(val, splitChar)
	if len(vals) < 2 {
		return nil, nil
	}
	return map[string]any{"Min": conv(vals[0]), "Max": conv(vals[1])}, nil
}
