package domain

import (
	"bytes"
	"sync"
	"text/template"
)

// bufPool buffer池，避免每次渲染都分配新内存
var bufPool = sync.Pool{
	New: func() any { return bytes.NewBuffer(nil) },
}

// Generator 代码生成领域服务
// 职责：将领域模型(RedisString/RedisHash)通过模板渲染为Go代码
type Generator struct {
	strTpl  *template.Template
	hashTpl *template.Template
}

// NewGenerator 创建生成器，注入已解析好的模板
func NewGenerator(strTpl, hashTpl *template.Template) *Generator {
	return &Generator{strTpl: strTpl, hashTpl: hashTpl}
}

// GenerateAll 执行全部代码生成，通过OutputPort输出
func (g *Generator) GenerateAll(ctx *ParseContext, out OutputPort) error {
	for _, item := range ctx.Strings {
		if err := g.generateOne(g.strTpl, out, item.Pkg, item.Name, item); err != nil {
			return err
		}
	}
	for _, item := range ctx.Hashs {
		if err := g.generateOne(g.hashTpl, out, item.Pkg, item.Name, item); err != nil {
			return err
		}
	}
	return nil
}

// generateOne 渲染单个模型并输出
func (g *Generator) generateOne(tpl *template.Template, out OutputPort, pkg, name string, model any) error {
	content, err := render(tpl, model)
	if err != nil {
		return err
	}
	filename := pkg + "/" + name + ".gen.go"
	return out.Write(filename, content)
}

// render 执行模板渲染（使用buffer池优化内存）
func render(tpl *template.Template, data any) ([]byte, error) {
	buf := bufPool.Get().(*bytes.Buffer)
	defer func() { buf.Reset(); bufPool.Put(buf) }()

	if err := tpl.Execute(buf, data); err != nil {
		return nil, err
	}
	result := make([]byte, buf.Len())
	copy(result, buf.Bytes())
	return result, nil
}
