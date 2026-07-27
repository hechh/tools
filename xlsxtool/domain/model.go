package domain

import (
	"strings"

	"github.com/iancoleman/strcase"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

// EValue 枚举值
type EValue struct {
	Type  string
	Name  string
	Value int32
	Desc  string
}

// Enum 枚举定义
type Enum struct {
	Type    string
	Values  []*EValue
	DescMap map[string]*EValue
}

// Add 添加枚举值
func (e *Enum) Add(desc, name string, value int32) {
	ev := &EValue{Type: e.Type, Name: name, Value: value, Desc: desc}
	e.Values = append(e.Values, ev)
	e.DescMap[desc] = ev
}

// Field 结构体字段
type Field struct {
	Name       string
	Type       string
	OriginType string // 原始类型,用于数据转换
	ProtoType  string // Proto类型,用于数据转换
	Desc       string
	Position   int32
	Descriptor protoreflect.FieldDescriptor
}

// Index 结构体索引
type Index struct {
	Type string
	Name string
	List []*Field
	Next *Index
}

// Struct 结构体定义
type Struct struct {
	Type          string
	AryDescriptor protoreflect.MessageDescriptor
	Descriptor    protoreflect.MessageDescriptor
	FieldList     []*Field
	FieldMap      map[string]*Field
	IndexList     []*Index
	IndexMap      map[string]*Index
	Rows          [][]string
}

// Table Excel表格原始数据
type Table struct {
	Sheet string
	Type  string
	Rules []string
	Rows  [][]string
	Token uint32
}

// Proto Proto类型元信息
type Proto struct {
	Type     string
	Pkg      string
	GoPkg    string
	Filename string
	IsEnum   bool
}

// TypeConverter 类型转换器
type TypeConverter struct {
	Target string
	Proto  string
	Origin string
	Conv   ConvertFunc
}

// ConvertFunc 类型转换函数
type ConvertFunc func(val string, field protoreflect.FieldDescriptor, msg *dynamicpb.Message) error

// FieldParam 将单个字段转为参数声明字符串: "name Type" 或 "name pb.EnumType"
func (f *Field) FieldParam() string {
	name := strcase.ToLowerCamel(f.Name)
	if f.Descriptor != nil && f.Descriptor.Kind() == protoreflect.EnumKind {
		return name + " pb." + f.Type
	}
	return name + " " + f.Type
}

// FieldType 返回字段的 Go 类型字符串: "int32" 或 "pb.EnumType"
func (f *Field) FieldType() string {
	if f.Descriptor != nil && f.Descriptor.Kind() == protoreflect.EnumKind {
		return "pb." + f.Type
	}
	return f.Type
}

// FieldRef 返回字段引用表达式: "ref.FieldName" 或 "fieldName"（当 ref 为空时）
func (f *Field) FieldRef(ref string) string {
	name := strcase.ToLowerCamel(f.Name)
	if len(ref) > 0 {
		return ref + "." + f.Name
	}
	return name
}

// GetArg 生成方法参数列表。
// isnext=false 仅当前层级字段；isnext=true 遍历整条链的所有字段。
func (d *Index) GetArg(isnext bool) string {
	parts := make([]string, 0, len(d.List))
	for item := d; item != nil; item = item.Next {
		for _, f := range item.List {
			if f == nil {
				continue
			}
			parts = append(parts, f.FieldParam())
		}
		if !isnext {
			break
		}
	}
	return strings.Join(parts, ", ")
}

// GetType 生成类型列表（逗号分隔），用于 Tuple 类型等场景。
// isnext=false 仅当前层级；isnext=true 遍历全链。
func (d *Index) GetType(isnext bool) string {
	parts := make([]string, 0, len(d.List))
	for item := d; item != nil; item = item.Next {
		for _, f := range item.List {
			if f == nil {
				continue
			}
			parts = append(parts, f.FieldType())
		}
		if !isnext {
			break
		}
	}
	return strings.Join(parts, ", ")
}

// GetValue 生成值表达式列表（逗号分隔）。
// ref 非空时加前缀如 "item."；isnext 控制是否遍历链。
func (d *Index) GetValue(isnext bool, ref string) string {
	parts := make([]string, 0, len(d.List))
	for item := d; item != nil; item = item.Next {
		for _, f := range item.List {
			if f == nil {
				continue
			}
			parts = append(parts, f.FieldRef(ref))
		}
		if !isnext {
			break
		}
	}
	return strings.Join(parts, ", ")
}

// IsNext 判断是否有下一级索引（组合规则）。
func (d *Index) IsNext() bool { return d.Next != nil }

// IsNestedMap 判断是否为嵌套 map 容器类型（内层是 map，需独立字段名避免与基础容器冲突）。
// group@range 复用基础 group 容器，不需要特殊处理。
func (d *Index) IsNestedMap() bool {
	return d.IsNext() && strings.ToLower(d.Next.Type) == "map"
}

// CompositeFieldName 嵌套 map 容器的字段名，带 "Map" 后缀避免冲突。
// 例：configId → configIdMap。非嵌套 map 时返回空串。
func (d *Index) CompositeFieldName() string {
	if !d.IsNestedMap() {
		return ""
	}
	return strcase.ToLowerCamel(d.Name) + "Map"
}

// CompositeNameSuffix 组合规则的方法名后缀。
// 例：group@range → RangeLevelEnergy；group@map → MapSubId
func (d *Index) CompositeNameSuffix() string {
	if d.Next == nil {
		return ""
	}
	var prefix string
	switch strings.ToLower(d.Next.Type) {
	case "range":
		prefix = "Range"
	case "map":
		prefix = "Map"
	case "group":
		prefix = "Group"
	default:
		prefix = strcase.ToCamel(d.Next.Type)
	}
	var sb strings.Builder
	sb.WriteString(prefix)
	for _, f := range d.Next.List {
		if f != nil {
			sb.WriteString(f.Name)
		}
	}
	return sb.String()
}
