package domain

import (
	"strconv"
	"strings"
)

// ParseContext 解析上下文,持有所有解析结果
type ParseContext struct {
	Enums     []*Enum
	EnumMap   map[string]*Enum
	Structs   []*Struct
	StructMap map[string]*Struct
	Registry  *ProtoRegistry
}

// NewParseContext 创建解析上下文
func NewParseContext() *ParseContext {
	return &ParseContext{
		EnumMap:   make(map[string]*Enum),
		StructMap: make(map[string]*Struct),
		Registry:  NewProtoRegistry(),
	}
}

// WalkEnum 遍历枚举
func (ctx *ParseContext) WalkEnum(f func(*Enum) bool) {
	for _, item := range ctx.Enums {
		if !f(item) {
			return
		}
	}
}

// WalkStruct 遍历结构体
func (ctx *ParseContext) WalkStruct(f func(*Struct) bool) {
	for _, item := range ctx.Structs {
		if !f(item) {
			return
		}
	}
}

// ParseTable 解析单个表格(根据Token分发)
func (ctx *ParseContext) ParseTable(table *Table) {
	switch table.Token {
	case 1:
		ctx.parseEnum(table)
	case 2, 3:
		ctx.parseStruct(table)
	}
}

// parseEnum 解析枚举表格
func (ctx *ParseContext) parseEnum(table *Table) {
	for _, lines := range table.Rows {
		for _, str := range lines {
			if !strings.HasPrefix(strings.ToUpper(str), "E|") {
				continue
			}
			parts := strings.Split(str, "|")
			if len(parts) < 5 {
				continue
			}
			value, _ := strconv.ParseInt(parts[4], 10, 32)
			enumType := parts[2]

			enum, ok := ctx.EnumMap[enumType]
			if !ok {
				enum = &Enum{Type: enumType, DescMap: make(map[string]*EValue)}
				ctx.Enums = append(ctx.Enums, enum)
				ctx.EnumMap[enumType] = enum
			}
			enum.Add(parts[1], parts[3], int32(value))
		}
	}
}

// parseStruct 解析结构体表格
func (ctx *ParseContext) parseStruct(table *Table) {
	st := &Struct{
		Type:     table.Type,
		FieldMap: make(map[string]*Field),
		IndexMap: make(map[string]*Index),
		Rows:     table.Rows[3:],
	}

	for i, fieldType := range table.Rows[1] {
		if len(fieldType) <= 0 {
			continue
		}
		item := &Field{
			Name:       table.Rows[0][i],
			Type:       ParseType(fieldType), // 规范化类型（repeated/枚举/标量）
			OriginType: fieldType,            // 保存原始类型
			Desc:       table.Rows[2][i],
			Position:   int32(i) + 1,
		}
		st.FieldMap[item.Name] = item
		st.FieldList = append(st.FieldList, item)
	}

	for _, vrule := range table.Rules {
		var parent *Index
		for _, str := range strings.Split(vrule, "@") {
			vals := strings.Split(str, ":")
			idx := &Index{
				Type: vals[0],
				Name: strings.ReplaceAll(vals[1], ",", ""),
			}

			if parent != nil {
				parent.Next = idx
			} else {
				st.IndexMap[idx.Name] = idx
				st.IndexList = append(st.IndexList, idx)
			}
			parent = idx

			for _, fieldName := range strings.Split(vals[1], ",") {
				idx.List = append(idx.List, st.FieldMap[fieldName])
			}
		}
	}
	ctx.Structs = append(ctx.Structs, st)
	ctx.StructMap[st.Type] = st
}

// ParseType 类型名称规范化：[] 前缀 → repeated，&/* 前缀去掉，其余按注册的 proto 类型映射
func ParseType(str string) string {
	switch {
	case strings.HasPrefix(str, "[]"):
		return "repeated " + ParseType(str[2:])
	case strings.HasPrefix(str, "&"), strings.HasPrefix(str, "*"):
		return ParseType(str[1:])
	default:
		return GetProtoType(str)
	}
}

// GetStructImports 获取结构体引用的外部proto文件列表
func (ctx *ParseContext) GetStructImports() []string {
	imports := make(map[string]bool)

	for _, st := range ctx.Structs {
		for _, field := range st.FieldList {
			ft := field.Type
			if strings.HasPrefix(ft, "repeated ") {
				ft = strings.TrimPrefix(ft, "repeated ")
			}
			if IsBuiltinType(ft) {
				continue
			}
			if _, ok := ctx.EnumMap[ft]; ok {
				imports["enum.gen.proto"] = true
				continue
			}
			if _, ok := ctx.StructMap[ft]; ok {
				continue
			}
			if pp, ok := ctx.Registry.Get(ft); ok {
				if pp.Filename != "enum.gen.proto" && pp.Filename != "table.gen.proto" {
					imports[pp.Filename] = true
				}
			}
		}
	}

	rets := make([]string, 0, len(imports))
	for imp := range imports {
		rets = append(rets, imp)
	}
	return rets
}

var builtinTypes = map[string]bool{
	"double": true, "float": true, "int32": true, "int64": true,
	"uint32": true, "uint64": true, "sint32": true, "sint64": true,
	"fixed32": true, "fixed64": true, "sfixed32": true, "sfixed64": true,
	"bool": true, "string": true, "bytes": true,
}

// IsBuiltinType 是否是protobuf内置类型
func IsBuiltinType(typ string) bool {
	return builtinTypes[typ]
}
