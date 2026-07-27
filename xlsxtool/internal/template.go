package internal

const ProtoHeadTempl = `
/*本协议由xlsxtool工具生成，请勿手动修改！！！*/
syntax = "proto3";

package %s;

option go_package = "%s";

`

const ProtoImportTempl = `
{{- range $item := .}}
import "{{$item}}";
{{- end}}
`

const ProtoEnumTempl = `
{{- range $enum := .}}
enum {{$enum.Type}} {
	{{$enum.Type}}_None = 0; // 工具生成的默认值
{{- range $item := $enum.Values}}
	{{$item.Name}} = {{$item.Value}}; // {{$item.Desc}}
{{- end}}
}
{{end -}}
`

const ProtoStructTempl = `
{{- range $st := .}}
message {{$st.Type}} {
{{- range $item := $st.FieldList}}
	{{$item.Type}} {{$item.Name}} = {{$item.Position}}; // {{$item.Desc}}
{{- end}}
}

message {{$st.Type}}Ary {
	repeated {{$st.Type}} Ary = 1;
}
{{end}}
`

const ConfigCodeTempl = `
/*
* 本代码由xlsxtool工具生成，请勿手动修改
*/

{{$type := .Type}}

package {{ToSnakePkg $type}}

import (
	"google.golang.org/protobuf/proto"
)

const (
	SHEET_NAME = "{{$type}}"
)

var obj = atomic.Pointer[{{$type}}Data]{}


type {{$type}}Data struct {
	list []*pb.{{$type}}S
{{- range $index := .IndexList}}
{{- if ne $index.Type "range"}}
{{- if $index.IsNestedMap}}
	{{$index.CompositeFieldName}} {{compositeContainerType $index $type}} // 嵌套map容器
{{- else}}
	{{ToLowerCamel $index.Name}} {{containerType $index $type}}
{{- end}}
{{- end}}
{{- end}}
}

func init() {
	fwatcher.RegisterParser(SHEET_NAME, parse)
}

func Change(f func()) {
	fwatcher.RegisterChange(SHEET_NAME, f)
}

func parse(ary *pb.{{$type}}Ary) error {
	data := &{{$type}}Data{
{{- range $index := .IndexList}}
{{- if ne $index.Type "range"}}
{{- if $index.IsNestedMap}}
		{{$index.CompositeFieldName}}: make({{compositeContainerType $index $type}}),
{{- else}}
		{{ToLowerCamel $index.Name}}: make({{containerType $index $type}}),
{{- end}}
{{- end}}
{{- end}}
	}
	for _, item := range ary.Ary {
		data.list = append(data.list, item.ToS())
{{- range $index := .IndexList}}
{{- if ne $index.Type "range"}}
{{- if $index.IsNestedMap}}
{{- if (eq $index.Next.Type "map")}}
		// 嵌套map容器: {{$index.CompositeFieldName}}
		if _, ok := data.{{$index.CompositeFieldName}}[{{keyExpr "item" $index}}]; !ok {
			data.{{$index.CompositeFieldName}}[{{keyExpr "item" $index}}] = make({{compositeInnerContainerType $index $type}})
		}
		data.{{$index.CompositeFieldName}}[{{keyExpr "item" $index}}][{{keyExpr "item" $index.Next}}] = item.ToS()
{{- else}}
		// 外层map+内层slice容器: {{ToLowerCamel $index.Name}}
		if _, ok := data.{{ToLowerCamel $index.Name}}[{{keyExpr "item" $index}}]; !ok {
			data.{{ToLowerCamel $index.Name}}[{{keyExpr "item" $index}}] = []*pb.{{$type}}S{}
		}
		data.{{ToLowerCamel $index.Name}}[{{keyExpr "item" $index}}] = append(data.{{ToLowerCamel $index.Name}}[{{keyExpr "item" $index}}], item.ToS())
{{- end}}
{{- else}}
		{{if eq $index.Type "group"}}data.{{ToLowerCamel $index.Name}}[{{keyExpr "item" $index}}] = append(data.{{ToLowerCamel $index.Name}}[{{keyExpr "item" $index}}], item.ToS())
		{{else}}data.{{ToLowerCamel $index.Name}}[{{keyExpr "item" $index}}] = item.ToS()
		{{end}}
{{- end}}
{{- end}}
{{- end}}
	}
	obj.Store(data)
	return nil
}

func SGet(pos int) *pb.{{$type}}S {
	data := obj.Load()
	if data == nil {
		return nil
	}
	list := data.list
	if len(list) == 0 {
		return nil
	}
	if pos < 0 {
		pos = 0
	}
	if ll := len(list); ll-1 < pos {
		pos = ll-1
	}
	return list[pos]
}

func LGet() (rets []*pb.{{$type}}S) {
	data := obj.Load()
	if data == nil {
		return nil
	}
	list := data.list
	rets = make([]*pb.{{$type}}S, len(list))
	copy(rets, list)
	return
}

func Walk(f func(*pb.{{$type}}S)bool) {
	data := obj.Load()
	if data == nil {
		return
	}
	for _, item := range data.list {
		if !f(item) {
			return
		}
	}
}

{{range $index := .IndexList}}
{{if eq $index.Type "map"}}		{{/* map类型 */}}
{{if $index.IsNext}}	{{/* map@group 或 map@map 组合类型 */}}
{{if eq $index.Next.Type "group"}}		{{/* map@group: 先精确查记录，返回分组列表 */}}
func MGet{{$index.Name}}{{$index.CompositeNameSuffix}}({{$index.GetArg true}}) []*pb.{{$type}}S {
	data := obj.Load()
	if data == nil {
		return nil
	}
	src := data.{{ToLowerCamel $index.Name}}[{{keyExpr "" $index}}]
	if src == nil {
		return nil
	}
	rets := make([]*pb.{{$type}}S, len(src))
	copy(rets, src)
	return rets
}

{{else if eq $index.Next.Type "map"}}		{{/* map@map: 两级精确查找 */}}
func MGet{{$index.Name}}{{$index.CompositeNameSuffix}}({{$index.GetArg true}}) *pb.{{$type}}S {
	data := obj.Load()
	if data == nil {
		return nil
	}
	innerMap := data.{{$index.CompositeFieldName}}[{{keyExpr "" $index}}]
	if innerMap == nil {
		return nil
	}
	item := innerMap[{{keyExpr "" $index.Next}}]
	if item == nil {
		return nil
	}
	return item
}

{{end}}
{{else}}
func MGet{{$index.Name}}({{$index.GetArg false}}) *pb.{{$type}}S {
	data := obj.Load()
	if data == nil {
		return nil
	}
	item := data.{{ToLowerCamel $index.Name}}[{{keyExpr "" $index}}]
	if item == nil {
		return nil
	}
	return item
}

{{end}}

{{else if eq $index.Type "group"}}		{{/* group类型 */}}
{{if $index.IsNext}}	{{/* group@range 或 group@map 组合类型 */}}
{{if eq $index.Next.Type "range"}}		{{/* group@range: 先分组再二分查找 */}}
func GGet{{$index.Name}}{{$index.CompositeNameSuffix}}({{$index.GetArg true}}) *pb.{{$type}}S {
	data := obj.Load()
	if data == nil {
		return nil
	}
	items, ok := data.{{ToLowerCamel $index.Name}}[{{keyExpr "" $index}}]
	if !ok {
		return nil
	}
{{compositeSearchExpr $index}}
{{if eq (len $index.Next.List) 1}}	idx := {{compositeFieldNames $index}} - 1
{{else}}	idx := util.Min({{compositeFieldNames $index}}) - 1
{{end}}
	if idx < 0 {
		return nil
	}
	return items[idx]
}

{{else if eq $index.Next.Type "map"}}		{{/* group@map: 先分组再精确查找 */}}
func GGet{{$index.Name}}{{$index.CompositeNameSuffix}}({{$index.GetArg true}}) *pb.{{$type}}S {
	data := obj.Load()
	if data == nil {
		return nil
	}
	innerMap, ok := data.{{$index.CompositeFieldName}}[{{keyExpr "" $index}}]
	if !ok {
		return nil
	}
	item := innerMap[{{keyExpr "" $index.Next}}]
	if item == nil {
		return nil
	}
	return item
}

{{end}}
{{/* 基础 group 方法始终生成 */}}
func GGet{{$index.Name}}({{$index.GetArg false}}) []*pb.{{$type}}S {
	data := obj.Load()
	if data == nil {
		return nil
	}
	src := data.{{ToLowerCamel $index.Name}}[{{keyExpr "" $index}}]
	if src == nil {
		return nil
	}
	rets := make([]*pb.{{$type}}S, len(src))
	copy(rets, src)
	return rets
}

func GWalk{{$index.Name}}({{$index.GetArg false}}, f func(*pb.{{$type}}S)bool) {
	data := obj.Load()
	if data == nil {
		return
	}
	src := data.{{ToLowerCamel $index.Name}}[{{keyExpr "" $index}}]
	for _, item := range src {
		if !f(item) {
			return
		}
	}
}

{{else}}
func GGet{{$index.Name}}({{$index.GetArg false}}) []*pb.{{$type}}S {
	data := obj.Load()
	if data == nil {
		return nil
	}
	src := data.{{ToLowerCamel $index.Name}}[{{keyExpr "" $index}}]
	if src == nil {
		return nil
	}
	rets := make([]*pb.{{$type}}S, len(src))
	copy(rets, src)
	return rets
}

func GWalk{{$index.Name}}({{$index.GetArg false}}, f func(*pb.{{$type}}S)bool) {
	data := obj.Load()
	if data == nil {
		return
	}
	src := data.{{ToLowerCamel $index.Name}}[{{keyExpr "" $index}}]
	for _, item := range src {
		if !f(item) {
			return
		}
	}
}

{{end}}

{{else if eq $index.Type "range"}}		{{/* range类型：二分查找，返回所有字段均<=给定值的最大配置 */}}
func Range{{$index.Name}}({{$index.GetArg false}}) *pb.{{$type}}S {
	data := obj.Load()
	if data == nil {
		return nil
	}
	list := data.list
	if len(list) == 0 {
		return nil
	}
{{rangeSearchExpr $index}}
{{if eq (len $index.List) 1}}	idx := {{rangeFieldNames $index}} - 1
{{else}}	idx := util.Min({{rangeFieldNames $index}}) - 1
{{end}}
	if idx < 0 {
		return nil
	}
	return list[idx]
}

{{end}}
{{end}}
`
