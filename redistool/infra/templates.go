package infra

import (
	"fmt"
	"strings"
	"text/template"

	"github.com/hechh/tools/redistool/domain"
)

const StringTemplStr = `/*
* 本代码由redistool工具生成，请勿手动修改
*/

package {{.Pkg}}

import (
	"fmt"
	"richgame/common/pb"
	"time"

	"github.com/hechh/framework/define"
	"github.com/hechh/library/base/logic"
	"github.com/hechh/library/base/safe"
	"github.com/hechh/library/redispool"
	"github.com/hechh/library/redispool/datatype"
)
{{if .HasDbConst}}
const DBNAME = "{{.DbName}}"
{{end}}
var DataType = datatype.NewDataType[pb.{{.Name}}](
	{{.ClientFuncRef}},
	{{.DataTypeFlags}},
)`

const StringMethods = `

func ST({{GetArgs .Keys}}) redispool.IData {
	return datatype.S{{fieldCount .Keys}}(DataType, GetKey, {{.ClientArg}}{{if .Keys}}, {{GetValues .Keys}}{{end}})
}

func GetKey({{GetArgs .Keys}}) string {
{{- if .Keys}}
	return fmt.Sprintf("{{.Format}}", {{.GetKeyFmtArgs}})
{{- else}}
	return "{{.Format}}"
{{- end}}
}

func Get({{GetArgs .Keys}}) (*pb.{{.Name}}, bool, error) {
	client := DataType.GetClient({{.ClientArg}})
	if client == nil {
		return nil, false, fmt.Errorf("{{.DbErrorHint}}数据库不存在")
	}
	key := GetKey({{.GetKeyCallArgs}})
	body, err := client.Get(key)
	if err != nil {
		return nil, false, err
	}
	item := &pb.{{.Name}}{}
	if len(body) > 0 {
		if err := item.UnmarshalVT(safe.StringToBytes(body)); err != nil {
			return nil, false, err
		}
	}
	return item, len(body) <= 0, nil
}

func Set({{GetArgs .Keys}}, val *pb.{{.Name}}, expiration time.Duration) error {
	client := DataType.GetClient({{.ClientArg}})
	if client == nil {
		return fmt.Errorf("{{.DbErrorHint}}数据库不存在")
	}
	key := GetKey({{.GetKeyCallArgs}})
	buf, err := val.MarshalVT()
	if err != nil {
		return err
	}
	return client.Set(key, safe.BytesToString(buf), expiration)
}

func Del({{GetArgs .Keys}}) error {
	client := DataType.GetClient({{.ClientArg}})
	if client == nil {
		return fmt.Errorf("{{.DbErrorHint}}数据库不存在")
	}
	key := GetKey({{.GetKeyCallArgs}})
	_, err := client.Del(key)
	return err
}

func GetByCache(ctx define.IContext{{if NotUidArgs .Keys}}, {{NotUidArgs .Keys}}{{end}}) (*pb.{{.Name}}, bool, error) {
	key := GetKey({{CtxCallArgs .Keys}})
	if obj, ok := ctx.GetCache(key); ok {
		return obj.(*pb.{{.Name}}), false, nil
	}
	item, isNew, err := Get({{CtxCallArgs .Keys}})
	if err != nil {
		return item, isNew, err
	}
	ctx.SetCache(key, item, {{.CacheFlag}})
	return item, isNew, nil
}

func Change(ctx define.IContext{{if NotUidArgs .Keys}}, {{NotUidArgs .Keys}}{{end}}) {
	ctx.Change(GetKey({{CtxCallArgs .Keys}}))
}

func Read(ctx define.IContext{{if NotUidArgs .Keys}}, {{NotUidArgs .Keys}}{{end}}) *pb.{{.Name}} {
	if obj, ok := ctx.GetCache(GetKey({{CtxCallArgs .Keys}})); ok {
		return obj.(*pb.{{.Name}})
	}
	return nil
}

func IsChanged(ctx define.IContext{{if NotUidArgs .Keys}}, {{NotUidArgs .Keys}}{{end}}) bool {
	return ctx.IsChanged(GetKey({{CtxCallArgs .Keys}}))
}

func Reset(ctx define.IContext{{if NotUidArgs .Keys}}, {{NotUidArgs .Keys}}{{end}}) {
	ctx.Reset(GetKey({{CtxCallArgs .Keys}}))
}
`

