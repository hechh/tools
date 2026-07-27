package internal

import (
	"bytes"
	"go/ast"
	"path/filepath"
	"strings"
	"text/template"

	futil "github.com/hechh/library/base/fileutil"
)

type StructDescriptor struct {
	Name string
	TypeDescriptor
}

type EnumDescriptor struct {
	Name string
	TypeDescriptor
}

type Parser struct {
	pkgName string
	list    []*StructDescriptor
	enums   []*EnumDescriptor
}

func (p *Parser) Visit(n ast.Node) ast.Visitor {
	switch vv := n.(type) {
	case *ast.File:
		p.pkgName = vv.Name.Name
		return p
	case *ast.GenDecl:
		return p
	case *ast.TypeSpec:
		switch vv.Type.(type) {
		case *ast.StructType:
			item := ParseType(vv.Type)
			p.list = append(p.list, &StructDescriptor{
				Name:           vv.Name.Name,
				TypeDescriptor: item,
			})
		case *ast.Ident:
			item := ParseType(vv.Type)
			p.enums = append(p.enums, &EnumDescriptor{
				Name:           vv.Name.Name,
				TypeDescriptor: item,
			})
		}
		return nil
	}
	return nil
}

func (p *Parser) GetPkgName() string {
	return p.pkgName
}

func (p *Parser) GetAllEnum() []*EnumDescriptor {
	return p.enums
}

func (p *Parser) GetAllStruct() (rets []*StructDescriptor) {
	for _, item := range p.list {
		if strings.HasSuffix(item.Name, "Rsp") ||
			strings.HasSuffix(item.Name, "Req") ||
			strings.HasSuffix(item.Name, "ConfigS") ||
			strings.HasSuffix(item.Name, "Config") ||
			strings.HasSuffix(item.Name, "ConfigAry") {
			continue
		}
		rets = append(rets, item)
	}
	return
}

func (p *Parser) GetAllRsp() (rets []*StructDescriptor) {
	for _, item := range p.list {
		if strings.HasSuffix(item.Name, "Rsp") {
			rets = append(rets, item)
		}
	}
	return
}

// GetAllConfig 返回所有以 Config 结尾（但不以 ConfigAry 或 ConfigS 结尾）的配置结构体
func (p *Parser) GetAllConfig() (rets []*StructDescriptor) {
	for _, item := range p.list {
		if strings.HasSuffix(item.Name, "Config") &&
			!strings.HasSuffix(item.Name, "ConfigAry") &&
			!strings.HasSuffix(item.Name, "ConfigS") {
			rets = append(rets, item)
		}
	}
	return
}

// isRewardType 递归检查类型描述符是否为 *Reward 或 []*Reward
func isRewardType(desc TypeDescriptor) (isReward bool, isSlice bool) {
	inner := desc
	for {
		switch inner.Kind() {
		case KindSlice:
			isSlice = true
			elems := inner.Elements()
			if len(elems) == 0 {
				return false, false
			}
			inner = elems[0]
		case KindPointer:
			elems := inner.Elements()
			if len(elems) == 0 {
				return false, false
			}
			inner = elems[0]
		default:
			// 检查选择器类型 (pkg.Reward)
			if inner.Kind() == KindSelector {
				if sel, ok := inner.(*SelectorTypeDescriptor); ok {
					return sel.Sel == "Reward", isSlice
				}
			}
			// 检查裸标识符类型 (Reward，无包前缀)
			if inner.Kind() == KindBasic {
				if ident, ok := inner.(*IdentTypeDescriptor); ok {
					return ident.IdentName == "Reward", isSlice
				}
			}
			return false, false
		}
	}
}

func (p *Parser) Gen(dst string) error {
	funcMap := template.FuncMap{
		"hasSuffix":     strings.HasSuffix,
		"isExported":    func(name string) bool { return len(name) > 0 && name[0] >= 'A' && name[0] <= 'Z' },
		"isRewardField": func(m Member) bool { ok, _ := isRewardType(m.Type); return ok },
		"isRewardSlice": func(m Member) bool { _, ok := isRewardType(m.Type); return ok },
		"memberType":    func(m Member) string { return m.Type.Name() },
	}
	tplObj := template.Must(template.New("pb").Funcs(funcMap).Parse(templ))
	buf := bytes.NewBuffer(nil)
	tplObj.Execute(buf, p)
	return futil.Save(filepath.Join(dst, "common.gen.pb.go"), buf.Bytes())
}
