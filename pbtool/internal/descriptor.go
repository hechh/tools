package internal

import (
	"bytes"
	"fmt"
	"go/ast"
	"strings"
)

type TypeKind string

const (
	KindBasic     TypeKind = "basic"     // 基本类型 (int, string, bool)
	KindPointer   TypeKind = "pointer"   // 指针类型 (*T)
	KindArray     TypeKind = "array"     // 数组类型 ([N]T)
	KindSlice     TypeKind = "slice"     // 切片类型 ([]T)
	KindMap       TypeKind = "map"       // 映射类型 (map[K]V)
	KindChan      TypeKind = "chan"      // 通道类型 (chan T, <-chan T, chan<- T)
	KindFunc      TypeKind = "func"      // 函数类型 (func(...))
	KindStruct    TypeKind = "struct"    // 结构体类型 (struct{})
	KindInterface TypeKind = "interface" // 接口类型 (interface{})
	KindSelector  TypeKind = "selector"  // 选择器类型 (pkg.Type, struct.Field)
	KindParen     TypeKind = "paren"     // 括号类型 ((T))
	KindEllipsis  TypeKind = "ellipsis"  // 不定参数 (...)
	KindGeneric   TypeKind = "generic"   // 泛型实例化类型 (List[T])
	KindUnknown   TypeKind = "unknown"   // 未知类型
)

type ChanDir string

const (
	ChanDirSend ChanDir = "send" // 只发送 (chan<- T)
	ChanDirRecv ChanDir = "recv" // 只接收 (<-chan T)
	ChanDirBoth ChanDir = "both" // 双向 (chan T)
)

// Member 表示结构体的字段或接口的方法
type Member struct {
	Name string         // 字段名或方法名
	Type TypeDescriptor // 字段类型或方法签名
	Tag  string         // 结构体字段的标签 (例如 `json:"name"`)
}

// TypeDescriptor 是描述Go类型的通用接口
type TypeDescriptor interface {
	// Name 返回类型的完整字符串表示
	Name() string
	// Kind 返回类型的基本种类
	Kind() TypeKind
	// Elements 返回构成该类型的元素类型描述符。
	// 例如，[]int 的 Elements 是 []TypeDescriptor{intDesc}
	// *int 的 Elements 是 []TypeDescriptor{intDesc}
	// map[K]V 的 Elements 是 []TypeDescriptor{KDesc, VDesc}
	Elements() []TypeDescriptor
	// Members 返回结构体的字段或接口的方法。
	// 对于非结构体和接口类型，返回 nil。
	Members() []Member
	// String 返回与 Name() 相同的字符串表示，用于 fmt 打印
	String() string
}

// BaseTypeDescriptor 提供了  TypeDescriptor 接口的基础实现
// 所有具体的类型描述符都可以嵌入它
type BaseTypeDescriptor struct {
	typeName string
	typeKind TypeKind
	elements []TypeDescriptor
	members  []Member
}

func (b *BaseTypeDescriptor) Name() string               { return b.typeName }
func (b *BaseTypeDescriptor) Kind() TypeKind             { return b.typeKind }
func (b *BaseTypeDescriptor) Elements() []TypeDescriptor { return b.elements }
func (b *BaseTypeDescriptor) Members() []Member          { return b.members }
func (b *BaseTypeDescriptor) String() string             { return b.Name() }

// 以下是具体的类型描述符结构体
// 描述基本类型 (int, string, etc.)
type IdentTypeDescriptor struct {
	BaseTypeDescriptor
	IdentName string // 标识符名称，如 "int", "string", "MyCustomType"
}

// PointerTypeDescriptor 描述指针类型 (*T)
type PointerTypeDescriptor struct {
	BaseTypeDescriptor
	Elem TypeDescriptor // 指针指向的元素类型
}

// ArrayTypeDescriptor 描述数组类型 ([N]T)
type ArrayTypeDescriptor struct {
	BaseTypeDescriptor
	LenExpr string         // 数组长度的表达式字符串，如 "5", "N"
	Elem    TypeDescriptor // 数组元素的类型
}

// SliceTypeDescriptor 描述切片类型 ([]T)
type SliceTypeDescriptor struct {
	BaseTypeDescriptor
	Elem TypeDescriptor // 切片元素的类型
}

