package app

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/hechh/tools/xlsxtool/domain"
	"github.com/hechh/tools/xlsxtool/internal"

	"github.com/iancoleman/strcase"
	"golang.org/x/tools/imports"
)

// GenCode 生成 Go 查询代码（common/table/<snake>/XxxConfig.gen.go）
func GenCode(ctx *domain.ParseContext, dstDir string, save func(string, string, []byte) error) error {
	resolveEnumFlags(ctx)

	tpl, err := template.New("templ").Funcs(template.FuncMap{
		"ToSnake":                     strcase.ToSnake,
		"ToSnakePkg":                  func(s string) string { return strcase.ToSnake(s) },
		"ToLowerCamel":                strcase.ToLowerCamel,
		"containerType":               containerType,
		"keyExpr":                     keyExpr,
		"rangeSearchExpr":             rangeSearchExpr,
		"rangeFieldNames":             rangeFieldNames,
		"compositeSearchExpr":         compositeSearchExpr,
		"compositeFieldNames":         compositeFieldNames,
		"compositeInnerContainerType": compositeInnerContainerType,
		"compositeContainerType":      compositeContainerType,
	}).Parse(internal.ConfigCodeTempl)
	if err != nil {
		return err
	}

	for _, item := range ctx.Structs {
		pkgname := strcase.ToSnake(item.Type)
		buf := bytes.NewBuffer(nil)
		if err := tpl.Execute(buf, item); err != nil {
			return err
		}

		filename := filepath.Join(dstDir, pkgname, item.Type+".gen.go")
		processed, err := imports.Process(filename, buf.Bytes(), nil)
		if err != nil {
			return fmt.Errorf("imports.Process %s: %w", filename, err)
		}
		outDir := path.Join(dstDir, pkgname)
		if err := os.MkdirAll(outDir, 0o755); err != nil {
			return err
		}
		if err := save(outDir, item.Type+".gen.go", processed); err != nil {
			return err
		}
		fmt.Printf("[OK] 生成: %s\n", filename)
	}
	return nil
}

// rangeSearchExpr 生成 sort.Search 代码块（基础 range 索引用 list 变量）
func rangeSearchExpr(idx *domain.Index) string {
	var sb strings.Builder
	for _, f := range idx.List {
		if f == nil {
			continue
		}
		name := strcase.ToLowerCamel(f.Name)
		fmt.Fprintf(&sb, "\tsearch_%s := sort.Search(len(list), func(i int) bool {\n", name)
		fmt.Fprintf(&sb, "\t\treturn list[i].%s > %s\n", f.Name, name)
		sb.WriteString("\t})\n")
	}
	return sb.String()
}

// rangeFieldNames 返回 search_xxx 变量名列表（逗号分隔）
func rangeFieldNames(idx *domain.Index) string {
	strs := make([]string, 0, len(idx.List))
	for _, f := range idx.List {
		if f != nil {
			strs = append(strs, "search_"+strcase.ToLowerCamel(f.Name))
		}
	}
	return strings.Join(strs, ", ")
}

// containerType 基础容器类型: map[key]*type 或 map[key][]*type
func containerType(idx *domain.Index, structType string) string {
	return "map[" + indexKeyType(idx) + "]" + indexValueType(idx, structType)
}

// keyExpr key 表达式：多 key 用 tuple.TupleN(...)，单 key 直接返回值
func keyExpr(ref string, idx *domain.Index) string {
	if len(idx.List) > 1 {
		parts := make([]string, 0, len(idx.List))
		for _, f := range idx.List {
			if f == nil {
				continue
			}
			parts = append(parts, f.FieldRef(ref))
		}
		return fmt.Sprintf("tuple.T%d(%s)", len(parts), strings.Join(parts, ","))
	}
	return idx.GetValue(false, ref)
}

// indexKeyType map 的 key 类型：单 key 为字段类型，多 key 为 tuple.TupleN[T,...]
func indexKeyType(idx *domain.Index) string {
	if len(idx.List) > 1 {
		types := make([]string, 0, len(idx.List))
		for _, f := range idx.List {
			if f != nil {
				types = append(types, fieldGoType(f))
			}
		}
		return fmt.Sprintf("tuple.Tuple%d[%s]", len(types), strings.Join(types, ","))
	}
	if idx.List[0] == nil {
		return "interface{}"
	}
	return fieldGoType(idx.List[0])
}

// indexValueType map 的 value 类型：group 返回 []*pb.XxxS，其余返回 *pb.XxxS
func indexValueType(idx *domain.Index, structType string) string {
	if idx.Type == "group" {
		return "[]*pb." + structType + "S"
	}
	return "*pb." + structType + "S"
}

// fieldGoType 字段的 Go 类型字符串（复用 FieldType，含 repeated 枚举处理）
func fieldGoType(f *domain.Field) string {
	return f.FieldType()
}

// compositeSearchExpr 组合规则中内层 range 的 sort.Search（使用 items 变量）
func compositeSearchExpr(idx *domain.Index) string {
	if !idx.IsNext() || idx.Next.Type != "range" {
		return ""
	}
	var sb strings.Builder
	for _, f := range idx.Next.List {
		if f == nil {
			continue
		}
		name := strcase.ToLowerCamel(f.Name)
		fmt.Fprintf(&sb, "\tsearch_%s := sort.Search(len(items), func(i int) bool {\n", name)
		fmt.Fprintf(&sb, "\t\treturn items[i].%s > %s\n", f.Name, name)
		sb.WriteString("\t})\n")
	}
	return sb.String()
}

// compositeFieldNames 内层 search_ 变量名列表
func compositeFieldNames(idx *domain.Index) string {
	if idx.Next == nil {
		return ""
	}
	strs := make([]string, 0, len(idx.Next.List))
	for _, f := range idx.Next.List {
		if f != nil {
			strs = append(strs, "search_"+strcase.ToLowerCamel(f.Name))
		}
	}
	return strings.Join(strs, ", ")
}

// compositeInnerContainerType 内层容器类型: map[InnerKey]*pb.XxxS
func compositeInnerContainerType(idx *domain.Index, structType string) string {
	if idx.Next == nil {
		return ""
	}
	return "map[" + indexKeyType(idx.Next) + "]*pb." + structType + "S"
}

// compositeContainerType 外层组合容器类型:
// group@map/map@map → map[Outer]map[Inner]*pb.XxxS
// map@group/map@range → map[Outer][]*XxxS
// group@range → 复用基础 containerType
func compositeContainerType(idx *domain.Index, structType string) string {
	if !idx.IsNext() {
		return containerType(idx, structType)
	}
	outerKey := indexKeyType(idx)
	switch idx.Next.Type {
	case "map":
		return "map[" + outerKey + "]map[" + indexKeyType(idx.Next) + "]*pb." + structType + "S"
	case "group", "range":
		if idx.Type == "map" {
			return "map[" + outerKey + "][]*pb." + structType + "S"
		}
		fallthrough
	default:
		return containerType(idx, structType)
	}
}
