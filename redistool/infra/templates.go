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
`

const StringMethods = `

func GetKey(ctx define.IContext{{if .GetNonUidKeys}}, {{GetArgs .GetNonUidKeys}}{{end}}) string {
{{- if .Keys}}
	return fmt.Sprintf("{{.Format}}", {{.GetKeyFmtArgs}})
{{- else}}
	return "{{.Format}}"
{{- end}}
}

func Get(ctx define.IContext{{if .GetNonUidKeys}}, {{GetArgs .GetNonUidKeys}}{{end}}) (*pb.{{.Name}}, bool, error) {
	key := GetKey({{.GetKeyCallArgs}})
	if obj, ok := ctx.GetCache(key); ok {
		return obj.(*pb.{{.Name}}), false, nil
	}
	client := {{.ClientCallExpr}}
	if client == nil {
		return nil, false, fmt.Errorf("{{.DbErrorHint}}数据库不存在")
	}
	body, err := client.Get(key)
	if err != nil {
		return nil, false, err
	}
	if len(body) <= 0 {
		return nil, true, nil
	}
	item := &pb.{{.Name}}{}
	if err := item.UnmarshalVT([]byte(body)); err != nil {
		return nil, false, err
	}
	ctx.SetCache(key, item)
	return item, false, nil
}

func Set(ctx define.IContext{{if .GetNonUidKeys}}, {{GetArgs .GetNonUidKeys}}{{end}}, val *pb.{{.Name}}, expiration time.Duration) error {
	key := GetKey({{.GetKeyCallArgs}})
	client := {{.ClientCallExpr}}
	if client == nil {
		return fmt.Errorf("{{.DbErrorHint}}数据库不存在")
	}
	buf, err := val.MarshalVT()
	if err != nil {
		return err
	}
	ctx.SetCache(key, val)
	return client.Set(key, string(buf), expiration)
}

func Del(ctx define.IContext{{if .GetNonUidKeys}}, {{GetArgs .GetNonUidKeys}}{{end}}) error {
	client := {{.ClientCallExpr}}
	if client == nil {
		return fmt.Errorf("{{.DbErrorHint}}数据库不存在")
	}
	key := GetKey({{.GetKeyCallArgs}})
	_, err := client.Del(key)
	return err
}

func Remove(ctx define.IContext, {{GetBatchShardArg .}}keys ...string) error {
	if len(keys) <= 0 {
		return nil
	}
	client := {{.BatchClientExpr}}
	if client == nil {
		return fmt.Errorf("{{.DbErrorHint}}数据库不存在")
	}
	_, err := client.Del(keys...)
	return err
}

func Change(ctx define.IContext{{if .GetNonUidKeys}}, {{GetArgs .GetNonUidKeys}}{{end}}) {
	ctx.Change(GetKey({{.GetKeyCallArgs}}))
}

func Read(ctx define.IContext{{if .GetNonUidKeys}}, {{GetArgs .GetNonUidKeys}}{{end}}) *pb.{{.Name}} {
	if obj, ok := ctx.GetCache(GetKey({{.GetKeyCallArgs}})); ok {
		return obj.(*pb.{{.Name}})
	}
	return nil
}

func MSet(ctx define.IContext, {{GetBatchShardArg .}}vals map[string]*pb.{{.Name}}) error {
	client := {{.BatchClientExpr}}
	if client == nil {
		return fmt.Errorf("{{.DbErrorHint}}数据库不存在")
	}
	args := []any{}
	for key, val := range vals {
		args = append(args, key)
		buf, err := val.MarshalVT()
		if err != nil {
			return err
		}
		mlog.Trace("[redis-save]", key, val)
		args = append(args, string(buf))
	}
	return client.MSet(args...)
}