// MapTypeDescriptor 描述映射类型 (map[K]V)
type MapTypeDescriptor struct {
	BaseTypeDescriptor
	Key   TypeDescriptor // 键类型
	Value TypeDescriptor // 值类型
}

// ChanTypeDescriptor 描述通道类型 (chan T)
type ChanTypeDescriptor struct {
	BaseTypeDescriptor
	Dir  ChanDir        // 通道方向
	Elem TypeDescriptor // 通道元素的类型
}

// FuncTypeDescriptor 描述函数类型 (func(...))
type FuncTypeDescriptor struct {
	BaseTypeDescriptor
	Params  []Member // 参数列表
	Results []Member // 返回值列表
}

// StructTypeDescriptor 描述结构体类型 (struct{})
type StructTypeDescriptor struct {
	BaseTypeDescriptor
	Fields []Member // 字段列表
}

// InterfaceTypeDescriptor 描述接口类型 (interface{})
type InterfaceTypeDescriptor struct {
	BaseTypeDescriptor
	Methods []Member // 方法列表
}

// SelectorTypeDescriptor 描述选择器类型 (pkg.Type)
type SelectorTypeDescriptor struct {
	BaseTypeDescriptor
	X   TypeDescriptor // 左侧表达式，如 pkg
	Sel string         // 选择器名称，如 Type
}

// ParenTypeDescriptor 描述括号类型 ((T))
type ParenTypeDescriptor struct {
	BaseTypeDescriptor
	Expr TypeDescriptor // 括号内的表达式
}

// EllipsisTypeDescriptor 描述不定参数 (...)
type EllipsisTypeDescriptor struct {
	BaseTypeDescriptor
	Elem TypeDescriptor // 不定参数的元素类型
}

// GenericTypeDescriptor 描述泛型实例化类型 (List[T])
type GenericTypeDescriptor struct {
	BaseTypeDescriptor
	X       TypeDescriptor   // 泛型类型，如 List
	Indices []TypeDescriptor // 类型参数，如 [T, int]
}

// ParseType 将一个 ast.Expr 解析为  TypeDescriptor
func ParseType(expr ast.Expr) TypeDescriptor {
	switch t := expr.(type) {
	case *ast.Ident:
		return ParseIdent(t)
	case *ast.StarExpr:
		return ParseStarExpr(t)
	case *ast.ArrayType:
		return ParseArrayType(t)
	case *ast.MapType:
		return ParseMapType(t)
	case *ast.ChanType:
		return ParseChanType(t)
	case *ast.FuncType:
		return ParseFuncType(t)
	case *ast.StructType:
		return ParseStructType(t)
	case *ast.InterfaceType:
		return ParseInterfaceType(t)
	case *ast.SelectorExpr:
		return ParseSelectorExpr(t)
	case *ast.ParenExpr:
		return ParseParenExpr(t)
	case *ast.Ellipsis:
		return ParseEllipsis(t)
	case *ast.IndexExpr:
		return ParseIndexExpr(t)
	case *ast.IndexListExpr:
		return ParseIndexListExpr(t)
	default:
		return &BaseTypeDescriptor{
			typeName: fmt.Sprintf("/* %T */", expr),
			typeKind: KindUnknown,
		}
	}
}

func ParseIdent(t *ast.Ident) TypeDescriptor {
	desc := &IdentTypeDescriptor{
		IdentName: t.Name,
	}
	desc.typeName = t.Name
	desc.typeKind = KindBasic
	return desc
}

func ParseStarExpr(t *ast.StarExpr) TypeDescriptor {
	elemDesc := ParseType(t.X)
	desc := &PointerTypeDescriptor{Elem: elemDesc}
	desc.typeName = "*" + elemDesc.Name()
	desc.typeKind = KindPointer
	desc.elements = []TypeDescriptor{elemDesc}
	return desc
}

