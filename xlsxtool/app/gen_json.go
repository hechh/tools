package app

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/hechh/tools/xlsxtool/domain"
)

// GenJSON 生成JSON数据文件：每张表输出一个 XxxConfig.json，结构为 {"Ary":[...]}，
// 键名保持 Excel 字段名原样（PascalCase），与 pb.go json tag 一致，可 json.Unmarshal 倒回。
func GenJSON(ctx *domain.ParseContext, dstDir string, save func(string, string, []byte) error) error {
	registerEnumConvertors(ctx)

	count := 0
	for _, st := range ctx.Structs {
		rows := make([]map[string]any, 0, len(st.Rows))
		for _, row := range st.Rows {
			item, err := marshalRow(st, row)
			if err != nil {
				return err
			}
			rows = append(rows, item)
		}

		data, err := json.MarshalIndent(map[string]any{"Ary": rows}, "", "  ")
		if err != nil {
			return err
		}

		filename := st.Type + ".json"
		if err := save(dstDir, filename, data); err != nil {
			return err
		}
		fmt.Fprintf(os.Stdout, "[OK] 生成: %s/%s\n", dstDir, filename)
		count++
	}
	fmt.Fprintf(os.Stdout, "[OK] 共生成 %d 个数据文件\n", count)
	return nil
}

// registerEnumConvertors 注册枚举转换器：查找键 → 数值，未命中兜底 0
func registerEnumConvertors(ctx *domain.ParseContext) {
	ctx.WalkEnum(func(e *domain.Enum) bool {
		domain.RegisterConvertor(e.Type, func(val string) (any, error) {
			if item, ok := e.DescMap[val]; ok {
				return item.Value, nil
			}
			return 0, nil
		}, e.Type)
		return true
	})
}

// marshalRow 将一行数据转为 map[string]any，键名为字段名原样（PascalCase）
func marshalRow(st *domain.Struct, row []string) (map[string]any, error) {
	item := make(map[string]any, len(st.FieldList))
	for _, field := range st.FieldList {
		if int(field.Position)-1 >= len(row) {
			continue
		}
		value := row[field.Position-1]
		originType := field.OriginType
		switch {
		case strings.HasPrefix(originType, "[]"):
			// repeated: 以 | 分隔逐元素转换。转换结果为 nil 的元素跳过
			// （如空 Reward/Range），其余按值 append；空串按零值转换
			// （如枚举 → 0 元素），与 xlsxtool 行为一致。空数组输出 [] 而非 null
			inner := strings.TrimPrefix(originType, "[]")
			arr := make([]any, 0)
			for _, v := range strings.Split(value, "|") {
				elem, err := domain.Convert(inner, v)
				if err != nil {
					return nil, err
				}
				if elem == nil {
					continue
				}
				arr = append(arr, elem)
			}
			item[field.Name] = arr
		case strings.HasPrefix(originType, "&"), strings.HasPrefix(originType, "*"):
			// 外部 message 引用，去掉前缀后按类型名转换
			conv, err := domain.Convert(originType[1:], value)
			if err != nil {
				return nil, err
			}
			if conv != nil {
				item[field.Name] = conv
			}
		default:
			conv, err := domain.Convert(originType, value)
			if err != nil {
				return nil, err
			}
			if conv != nil {
				item[field.Name] = conv
			}
		}
	}
	return item, nil
}