func MGet(ctx define.IContext, {{GetBatchShardArg .}}keys ...string) (map[string]*pb.{{.Name}}, error) {
	rets := map[string]*pb.{{.Name}}{}
	missing := make([]string, 0, len(keys))
	for _, key := range keys {
		if obj, ok := ctx.GetCache(key); ok {
			rets[key] = obj.(*pb.{{.Name}})
		} else {
			missing = append(missing, key)
		}
	}
	if len(missing) == 0 {
		return rets, nil
	}
	client := {{.BatchClientExpr}}
	if client == nil {
		return nil, fmt.Errorf("{{.DbErrorHint}}数据库不存在")
	}
	values, err := client.MGet(missing...)
	if err != nil {
		return nil, err
	}
	for i, key := range missing {
		value := values[i]
		if value == nil {
			continue
		}
		switch v := value.(type) {
		case string:
			if len(v) > 0 {
				item := &pb.{{.Name}}{}
				if err := item.UnmarshalVT([]byte(v)); err == nil {
					mlog.Trace("[redis-load]", key, item)
					ctx.SetCache(key, item)
					rets[key] = item
				}
			}
		case []byte:
			if len(v) > 0 {
				item := &pb.{{.Name}}{}
				if err := item.UnmarshalVT(v); err == nil {
					mlog.Trace("[redis-load]", key, item)
					ctx.SetCache(key, item)
					rets[key] = item
				}
			}
		}
	}
	return rets, nil
}

`

const HashTemplStr = `/*
* 本代码由redistool工具生成，请勿手动修改
*/

package {{.Pkg}}
`

const HashMethods = `

func GetKey(ctx define.IContext{{if .GetNonUidKeys}}, {{GetArgs .GetNonUidKeys}}{{end}}) string {
{{- if .Keys}}
	return fmt.Sprintf("{{.KeyFmt}}", {{.GetKeyFmtArgs}})
{{- else}}
	return "{{.KeyFmt}}"
{{- end}}
}

func GetField(ctx define.IContext{{if .GetNonUidFields}}, {{GetArgs .GetNonUidFields}}{{end}}) string {
{{- if .Fields}}
	return fmt.Sprintf("{{.FieldFmt}}", {{.GetFieldFmtArgs}})
{{- else}}
	return "{{.FieldFmt}}"
{{- end}}
}

func HGet(ctx define.IContext{{if .GetHashFuncExtraParams}}, {{GetArgs .GetHashFuncExtraParams}}{{end}}) (*pb.{{.Name}}, bool, error) {
	key := GetKey({{.GetKeyCallArgs}})
	field := GetField({{.GetFieldCallArgs}})
	cacheKey := key + field
	if obj, ok := ctx.GetCache(cacheKey); ok {
		return obj.(*pb.{{.Name}}), false, nil
	}
	client := {{.ClientCallExpr}}
	if client == nil {
		return nil, false, fmt.Errorf("{{.DbErrorHint}}数据库不存在")
	}
	body, err := client.HGet(key, field)
	if err != nil {
		return nil, false, err
	}
	if len(body) <= 0 {
		return nil, true, nil
	}
	item := &pb.{{.Name}}{}
	if err := item.UnmarshalVT([]byte(body)); err != nil {
		return nil, false, err
	}
	ctx.SetCache(cacheKey, item)
	return item, false, nil
}

func HSet(ctx define.IContext{{if .GetHashFuncExtraParams}}, {{GetArgs .GetHashFuncExtraParams}}{{end}}, val *pb.{{.Name}}) error {
	key := GetKey({{.GetKeyCallArgs}})
	field := GetField({{.GetFieldCallArgs}})
	client := {{.ClientCallExpr}}
	if client == nil {
		return fmt.Errorf("{{.DbErrorHint}}数据库不存在")
	}
	buf, err := val.MarshalVT()
	if err != nil {
		return err
	}
	ctx.SetCache(key+field, val)
	return client.HSet(key, field, string(buf))
}

func HDel(ctx define.IContext{{if .GetHashFuncExtraParams}}, {{GetArgs .GetHashFuncExtraParams}}{{end}}) error {
	client := {{.ClientCallExpr}}
	if client == nil {
		return fmt.Errorf("{{.DbErrorHint}}数据库不存在")
	}
	key := GetKey({{.GetKeyCallArgs}})
	field := GetField({{.GetFieldCallArgs}})
	_, err := client.HDel(key, field)
	return err
}

func HRemove(ctx define.IContext{{if .GetNonUidKeys}}, {{GetArgs .GetNonUidKeys}}{{end}}, fields ...string) error {
	if len(fields) <= 0 {
		return nil
	}
	client := {{.ClientCallExpr}}
	if client == nil {
		return fmt.Errorf("{{.DbErrorHint}}数据库不存在")
	}
	key := GetKey({{.GetKeyCallArgs}})
	_, err := client.HDel(key, fields...)
	return err
}

