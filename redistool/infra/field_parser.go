package infra

import (
	"fmt"
	"strings"

	"github.com/hechh/tools/redistool/domain"
)

// ParseFieldFormat 解析字段格式串 "key:name@type,name2@type2" -> (格式化串, 字段列表)
// 支持两种格式：
//   - "prefix:Name@type"   → 格式串 "prefix:%d", fields=[{Name,type}]  (带前缀)
//   - "Name@type"           → 格式串 "%d",       fields=[{Name,type}]    (无前缀，纯参数)
//
// 这是纯函数，无副作用，可被领域层或基础设施层安全调用
func ParseFieldFormat(str string) (format string, fields []*domain.Field) {
	if !strings.Contains(str, "@") {
		format = str
		return
	}
	hasPrefix := false
	if pos := strings.Index(str, ":"); pos >= 0 && pos < strings.Index(str, "@") {
		// 冒号在 @ 之前，说明是 "prefix:Name@type" 格式
		format = str[:pos]
		str = str[pos+1:]
		hasPrefix = true
	}
	for _, segment := range strings.Split(str, ",") {
		pos := strings.Index(segment, "@")
		if pos < 0 {
			continue
		}
		name := segment[:pos]
		typ := segment[pos+1:]
		switch strings.ToLower(typ) {
		case "string":
			format += ":%s"
		default:
			format += ":%d"
		}
		fields = append(fields, &domain.Field{Name: name, Type: typ})
	}
	// 无前缀时去掉首字符冒号
	if !hasPrefix && len(format) > 0 && format[0] == ':' {
		format = format[1:]
	}
	return
}

// ParseDbSpec 解析数据库规格字符串，返回 (DbType, 静态DbName/常量名, ShardField)
// 支持格式：
//   - "MyDb"                -> (Static, "MyDb", nil)              静态数据库名
//   - "global:ConstName"    -> (Global, "ConstName", nil)         全局常量引用（无函数参数）
//   - "global:Name@string"  -> (Global, "", &Field{Name:"Name"})  全局运行时参数
//   - "shards:uid@uint64"   -> (Shards, "", &Field{...})          分片路由参数
func ParseDbSpec(dbSpec string) (dbType domain.DbType, dbName string, shardField *domain.Field) {
	if !strings.HasPrefix(dbSpec, "global:") && dbSpec != "shards" &&
		!strings.HasPrefix(dbSpec, "shards:") && !strings.HasPrefix(dbSpec, "shards@") {
		return domain.DbTypeStatic, dbSpec, nil
	}

	if strings.HasPrefix(dbSpec, "global:") {
		rest := strings.TrimPrefix(dbSpec, "global:")
		if !strings.Contains(rest, "@") {
			// 无 @ 分隔符：常量格式 global:ConstName，直接使用代码中的常量
			return domain.DbTypeGlobal, rest, nil
		}
		_, fields := ParseFieldFormat(rest)
		if len(fields) > 0 {
			f := fields[0]
			if strings.ToLower(f.Type) != "string" {
				fmt.Printf("[redistool] global 运行时参数必须是 string 类型（表示数据库名），当前声明为 %s@%s\n", f.Name, f.Type)
				return domain.DbTypeGlobal, "", nil
			}
			return domain.DbTypeGlobal, "", f
		}
		return domain.DbTypeGlobal, "", nil
	}

	// shards / shards:field@type / shards@field@type
	if dbSpec == "shards" {
		return domain.DbTypeShards, "", nil
	}
	// 优先匹配 shards:field@type（实际注解使用冒号）
	var rest string
	if strings.HasPrefix(dbSpec, "shards:") {
		rest = strings.TrimPrefix(dbSpec, "shards:")
	} else {
		rest = strings.TrimPrefix(dbSpec, "shards@")
	}
	parts := strings.SplitN(rest, "@", 2)
	if len(parts) == 2 {
		return domain.DbTypeShards, "", &domain.Field{Name: parts[0], Type: parts[1]}
	}
	return domain.DbTypeShards, "", nil
}