func ParseArrayType(t *ast.ArrayType) TypeDescriptor {
	elemDesc := ParseType(t.Elt)
	if t.Len == nil {
		// 切片类型
		desc := &SliceTypeDescriptor{Elem: elemDesc}
		desc.typeName = "[]" + elemDesc.Name()
		desc.typeKind = KindSlice
		desc.elements = []TypeDescriptor{elemDesc}
		return desc
	} else {
		// 数组类型
		lenStr := ExprToString(t.Len)
		desc := &ArrayTypeDescriptor{
			LenExpr: lenStr,
			Elem:    elemDesc,
		}
		desc.typeName = "[" + lenStr + "]" + elemDesc.Name()
		desc.typeKind = KindArray
		desc.elements = []TypeDescriptor{elemDesc}
		return desc
	}
}

func ParseMapType(t *ast.MapType) TypeDescriptor {
	keyDesc := ParseType(t.Key)
	valueDesc := ParseType(t.Value)
	desc := &MapTypeDescriptor{Key: keyDesc, Value: valueDesc}
	desc.typeName = "map[" + keyDesc.Name() + "]" + valueDesc.Name()
	desc.typeKind = KindMap
	desc.elements = []TypeDescriptor{keyDesc, valueDesc}
	return desc
}

func ParseChanType(t *ast.ChanType) TypeDescriptor {
	elemDesc := ParseType(t.Value)
	var dir ChanDir
	var dirStr string
	switch t.Dir {
	case ast.SEND:
		dir = ChanDirSend
		dirStr = "chan<- "
	case ast.RECV:
		dir = ChanDirRecv
		dirStr = "<-chan "
	default:
		dir = ChanDirBoth
		dirStr = "chan "
	}
	desc := &ChanTypeDescriptor{Dir: dir, Elem: elemDesc}
	desc.typeName = dirStr + elemDesc.Name()
	desc.typeKind = KindChan
	desc.elements = []TypeDescriptor{elemDesc}
	return desc
}

func ParseFuncType(t *ast.FuncType) TypeDescriptor {
	params := ParseFieldList(t.Params)
	results := ParseFieldList(t.Results)

	var buf bytes.Buffer
	buf.WriteString("func(")
	for i, param := range params {
		if i > 0 {
			buf.WriteString(", ")
		}
		if param.Name != "" {
			buf.WriteString(param.Name)
			buf.WriteString(" ")
		}
		buf.WriteString(param.Type.Name())
	}
	buf.WriteString(")")

	if len(results) > 0 {
		if len(results) == 1 && results[0].Name == "" {
			buf.WriteString(" ")
			buf.WriteString(results[0].Type.Name())
		} else {
			buf.WriteString(" (")
			for i, res := range results {
				if i > 0 {
					buf.WriteString(", ")
				}
				if res.Name != "" {
					buf.WriteString(res.Name)
					buf.WriteString(" ")
				}
				buf.WriteString(res.Type.Name())
			}
			buf.WriteString(")")
		}
	}

	desc := &FuncTypeDescriptor{Params: params, Results: results}
	desc.typeName = buf.String()
	desc.typeKind = KindFunc
	// 将参数和返回值合并为 elements 以便通用访问
	allElements := make([]TypeDescriptor, 0, len(params)+len(results))
	for _, p := range params {
		allElements = append(allElements, p.Type)
	}
	for _, r := range results {
		allElements = append(allElements, r.Type)
	}
	desc.elements = allElements
	return desc
}

func ParseStructType(t *ast.StructType) TypeDescriptor {
	fields := ParseFieldList(t.Fields)
	desc := &StructTypeDescriptor{Fields: fields}

	if len(fields) == 0 {
		desc.typeName = "struct{}"
	} else {
		var buf bytes.Buffer
		buf.WriteString("struct {\n")
		for _, field := range fields {
			buf.WriteString("\t")
			if field.Name != "" {
				buf.WriteString(field.Name)
				buf.WriteString(" ")
			}
			buf.WriteString(field.Type.Name())
			if field.Tag != "" {
				buf.WriteString(" ")
				buf.WriteString(field.Tag)
			}
			buf.WriteString("\n")
		}
		buf.WriteString("}")
		desc.typeName = buf.String()
	}
	desc.typeKind = KindStruct
	desc.members = fields
	return desc
}