func Change(ctx define.IContext{{if .GetHashFuncExtraParams}}, {{GetArgs .GetHashFuncExtraParams}}{{end}}) {
	ctx.Change(GetKey({{.GetKeyCallArgs}}) + GetField({{.GetFieldCallArgs}}))
}

func Read(ctx define.IContext{{if .GetHashFuncExtraParams}}, {{GetArgs .GetHashFuncExtraParams}}{{end}}) *pb.{{.Name}} {
	cacheKey := GetKey({{.GetKeyCallArgs}}) + GetField({{.GetFieldCallArgs}})
	if obj, ok := ctx.GetCache(cacheKey); ok {
		return obj.(*pb.{{.Name}})
	}
	return nil
}

func HGetAll(ctx define.IContext{{if .GetNonUidKeys}}, {{GetArgs .GetNonUidKeys}}{{end}}) (ret map[string]*pb.{{.Name}}, err error) {
	client := {{.ClientCallExpr}}
	if client == nil {
		return nil, fmt.Errorf("{{.DbErrorHint}}数据库不存在")
	}
	key := GetKey({{.GetKeyCallArgs}})
	kvs, err := client.HGetAll(key)
	if err != nil {
		return nil, err
	}
	ret = make(map[string]*pb.{{.Name}})
	for k, item := range kvs {
		if len(item) <= 0 {
			continue
		}
		data := &pb.{{.Name}}{}
		if err := data.UnmarshalVT([]byte(item)); err != nil {
			return nil, err
		}
		mlog.Tracef("[redis-load]", key, k, data)
		ret[k] = data
	}
	return
}

func HLen(ctx define.IContext{{if .GetNonUidKeys}}, {{GetArgs .GetNonUidKeys}}{{end}}) (int64, error) {
	client := {{.ClientCallExpr}}
	if client == nil {
		return 0, fmt.Errorf("{{.DbErrorHint}}数据库不存在")
	}
	return client.HLen(GetKey({{.GetKeyCallArgs}}))
}

func HMGet(ctx define.IContext{{if .GetNonUidKeys}}, {{GetArgs .GetNonUidKeys}}{{end}}, fields ...string) (map[string]*pb.{{.Name}}, error) {
	if len(fields) <= 0 {
		return nil, nil
	}
	rets := map[string]*pb.{{.Name}}{}
	key := GetKey({{.GetKeyCallArgs}})
	missing := make([]string, 0, len(fields))
	for _, field := range fields {
		cacheKey := key + field
		if obj, ok := ctx.GetCache(cacheKey); ok {
			rets[field] = obj.(*pb.{{.Name}})
		} else {
			missing = append(missing, field)
		}
	}
	if len(missing) == 0 {
		return rets, nil
	}
	client := {{.ClientCallExpr}}
	if client == nil {
		return nil, fmt.Errorf("{{.DbErrorHint}}数据库不存在")
	}
	results, err := client.HMGet(key, missing...)
	if err != nil {
		return nil, err
	}
	for i, field := range missing {
		if results[i] == nil {
			continue
		}
		var buf []byte
		switch vv := results[i].(type) {
		case string:
			buf = []byte(vv)
		case []byte:
			buf = vv
		default:
			return nil, fmt.Errorf("数据类型不支持")
		}
		item := &pb.{{.Name}}{}
		if err := item.UnmarshalVT(buf); err != nil {
			return nil, err
		}
		mlog.Trace("[redis-load]", key, field, item)
		ctx.SetCache(key+field, item)
		rets[field] = item
	}
	return rets, nil
}

func HMSet(ctx define.IContext{{if .GetNonUidKeys}}, {{GetArgs .GetNonUidKeys}}{{end}}, data map[string]*pb.{{.Name}}) error {
	client := {{.ClientCallExpr}}
	if client == nil {
		return fmt.Errorf("{{.DbErrorHint}}数据库不存在")
	}
	key := GetKey({{.GetKeyCallArgs}})
	vals := []any{}
	for k, v := range data {
		buf, err := v.MarshalVT()
		if err != nil {
			return err
		}
		mlog.Trace("[redis-save]", key, k, v)
		vals = append(vals, k, string(buf))
	}
	return client.HMSet(key, vals...)
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