const HashTemplStr = `/*
* 本代码由redistool工具生成，请勿手动修改
*/

package {{.Pkg}}

import (
	"fmt"
	"richgame/common/pb"

	"github.com/hechh/framework/define"
	"github.com/hechh/library/base/logic"
	"github.com/hechh/library/base/safe"
	"github.com/hechh/library/redispool"
	"github.com/hechh/library/redispool/datatype"
)
{{if .HasDbConst}}
const DBNAME = "{{.DbName}}"
{{end}}{{if not .Keys}}
const KEY = "{{.KeyFmt}}"
{{end}}
var DataType = datatype.NewDataType[pb.{{.Name}}](
	{{.ClientFuncRef}},
	{{.DataTypeFlags}},
)`

const HashMethods = `
func HT({{GetArgs .GetHashFuncExtraParams}}) redispool.IData {
	return datatype.H{{fieldCount .Fields}}(DataType, {{if .Keys}}GetKey({{.GetKeyCallArgs}}){{else}}KEY{{end}}, GetField, {{.ClientArg}}{{if .Fields}}, {{GetValues .Fields}}{{end}})
}

{{- if .Keys}}
func GetKey({{GetArgs .Keys}}) string {
	return fmt.Sprintf("{{.KeyFmt}}", {{.GetKeyFmtArgs}})
}
{{- end}}

func GetField({{GetArgs .Fields}}) string {
{{- if .Fields}}
	return fmt.Sprintf("{{.FieldFmt}}", {{.GetFieldFmtArgs}})
{{- else}}
	return "{{.FieldFmt}}"
{{- end}}
}

func HGet({{GetArgs .GetHashFuncExtraParams}}) (*pb.{{.Name}}, bool, error) {
	{{- if .Keys}}
	key := GetKey({{.GetKeyCallArgs}})
	{{- else}}
	key := KEY
	{{- end}}
	field := GetField({{.GetFieldCallArgs}})
	client := DataType.GetClient({{.ClientArg}})
	if client == nil {
		return nil, false, fmt.Errorf("{{.DbErrorHint}}数据库不存在")
	}
	body, err := client.HGet(key, field)
	if err != nil {
		return nil, false, err
	}
	item := &pb.{{.Name}}{}
	if len(body) > 0 {
		if err := item.UnmarshalVT(safe.StringToBytes(body)); err != nil {
			return nil, false, err
		}
	}
	return item, len(body) <= 0, nil
}

func HSet({{GetArgs .GetHashFuncExtraParams}}, val *pb.{{.Name}}) error {
	{{- if .Keys}}
	key := GetKey({{.GetKeyCallArgs}})
	{{- else}}
	key := KEY
	{{- end}}
	field := GetField({{.GetFieldCallArgs}})
	client := DataType.GetClient({{.ClientArg}})
	if client == nil {
		return fmt.Errorf("{{.DbErrorHint}}数据库不存在")
	}
	buf, err := val.MarshalVT()
	if err != nil {
		return err
	}
	return client.HSet(key, field, safe.BytesToString(buf))
}

func HDel({{GetArgs .GetHashFuncExtraParams}}) error {
	client := DataType.GetClient({{.ClientArg}})
	if client == nil {
		return fmt.Errorf("{{.DbErrorHint}}数据库不存在")
	}
	{{- if .Keys}}
	key := GetKey({{.GetKeyCallArgs}})
	{{- else}}
	key := KEY
	{{- end}}
	field := GetField({{.GetFieldCallArgs}})
	_, err := client.HDel(key, field)
	return err
}

func HMGet({{if HasAnyFields .GetHashKeyParams}}{{GetArgs .GetHashKeyParams}}, {{end}}fields ...string) (map[string]*pb.{{.Name}}, error) {
	client := DataType.GetClient({{.ClientArg}})
	if client == nil {
		return nil, fmt.Errorf("{{.DbErrorHint}}数据库不存在")
	}
	{{- if .Keys}}
	key := GetKey({{.GetKeyCallArgs}})
	{{- else}}
	key := KEY
	{{- end}}
	vals, err := client.HMGet(key, fields...)
	if err != nil {
		return nil, err
	}
	result := make(map[string]*pb.{{.Name}}, len(fields))
	for i, field := range fields {
		if i >= len(vals) || vals[i] == nil {
			continue
		}
		body, ok := vals[i].(string)
		if !ok || len(body) == 0 {
			continue
		}
		item := &pb.{{.Name}}{}
		if err := item.UnmarshalVT(safe.StringToBytes(body)); err != nil {
			return nil, err
		}
		result[field] = item
	}
	return result, nil
}

func HMSet({{if HasAnyFields .GetHashKeyParams}}{{GetArgs .GetHashKeyParams}}, {{end}}data map[string]*pb.{{.Name}}) error {
	client := DataType.GetClient({{.ClientArg}})
	if client == nil {
		return fmt.Errorf("{{.DbErrorHint}}数据库不存在")
	}
	{{- if .Keys}}
	key := GetKey({{.GetKeyCallArgs}})
	{{- else}}
	key := KEY
	{{- end}}
	args := make([]any, 0, len(data)*2)
	for field, val := range data {
		buf, err := val.MarshalVT()
		if err != nil {
			return err
		}
		args = append(args, field, safe.BytesToString(buf))
	}
	return client.HMSet(key, args...)
}

func HLen({{GetArgs .Keys}}) (int64, error) {
	client := DataType.GetClient({{.ClientArg}})
	if client == nil {
		return 0, fmt.Errorf("{{.DbErrorHint}}数据库不存在")
	}
	return client.HLen({{if .Keys}}GetKey({{.GetKeyCallArgs}}){{else}}KEY{{end}})
}

func HGetByCache(ctx define.IContext{{if NotUidArgs .GetHashFuncExtraParams}}, {{NotUidArgs .GetHashFuncExtraParams}}{{end}}) (*pb.{{.Name}}, bool, error) {
	{{- if .Keys}}
	key := GetKey({{CtxCallArgs .Keys}})
	{{- else}}
	key := KEY
	{{- end}}
	field := GetField({{CtxCallArgs .Fields}})
	cacheKey := key + field
	if obj, ok := ctx.GetCache(cacheKey); ok {
		return obj.(*pb.{{.Name}}), false, nil
	}
	item, isNew, err := HGet({{CtxCallArgs .GetHashFuncExtraParams}})
	if err != nil {
		return item, isNew, err
	}
	ctx.SetCache(cacheKey, item, {{.CacheFlag}})
	return item, isNew, nil
}

func Change(ctx define.IContext{{if NotUidArgs .GetHashFuncExtraParams}}, {{NotUidArgs .GetHashFuncExtraParams}}{{end}}) {
	{{- if .Keys}}
	ctx.Change(GetKey({{CtxCallArgs .Keys}}) + GetField({{CtxCallArgs .Fields}}))
	{{- else}}
	ctx.Change(KEY + GetField({{CtxCallArgs .Fields}}))
	{{- end}}
}

func Read(ctx define.IContext{{if NotUidArgs .GetHashFuncExtraParams}}, {{NotUidArgs .GetHashFuncExtraParams}}{{end}}) *pb.{{.Name}} {
	{{- if .Keys}}
	cacheKey := GetKey({{CtxCallArgs .Keys}}) + GetField({{CtxCallArgs .Fields}})
	{{- else}}
	cacheKey := KEY + GetField({{CtxCallArgs .Fields}})
	{{- end}}
	if obj, ok := ctx.GetCache(cacheKey); ok {
		return obj.(*pb.{{.Name}})
	}
	return nil
}

func IsChanged(ctx define.IContext{{if NotUidArgs .GetHashFuncExtraParams}}, {{NotUidArgs .GetHashFuncExtraParams}}{{end}}) bool {
	{{- if .Keys}}
	return ctx.IsChanged(GetKey({{CtxCallArgs .Keys}}) + GetField({{CtxCallArgs .Fields}}))
	{{- else}}
	return ctx.IsChanged(KEY + GetField({{CtxCallArgs .Fields}}))
	{{- end}}
}

func Reset(ctx define.IContext{{if NotUidArgs .GetHashFuncExtraParams}}, {{NotUidArgs .GetHashFuncExtraParams}}{{end}}) {
	{{- if .Keys}}
	ctx.Reset(GetKey({{CtxCallArgs .Keys}}) + GetField({{CtxCallArgs .Fields}}))
	{{- else}}
	ctx.Reset(KEY + GetField({{CtxCallArgs .Fields}}))
	{{- end}}
}
`

