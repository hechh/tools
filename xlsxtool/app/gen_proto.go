package app

import (
	"bytes"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/template"

	"github.com/hechh/tools/xlsxtool/domain"
	"github.com/hechh/tools/xlsxtool/internal"
)

// GenProto proto生成用例
func GenProto(ctx *domain.ParseContext, dstDir string, save func(string, string, []byte) error) error {
	// 生成 enum.gen.proto
	if len(ctx.Enums) > 0 {
		for _, item := range ctx.Enums {
			sort.Slice(item.Values, func(i, j int) bool {
				return item.Values[i].Value < item.Values[j].Value
			})
		}
		sort.Slice(ctx.Enums, func(i, j int) bool {
			return strings.Compare(ctx.Enums[i].Type, ctx.Enums[j].Type) <= 0
		})

		buf := bytes.NewBuffer(nil)
		fmt.Fprintf(buf, internal.ProtoHeadTempl, ctx.Registry.Pkg, ctx.Registry.GoPkg)
		if err := executeTemplate("enum", internal.ProtoEnumTempl, buf, ctx.Enums); err != nil {
			return err
		}
		if err := save(dstDir, "enum.gen.proto", buf.Bytes()); err != nil {
			return err
		}
		fmt.Fprintf(os.Stdout, "[OK] 生成: %s/enum.gen.proto\n", dstDir)
	}

	// 生成 table.gen.proto
	if len(ctx.Structs) > 0 {
		for _, item := range ctx.Structs {
			sort.Slice(item.FieldList, func(i, j int) bool {
				return item.FieldList[i].Position < item.FieldList[j].Position
			})
		}
		sort.Slice(ctx.Structs, func(i, j int) bool {
			return strings.Compare(ctx.Structs[i].Type, ctx.Structs[j].Type) <= 0
		})

		buf := bytes.NewBuffer(nil)
		fmt.Fprintf(buf, internal.ProtoHeadTempl, ctx.Registry.Pkg, ctx.Registry.GoPkg)

		if imports := ctx.GetStructImports(); len(imports) > 0 {
			sort.Strings(imports)
			if err := executeTemplate("import", internal.ProtoImportTempl, buf, imports); err != nil {
				return err
			}
		}

		if err := executeTemplate("struct", internal.ProtoStructTempl, buf, ctx.Structs); err != nil {
			return err
		}
		if err := save(dstDir, "table.gen.proto", buf.Bytes()); err != nil {
			return err
		}
		fmt.Fprintf(os.Stdout, "[OK] 生成: %s/table.gen.proto\n", dstDir)
	}
	return nil
}

func executeTemplate(name, text string, buf *bytes.Buffer, data any) error {
	tpl, err := template.New(name).Parse(text)
	if err != nil {
		return err
	}
	return tpl.Execute(buf, data)
}