func ParseInterfaceType(t *ast.InterfaceType) TypeDescriptor {
	methods := ParseFieldList(t.Methods)
	desc := &InterfaceTypeDescriptor{Methods: methods}

	if len(methods) == 0 {
		desc.typeName = "interface{}"
	} else {
		var buf bytes.Buffer
		buf.WriteString("interface {\n")
		for _, method := range methods {
			buf.WriteString("\t")
			buf.WriteString(method.Name)
			buf.WriteString(method.Type.Name()) // method.Type is a FuncTypeDescriptor
			buf.WriteString("\n")
		}
		buf.WriteString("}")
		desc.typeName = buf.String()
	}
	desc.typeKind = KindInterface
	desc.members = methods
	return desc
}

func ParseSelectorExpr(t *ast.SelectorExpr) TypeDescriptor {
	xDesc := ParseType(t.X)
	desc := &SelectorTypeDescriptor{X: xDesc, Sel: t.Sel.Name}
	desc.typeName = xDesc.Name() + "." + t.Sel.Name
	desc.typeKind = KindSelector
	desc.elements = []TypeDescriptor{xDesc}
	return desc
}

func ParseParenExpr(t *ast.ParenExpr) TypeDescriptor {
	exprDesc := ParseType(t.X)
	desc := &ParenTypeDescriptor{Expr: exprDesc}
	desc.typeName = "(" + exprDesc.Name() + ")"
	desc.typeKind = KindParen
	desc.elements = []TypeDescriptor{exprDesc}
	return desc
}

func ParseEllipsis(t *ast.Ellipsis) TypeDescriptor {
	elemDesc := ParseType(t.Elt)
	desc := &EllipsisTypeDescriptor{Elem: elemDesc}
	desc.typeName = "..." + elemDesc.Name()
	desc.typeKind = KindEllipsis
	desc.elements = []TypeDescriptor{elemDesc}
	return desc
}

func ParseIndexExpr(t *ast.IndexExpr) TypeDescriptor {
	xDesc := ParseType(t.X)
	indexDesc := ParseType(t.Index)
	desc := &GenericTypeDescriptor{
		X:       xDesc,
		Indices: []TypeDescriptor{indexDesc},
	}
	desc.typeName = xDesc.Name() + "[" + indexDesc.Name() + "]"
	desc.typeKind = KindGeneric
	desc.elements = append([]TypeDescriptor{xDesc}, indexDesc)
	return desc
}

func ParseIndexListExpr(t *ast.IndexListExpr) TypeDescriptor {
	xDesc := ParseType(t.X)
	indices := make([]TypeDescriptor, len(t.Indices))
	for i, idx := range t.Indices {
		indices[i] = ParseType(idx)
	}

	indexNames := make([]string, len(indices))
	for i, idx := range indices {
		indexNames[i] = idx.Name()
	}

	desc := &GenericTypeDescriptor{
		X:       xDesc,
		Indices: indices,
	}
	desc.typeName = xDesc.Name() + "[" + strings.Join(indexNames, ", ") + "]"
	desc.typeKind = KindGeneric
	desc.elements = append([]TypeDescriptor{xDesc}, indices...)
	return desc
}

// parseFieldList 将 ast.FieldList 解析为 [] Member
func ParseFieldList(fieldList *ast.FieldList) []Member {
	if fieldList == nil {
		return nil
	}
	members := make([]Member, 0, len(fieldList.List))
	for _, field := range fieldList.List {
		fieldType := ParseType(field.Type)
		tag := ""
		if field.Tag != nil {
			tag = field.Tag.Value
		}
		if len(field.Names) == 0 {
			// 嵌入字段或接口方法
			members = append(members, Member{
				Type: fieldType,
				Tag:  tag,
			})
		} else {
			for _, name := range field.Names {
				members = append(members, Member{
					Name: name.Name,
					Type: fieldType,
					Tag:  tag,
				})
			}
		}
	}
	return members
}