// TemplateFuncMap 模板函数映射，供模板渲染时使用
var TemplateFuncMap = template.FuncMap{
	"GetValues":        templateGetValues,
	"GetArgs":          templateGetArgs,
	"GetTypeList":      templateGetTypeList,
	"GetShardArg":      templateGetShardArg,
	"GetBatchShardArg": templateGetBatchShardArg,
	"HasAnyFields":     templateHasAnyFields,
	"JoinArgs":         templateJoinArgs,
	"JoinArgsTrail":    templateJoinArgsTrail,
	"ShardArgDecl":     templateShardArgDecl,
	"NotUidArgs":       templateNotUidArgs,
	"CtxCallArgs":      templateCtxCallArgs,
	"fieldCount":       templateFieldCount,
}

// BuildStringTemplate 构建String类型的Go代码模板
func BuildStringTemplate() (*template.Template, error) {
	fullSrc := StringTemplStr + "\n" + StringMethods
	tpl, err := template.New("redis_string").Funcs(TemplateFuncMap).Parse(fullSrc)
	if err != nil {
		return nil, err
	}
	return tpl, nil
}

// BuildHashTemplate 构建Hash类型的Go代码模板
func BuildHashTemplate() (*template.Template, error) {
	fullSrc := HashTemplStr + "\n" + HashMethods
	tpl, err := template.New("redis_hash").Funcs(TemplateFuncMap).Parse(fullSrc)
	if err != nil {
		return nil, err
	}
	return tpl, nil
}

