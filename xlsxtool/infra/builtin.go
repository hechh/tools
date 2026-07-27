package infra

import (
	"strings"
	"time"

	"github.com/hechh/tools/xlsxtool/domain"

	"github.com/spf13/cast"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

func init() {
	// 字符串日期转成时间戳
	domain.RegisterConvertor("int64", func(val string, field protoreflect.FieldDescriptor, msg *dynamicpb.Message) error {
		// 解析日期字符串,格式: "2026-03-31 12:00:00"
		loc, err := time.LoadLocation("Asia/Shanghai")
		if err != nil {
			return err
		}
		t, err := time.ParseInLocation("2006-01-02 15:04:05", val, loc)
		if err != nil {
			return err
		}
		timestamp := t.Unix()
		setField(field, msg, protoreflect.ValueOfInt64(timestamp))
		return nil
	}, "int64", "timestamp")

	// 整数类型
	domain.RegisterConvertor("int32", func(val string, field protoreflect.FieldDescriptor, msg *dynamicpb.Message) error {
		setField(field, msg, protoreflect.ValueOfInt32(cast.ToInt32(val)))
		return nil
	}, "int32", "int", "int8", "int16", "int32")

	domain.RegisterConvertor("int64", func(val string, field protoreflect.FieldDescriptor, msg *dynamicpb.Message) error {
		setField(field, msg, protoreflect.ValueOfInt64(cast.ToInt64(val)))
		return nil
	}, "int64", "int64")

	// 无符号整数类型
	domain.RegisterConvertor("uint32", func(val string, field protoreflect.FieldDescriptor, msg *dynamicpb.Message) error {
		setField(field, msg, protoreflect.ValueOfUint32(cast.ToUint32(val)))
		return nil
	}, "uint32", "uint", "uint8", "uint16", "uint32")

	domain.RegisterConvertor("uint64", func(val string, field protoreflect.FieldDescriptor, msg *dynamicpb.Message) error {
		setField(field, msg, protoreflect.ValueOfUint64(cast.ToUint64(val)))
		return nil
	}, "uint64", "uint64")

	// 浮点类型
	domain.RegisterConvertor("float32", func(val string, field protoreflect.FieldDescriptor, msg *dynamicpb.Message) error {
		setField(field, msg, protoreflect.ValueOfFloat32(cast.ToFloat32(val)))
		return nil
	}, "float", "float", "float32")

	domain.RegisterConvertor("float64", func(val string, field protoreflect.FieldDescriptor, msg *dynamicpb.Message) error {
		setField(field, msg, protoreflect.ValueOfFloat64(cast.ToFloat64(val)))
		return nil
	}, "double", "double", "float64")

	// 布尔和字符串
	domain.RegisterConvertor("bool", func(val string, field protoreflect.FieldDescriptor, msg *dynamicpb.Message) error {
		setField(field, msg, protoreflect.ValueOfBool(cast.ToBool(val)))
		return nil
	}, "bool", "bool")

	domain.RegisterConvertor("string", func(val string, field protoreflect.FieldDescriptor, msg *dynamicpb.Message) error {
		setField(field, msg, protoreflect.ValueOfString(val))
		return nil
	}, "string", "string")

	// 业务自定义类型
	domain.RegisterConvertor("Range64", func(val string, field protoreflect.FieldDescriptor, msg *dynamicpb.Message) error {
		splitChar := ","
		if pos := strings.Index(val, "|"); pos != -1 {
			splitChar = "|"
		}
		vals := strings.Split(val, splitChar)
		if len(vals) < 2 {
			return nil
		}
		item := dynamicpb.NewMessage(field.Message())
		ppField := item.Descriptor().Fields().ByName("Min")
		if err := domain.Convert("int64", vals[0], ppField, item); err != nil {
			return err
		}
		incrField := item.Descriptor().Fields().ByName("Max")
		if err := domain.Convert("int64", vals[1], incrField, item); err != nil {
			return err
		}
		switch {
		case field.IsList():
			msg.Mutable(field).List().Append(protoreflect.ValueOf(item))
		default:
			msg.Set(field, protoreflect.ValueOf(item))
		}
		return nil
	}, "Range64", "Range64")
	domain.RegisterConvertor("Range32", func(val string, field protoreflect.FieldDescriptor, msg *dynamicpb.Message) error {
		splitChar := ","
		if pos := strings.Index(val, "|"); pos != -1 {
			splitChar = "|"
		}
		vals := strings.Split(val, splitChar)
		if len(vals) < 2 {
			return nil
		}
		item := dynamicpb.NewMessage(field.Message())
		ppField := item.Descriptor().Fields().ByName("Min")
		if err := domain.Convert("int32", vals[0], ppField, item); err != nil {
			return err
		}
		incrField := item.Descriptor().Fields().ByName("Max")
		if err := domain.Convert("int32", vals[1], incrField, item); err != nil {
			return err
		}
		switch {
		case field.IsList():
			msg.Mutable(field).List().Append(protoreflect.ValueOf(item))
		default:
			msg.Set(field, protoreflect.ValueOf(item))
		}
		return nil
	}, "Range32", "Range32")

	domain.RegisterConvertor("Reward", func(val string, field protoreflect.FieldDescriptor, msg *dynamicpb.Message) error {
		vals := strings.Split(val, ",")
		if len(vals) < 2 {
			return nil
		}
		item := dynamicpb.NewMessage(field.Message())
		ppField := item.Descriptor().Fields().ByName("PropType")
		if err := domain.Convert("PropType", vals[0], ppField, item); err != nil {
			return err
		}
		switch {
		case len(vals) >= 3:
			// 3 参数模式: PropType, PropId, Incr（宝箱等需要指定具体道具 ID）
			pidField := item.Descriptor().Fields().ByName("PropId")
			if err := domain.Convert("uint32", vals[1], pidField, item); err != nil {
				return err
			}
			incrField := item.Descriptor().Fields().ByName("Incr")
			if err := domain.Convert("int64", vals[2], incrField, item); err != nil {
				return err
			}
		default:
			// 2 参数模式: PropType, Incr
			incrField := item.Descriptor().Fields().ByName("Incr")
			if err := domain.Convert("int64", vals[1], incrField, item); err != nil {
				return err
			}
		}
		switch {
		case field.IsList():
			msg.Mutable(field).List().Append(protoreflect.ValueOf(item))
		default:
			msg.Set(field, protoreflect.ValueOf(item))
		}
		return nil
	}, "Reward", "Reward")
}

func setField(field protoreflect.FieldDescriptor, msg *dynamicpb.Message, value protoreflect.Value) {
	switch {
	case field.IsList():
		msg.Mutable(field).List().Append(value)
	default:
		msg.Set(field, value)
	}
}