// ExprToString 是你提供的原始函数，用于在解析过程中获取表达式的字符串表示
// (例如，数组长度表达式)
func ExprToString(expr ast.Expr) string {
	// ... (此处粘贴你提供的 ExprToString 函数的完整实现)
	// 为了保持代码完整性，我在这里再次包含它
	if expr == nil {
		return "<nil>"
	}

	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + ExprToString(t.X)
	case *ast.ArrayType:
		if t.Len == nil {
			return "[]" + ExprToString(t.Elt)
		}
		switch lenExpr := t.Len.(type) {
		case *ast.BasicLit:
			return "[" + lenExpr.Value + "]" + ExprToString(t.Elt)
		case *ast.Ident:
			return "[" + lenExpr.Name + "]" + ExprToString(t.Elt)
		default:
			return "[?]" + ExprToString(t.Elt)
		}
	case *ast.MapType:
		return "map[" + ExprToString(t.Key) + "]" + ExprToString(t.Value)
	case *ast.ChanType:
		var prefix string
		switch t.Dir {
		case ast.SEND:
			prefix = "chan<- "
		case ast.RECV:
			prefix = "<-chan "
		default:
			prefix = "chan "
		}
		return prefix + ExprToString(t.Value)
	case *ast.FuncType:
		var buf strings.Builder
		buf.WriteString("func(")
		if t.Params != nil {
			for i, field := range t.Params.List {
				if i > 0 {
					buf.WriteString(", ")
				}
				if len(field.Names) > 0 {
					names := make([]string, len(field.Names))
					for j, name := range field.Names {
						names[j] = name.Name
					}
					buf.WriteString(strings.Join(names, ", "))
					buf.WriteString(" ")
				}
				buf.WriteString(ExprToString(field.Type))
			}
		}
		buf.WriteString(")")
		if t.Results != nil && len(t.Results.List) > 0 {
			if len(t.Results.List) == 1 && len(t.Results.List[0].Names) == 0 {
				buf.WriteString(" ")
				buf.WriteString(ExprToString(t.Results.List[0].Type))
			} else {
				buf.WriteString(" (")
				for i, field := range t.Results.List {
					if i > 0 {
						buf.WriteString(", ")
					}
					if len(field.Names) > 0 {
						names := make([]string, len(field.Names))
						for j, name := range field.Names {
							names[j] = name.Name
						}
						buf.WriteString(strings.Join(names, ", "))
						buf.WriteString(" ")
					}
					buf.WriteString(ExprToString(field.Type))
				}
				buf.WriteString(")")
			}
		}
		return buf.String()
	case *ast.StructType:
		if t.Fields == nil || len(t.Fields.List) == 0 {
			return "struct{}"
		}
		var buf strings.Builder
		buf.WriteString("struct {\n")
		for _, field := range t.Fields.List {
			buf.WriteString("\t")
			if len(field.Names) > 0 {
				names := make([]string, len(field.Names))
				for j, name := range field.Names {
					names[j] = name.Name
				}
				buf.WriteString(strings.Join(names, ", "))
				buf.WriteString(" ")
			}
			buf.WriteString(ExprToString(field.Type))
			if field.Tag != nil {
				buf.WriteString(" ")
				buf.WriteString(field.Tag.Value)
			}
			buf.WriteString("\n")
		}
		buf.WriteString("}")
		return buf.String()
	case *ast.InterfaceType:
		if t.Methods == nil || len(t.Methods.List) == 0 {
			return "interface{}"
		}
		var buf strings.Builder
		buf.WriteString("interface {\n")
		for _, method := range t.Methods.List {
			buf.WriteString("\t")
			if len(method.Names) > 0 {
				buf.WriteString(method.Names[0].Name)
				buf.WriteString(ExprToString(method.Type))
			} else {
				buf.WriteString(ExprToString(method.Type))
			}
			buf.WriteString("\n")
		}
		buf.WriteString("}")
		return buf.String()
	case *ast.SelectorExpr:
		return ExprToString(t.X) + "." + t.Sel.Name
	case *ast.ParenExpr:
		return "(" + ExprToString(t.X) + ")"
	case *ast.Ellipsis:
		return "..." + ExprToString(t.Elt)
	case *ast.IndexExpr:
		return ExprToString(t.X) + "[" + ExprToString(t.Index) + "]"
	case *ast.IndexListExpr:
		indices := make([]string, len(t.Indices))
		for i, index := range t.Indices {
			indices[i] = ExprToString(index)
		}
		return ExprToString(t.X) + "[" + strings.Join(indices, ", ") + "]"
	case *ast.BasicLit:
		return t.Value
	default:
		return fmt.Sprintf("/* %T */", expr)
	}
}