// templateGetValues 从字段列表中提取参数名列表（模板函数）
func templateGetValues(args ...[]*domain.Field) string {
	if len(args) == 0 {
		return ""
	}
	names := make([]string, 0, len(args)*4)
	for _, list := range args {
		for _, f := range list {
			names = append(names, f.Name)
		}
	}
	return strings.Join(names, ",")
}

// templateGetArgs 从字段列表中生成Go函数参数声明（模板函数）
func templateGetArgs(fields ...[]*domain.Field) string {
	if len(fields) == 0 {
		return ""
	}
	args := make([]string, 0, len(fields)*4)
	for _, list := range fields {
		for _, f := range list {
			args = append(args, fmt.Sprintf("%s %s", f.Name, f.Type))
		}
	}
	return strings.Join(args, ", ")
}

// templateGetTypeList 从字段列表中提取类型名列表，用于泛型类型参数（模板函数）
// 例: [{Username, string}] → "string"
//
//	[{GameId, int32}, {Type, string}] → "int32, string"
func templateGetTypeList(fields ...[]*domain.Field) string {
	if len(fields) == 0 {
		return ""
	}
	types := make([]string, 0, len(fields)*4)
	for _, list := range fields {
		for _, f := range list {
			types = append(types, f.Type)
		}
	}
	return strings.Join(types, ", ")
}

// templateGetShardArg 返回分片/全局参数的函数声明部分
// 当 ShardField 与 Key 同名（IsShardKeySame=true）时返回空
// 接受可选的后续字段列表：当后续参数为空时不输出尾逗号，避免 "uid uint64, , data" 的双逗号问题
func templateGetShardArg(m domain.ModelWithDbInfo, extraFields ...[]*domain.Field) string {
	sf := m.GetShardField()
	if sf == nil {
		return ""
	}
	if same, ok := m.(interface{ IsShardKeySame() bool }); ok && same.IsShardKeySame() {
		return ""
	}
	// 检查是否有任何后续参数
	hasExtra := false
	for _, list := range extraFields {
		if len(list) > 0 {
			hasExtra = true
			break
		}
	}
	if hasExtra {
		return fmt.Sprintf("%s %s, ", sf.Name, sf.Type)
	}
	return fmt.Sprintf("%s %s", sf.Name, sf.Type)
}

