package app

import (
	"fmt"
	"os"
	"strings"

	"github.com/hechh/tools/xlsxtool/domain"

	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

// GenData data生成用例
func GenData(ctx *domain.ParseContext, dstDir string, save func(string, string, []byte) error) error {
	registerEnumConvertors(ctx)

	count := 0
	for _, st := range ctx.Structs {
		st.Descriptor = ctx.Registry.FindMessage(st.Type)
		st.AryDescriptor = ctx.Registry.FindMessage(st.Type + "Ary")
		if st.Descriptor != nil {
			for _, field := range st.FieldList {
				field.Descriptor = st.Descriptor.Fields().ByName(protoreflect.Name(field.Name))
			}
		}

		if st.AryDescriptor == nil || st.Descriptor == nil {
			fmt.Fprintf(os.Stdout, "消息描述符未找到: %s\n", st.Type)
			continue
		}

		data, err := marshalStruct(st)
		if err != nil {
			return err
		}

		filename := st.Type + ".conf"
		if err := save(dstDir, filename, data); err != nil {
			return err
		}
		fmt.Fprintf(os.Stdout, "[OK] 生成: %s/%s\n", dstDir, filename)
		count++
	}
	fmt.Fprintf(os.Stdout, "[OK] 共生成 %d 个数据文件\n", count)
	return nil
}

func registerEnumConvertors(ctx *domain.ParseContext) {
	ctx.WalkEnum(func(e *domain.Enum) bool {
		domain.RegisterConvertor(e.Type, func(val string, field protoreflect.FieldDescriptor, msg *dynamicpb.Message) error {
			var value protoreflect.Value
			if item, ok := e.DescMap[val]; ok {
				value = protoreflect.ValueOfEnum(protoreflect.EnumNumber(item.Value))
			} else {
				value = protoreflect.ValueOfEnum(0)
			}
			switch {
			case field.IsList():
				msg.Mutable(field).List().Append(value)
			default:
				msg.Set(field, value)
			}
			return nil
		}, e.Type, e.Type)
		return true
	})
}

func marshalStruct(st *domain.Struct) ([]byte, error) {
	aryMsg := dynamicpb.NewMessage(st.AryDescriptor)
	aryField := st.AryDescriptor.Fields().ByName("Ary")
	if aryField == nil {
		return nil, fmt.Errorf("field 'Ary' not found in %sAry", st.Type)
	}

	list := aryMsg.Mutable(aryField).List()
	for _, row := range st.Rows {
		itemMsg := dynamicpb.NewMessage(st.Descriptor)
		for _, field := range st.FieldList {
			if int(field.Position)-1 >= len(row) {
				fmt.Fprintf(os.Stdout, "%s 字段位置越界: field=%s, pos=%d, rowLen=%d\n", st.Type, field.Name, field.Position, len(row))
				continue
			}
			if field.Descriptor == nil {
				continue
			}

			value := row[field.Position-1]
			// 使用原始类型进行转换
			originType := field.OriginType
			if originType == "" {
				originType = field.Type // 兼容旧数据
			}
			if strings.HasPrefix(originType, "[]") {
				originType = strings.TrimPrefix(originType, "[]")
			} else if strings.HasPrefix(originType, "&") || strings.HasPrefix(originType, "*") {
				originType = originType[1:]
			}

			if field.Descriptor.IsList() {
				for _, v := range strings.Split(value, "|") {
					if err := domain.Convert(originType, v, field.Descriptor, itemMsg); err != nil {
						return nil, err
					}
				}
			} else {
				if err := domain.Convert(originType, value, field.Descriptor, itemMsg); err != nil {
					return nil, err
				}
			}
		}
		list.Append(protoreflect.ValueOf(itemMsg))
	}
	return prototext.MarshalOptions{Multiline: true}.Marshal(aryMsg)
}