// templateHasAnyFields 判断传入的所有字段列表是否全部为空
func templateHasAnyFields(fields ...[]*domain.Field) bool {
	for _, list := range fields {
		if len(list) > 0 {
			return true
		}
	}
	return false
}

// templateGetBatchShardArg 批量操作(MSet/MGet)的分片/全局参数声明
// Shards: 固定输出 "shardId uint32, "，由业务层传入目标分片ID
// Global: 输出全局参数（如 "DbName string, "）
func templateGetBatchShardArg(m domain.ModelWithDbInfo) string {
	sf := m.GetShardField()
	if sf == nil {
		return ""
	}
	switch m.GetDbType() {
	case domain.DbTypeShards:
		return "shardId uint32, "
	case domain.DbTypeGlobal:
		return fmt.Sprintf("%s %s, ", sf.Name, sf.Type)
	default:
		return ""
	}
}

// templateJoinArgs 智能拼接函数参数声明列表，自动处理空段和逗号分隔
// 用法: {{JoinArgs (ShardArgDecl . .Keys) .Keys .Fields}}
// 每个参数可以是 []*domain.Field 切片或 string（如 shard 参数声明），自动过滤空段并用 ", " 连接
func templateJoinArgs(args ...any) string {
	parts := make([]string, 0, len(args))
	for _, arg := range args {
		switch v := arg.(type) {
		case string:
			if v != "" {
				parts = append(parts, v)
			}
		case []*domain.Field:
			if len(v) > 0 {
				parts = append(parts, templateGetArgs(v))
			}
		}
	}
	return strings.Join(parts, ", ")
}

// templateJoinArgsTrail 与 JoinArgs 相同，但当结果非空时在末尾追加 ", "
// 用于模板中参数列表后还有固定参数的场景，如: {{JoinArgsTrail (ShardArgDecl . .Keys) .Keys}}fields ...string
func templateJoinArgsTrail(args ...any) string {
	result := templateJoinArgs(args...)
	if result != "" {
		return result + ", "
	}
	return ""
}

// templateShardArgDecl 返回分片参数的声明字符串（不带尾逗号），供 JoinArgs 使用
// 当 ShardField 为 nil 或 IsShardKeySame 时返回空字符串
func templateShardArgDecl(m domain.ModelWithDbInfo, extraFields ...[]*domain.Field) string {
	sf := m.GetShardField()
	if sf == nil {
		return ""
	}
	if same, ok := m.(interface{ IsShardKeySame() bool }); ok && same.IsShardKeySame() {
		return ""
	}
	return fmt.Sprintf("%s %s", sf.Name, sf.Type)
}

// templateNotUidArgs 类似 GetArgs，但过滤掉名称含 "uid" 的字段
// 用于 :cache 函数签名中去除 uid 参数（由 ctx.GetUid() 替代）
func templateNotUidArgs(fields ...[]*domain.Field) string {
	var filtered []*domain.Field
	for _, list := range fields {
		for _, f := range list {
			if !strings.EqualFold(f.Name, "uid") {
				filtered = append(filtered, f)
			}
		}
	}
	return templateGetArgs(filtered)
}

// templateCtxCallArgs 生成调用参数列表，uid 字段替换为 ctx.GetUid()
// 用于 :cache 函数体中调用 GetKey/Get/HGet 等时传递参数
func templateCtxCallArgs(fields ...[]*domain.Field) string {
	var args []string
	for _, list := range fields {
		for _, f := range list {
			if strings.EqualFold(f.Name, "uid") {
				args = append(args, "ctx.GetUid()")
			} else {
				args = append(args, f.Name)
			}
		}
	}
	return strings.Join(args, ",")
}

// templateFieldCount 返回字段列表长度，用于模板中动态选择 S0~S3 / H0~H3
func templateFieldCount(fields ...[]*domain.Field) int {
	n := 0
	for _, list := range fields {
		n += len(list)
	}
	return n
}
