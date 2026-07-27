package domain

import (
	"fmt"
	"strings"
)

// Field 字段描述，用于key或field格式化中的占位符
type Field struct {
	Name string // 参数名（如 Uid, RoomId）
	Type string // 类型名（如 int64, string）
}

// DbType 数据库访问类型
type DbType string

const (
	DbTypeStatic DbType = ""       // 静态数据库名（旧格式兼容）：直接使用 GetByName("name")
	DbTypeGlobal DbType = "global" // 全局数据库（动态名称）：使用 GetByName(param）
	DbTypeShards DbType = "shards" // 分片数据库：使用 GetByUid / GetById 路由
)

// RedisString String类型Redis模型定义
type RedisString struct {
	Pkg        string   // 包名(snake_case)
	Name       string   // 结构体名(PascalCase)
	DbType     DbType   // 数据库访问类型
	DbName     string   // 静态目标Redis数据库名
	ShardField *Field   // 分片路由字段
	Format     string   // key格式串
	Keys       []*Field // key中包含的动态字段列表
	UseCache   bool     // 是否生成GetByCache缓存接口
}

// RedisHash Hash类型Redis模型定义
type RedisHash struct {
	Pkg        string   // 包名(snake_case)
	Name       string   // 结构体名(PascalCase)
	DbType     DbType   // 数据库访问类型
	DbName     string   // 静态目标Redis数据库名
	ShardField *Field   // 分片路由字段
	KeyFmt     string   // redis key格式串
	Keys       []*Field // key中的动态字段
	FieldFmt   string   // redis field格式串
	Fields     []*Field // field中的动态字段
	UseCache   bool     // 是否生成HGetByCache缓存接口
}

// ParseContext 解析上下文，聚合所有从源码中提取的模型
type ParseContext struct {
	Strings []*RedisString
	Hashs   []*RedisHash
}

func (c *ParseContext) AddString(m *RedisString) { c.Strings = append(c.Strings, m) }
func (c *ParseContext) AddHash(m *RedisHash)     { c.Hashs = append(c.Hashs, m) }

// ==================== 模板渲染用接口与计算属性 ====================

// ModelWithDbInfo 模板函数使用的统一接口
type ModelWithDbInfo interface {
	GetDbType() DbType
	GetDbName() string
	GetShardField() *Field
}

// --- RedisString 实现 ModelWithDbInfo ---
func (m *RedisString) GetDbType() DbType     { return m.DbType }
func (m *RedisString) GetDbName() string     { return m.DbName }
func (m *RedisString) GetShardField() *Field { return m.ShardField }

// UsesCache 是否启用cache规则
func (m *RedisString) UsesCache() bool { return m.UseCache }
func (m *RedisHash) UsesCache() bool   { return m.UseCache }

// GetCacheKeyExpr 生成缓存key的表达式
func (m *RedisString) GetCacheKeyExpr() string {
	return "GetKey(" + m.GetKeyCallArgs() + ")"
}

// GetNonUidKeys 返回Keys中名称不为uid的字段列表
func (m *RedisString) GetNonUidKeys() []*Field {
	var result []*Field
	for _, f := range m.Keys {
		if !strings.EqualFold(f.Name, "uid") {
			result = append(result, f)
		}
	}
	return result
}

// GetCacheCallArg 生成GetByCache中调用Get()的参数列表（过滤uid，用ctx.GetUid()替代）
func (m *RedisString) GetCacheCallArg() string {
	var args []string
	sf := m.ShardField
	if sf != nil && !m.IsShardKeySame() {
		args = append(args, "ctx.GetUid()")
	}
	for _, f := range m.Keys {
		if strings.EqualFold(f.Name, "uid") {
			args = append(args, "ctx.GetUid()")
		} else {
			args = append(args, f.Name)
		}
	}
	if len(args) == 0 {
		return "ctx.GetUid()"
	}
	return strings.Join(args, ", ")
}

// ClientCallExpr 返回内联客户端获取表达式（ctx版本）
func (m *RedisString) ClientCallExpr() string {
	switch m.DbType {
	case DbTypeGlobal:
		if m.ShardField != nil {
			return fmt.Sprintf(`redispool.GetByName(%s)`, m.ShardField.Name)
		}
		return fmt.Sprintf(`redispool.GetByName("%s")`, m.DbName)
	case DbTypeShards:
		if strings.EqualFold(m.ShardField.Name, "uid") {
			return `redispool.GetByUid(ctx.GetUid())`
		}
		return fmt.Sprintf(`redispool.GetByUid(%s)`, m.ShardField.Name)
	default:
		return fmt.Sprintf(`redispool.GetByName("%s")`, m.DbName)
	}
}

// GetKeyCallArgs 生成 GetKey() 调用的参数列表（ctx 作为第一参数，uid 字段被 ctx 消费）
// 例: Keys=[{Uid,uint64}] → "ctx"
//
//	Keys=[] → "ctx"
//	Keys=[{Username,string}] → "ctx, Username"
func (m *RedisString) GetKeyCallArgs() string {
	args := []string{"ctx"}
	for _, f := range m.Keys {
		if !strings.EqualFold(f.Name, "uid") {
			args = append(args, f.Name)
		}
	}
	return strings.Join(args, ",")
}

// GetKeyFmtArgs 生成 fmt.Sprintf 的参数列表（uid → ctx.GetUid()，其余用字段名）
// 例: Keys=[{Uid,uint64}] → "ctx.GetUid()"
//
//	Keys=[{Username,string}] → "Username"
//	Keys=[{Uid,uint64},{GameId,int32}] → "ctx.GetUid(), GameId"
func (m *RedisString) GetKeyFmtArgs() string {
	if len(m.Keys) == 0 {
		return ""
	}
	args := make([]string, 0, len(m.Keys))
	for _, f := range m.Keys {
		if strings.EqualFold(f.Name, "uid") {
			args = append(args, "ctx.GetUid()")
		} else {
			args = append(args, f.Name)
		}
	}
	return strings.Join(args, ",")
}

// --- RedisHash 实现 ModelWithDbInfo ---
func (m *RedisHash) GetDbType() DbType { return m.DbType }

// GetNonUidKeys 返回Keys中名称不为uid的字段列表
func (m *RedisHash) GetNonUidKeys() []*Field {
	var result []*Field
	for _, f := range m.Keys {
		if !strings.EqualFold(f.Name, "uid") {
			result = append(result, f)
		}
	}
	return result
}

// GetNonUidFields 返回Fields中名称不为uid的字段列表
func (m *RedisHash) GetNonUidFields() []*Field {
	var result []*Field
	for _, f := range m.Fields {
		if !strings.EqualFold(f.Name, "uid") {
			result = append(result, f)
		}
	}
	return result
}

// GetKeyCallArgs 生成 GetKey() 调用的参数列表（ctx + 非uid的key字段）
func (m *RedisHash) GetKeyCallArgs() string {
	args := []string{"ctx"}
	for _, f := range m.Keys {
		if !strings.EqualFold(f.Name, "uid") {
			args = append(args, f.Name)
		}
	}
	return strings.Join(args, ",")
}

// GetFieldCallArgs 生成 GetField() 调用的参数列表（ctx + 非uid的field字段）
func (m *RedisHash) GetFieldCallArgs() string {
	args := []string{"ctx"}
	for _, f := range m.Fields {
		if !strings.EqualFold(f.Name, "uid") {
			args = append(args, f.Name)
		}
	}
	return strings.Join(args, ",")
}

// GetKeyFmtArgs 生成 key 的 fmt.Sprintf 参数（uid→ctx.GetUid()，其余用字段名）
func (m *RedisHash) GetKeyFmtArgs() string {
	if len(m.Keys) == 0 {
		return ""
	}
	args := make([]string, 0, len(m.Keys))
	for _, f := range m.Keys {
		if strings.EqualFold(f.Name, "uid") {
			args = append(args, "ctx.GetUid()")
		} else {
			args = append(args, f.Name)
		}
	}
	return strings.Join(args, ",")
}

// GetFieldFmtArgs 生成 field 的 fmt.Sprintf 参数（uid→ctx.GetUid()，其余用字段名）
func (m *RedisHash) GetFieldFmtArgs() string {
	if len(m.Fields) == 0 {
		return ""
	}
	args := make([]string, 0, len(m.Fields))
	for _, f := range m.Fields {
		if strings.EqualFold(f.Name, "uid") {
			args = append(args, "ctx.GetUid()")
		} else {
			args = append(args, f.Name)
		}
	}
	return strings.Join(args, ",")
}

// ClientCallExpr 返回ctx感知的客户端获取表达式
func (m *RedisHash) ClientCallExpr() string {
	switch m.DbType {
	case DbTypeGlobal:
		if m.ShardField != nil {
			return fmt.Sprintf(`redispool.GetByName(%s)`, m.ShardField.Name)
		}
		return fmt.Sprintf(`redispool.GetByName("%s")`, m.DbName)
	case DbTypeShards:
		if strings.EqualFold(m.ShardField.Name, "uid") {
			return `redispool.GetByUid(ctx.GetUid())`
		}
		return fmt.Sprintf(`redispool.GetByUid(%s)`, m.ShardField.Name)
	default:
		return fmt.Sprintf(`redispool.GetByName("%s")`, m.DbName)
	}
}

// GetHashCacheKeyExpr 生成缓存key表达式: GetKey(ctx,...) + GetField(ctx,...)
func (m *RedisHash) GetHashCacheKeyExpr() string {
	return "GetKey(" + m.GetKeyCallArgs() + ") + GetField(" + m.GetFieldCallArgs() + ")"
}

// GetHashCallArgs 生成调用 HGet/HSget 的参数列表（ctx + 非uid的key和field，去重shard）
func (m *RedisHash) GetHashCallArgs() string {
	args := []string{"ctx"}
	sf := m.ShardField

	if sf != nil && m.IsShardFieldSame() {
		// shard == Fields[0]，已由 ctx 消费或字段保留
		if !strings.EqualFold(sf.Name, "uid") {
			args = append(args, sf.Name)
		}
		for _, f := range m.Keys {
			if !strings.EqualFold(f.Name, "uid") {
				args = append(args, f.Name)
			}
		}
		for i := 1; i < len(m.Fields); i++ {
			f := m.Fields[i]
			if !strings.EqualFold(f.Name, "uid") {
				args = append(args, f.Name)
			}
		}
	} else {
		if sf != nil && !m.IsShardKeySame() && !strings.EqualFold(sf.Name, "uid") {
			args = append(args, sf.Name)
		}
		for _, f := range m.Keys {
			if !strings.EqualFold(f.Name, "uid") {
				args = append(args, f.Name)
			}
		}
		for _, f := range m.Fields {
			if !strings.EqualFold(f.Name, "uid") {
				args = append(args, f.Name)
			}
		}
	}
	return strings.Join(args, ",")
}

// GetHashFuncExtraParams 返回 HGet/HSet/HDel 签名中 ctx 之后的额外参数
func (m *RedisHash) GetHashFuncExtraParams() []*Field {
	var result []*Field
	sf := m.ShardField

	if sf != nil && m.IsShardFieldSame() {
		if !strings.EqualFold(sf.Name, "uid") {
			result = append(result, sf)
		}
		for _, f := range m.Keys {
			if !strings.EqualFold(f.Name, "uid") {
				result = append(result, f)
			}
		}
		for i := 1; i < len(m.Fields); i++ {
			f := m.Fields[i]
			if !strings.EqualFold(f.Name, "uid") {
				result = append(result, f)
			}
		}
	} else {
		if sf != nil && !m.IsShardKeySame() && !strings.EqualFold(sf.Name, "uid") {
			result = append(result, sf)
		}
		for _, f := range m.Keys {
			if !strings.EqualFold(f.Name, "uid") {
				result = append(result, f)
			}
		}
		for _, f := range m.Fields {
			if !strings.EqualFold(f.Name, "uid") {
				result = append(result, f)
			}
		}
	}
	return result
}

// IsShardKeySame 判断分片字段是否与Key参数同名（模板用，避免重复输出参数）
func (m *RedisString) IsShardKeySame() bool {
	if m.ShardField == nil || len(m.Keys) == 0 {
		return false
	}
	return strings.EqualFold(m.ShardField.Name, m.Keys[0].Name)
}
func (m *RedisHash) IsShardKeySame() bool {
	if m.ShardField == nil || len(m.Keys) == 0 {
		return false
	}
	return strings.EqualFold(m.ShardField.Name, m.Keys[0].Name)
}

// IsShardFieldSame 判断分片字段是否与Field参数同名（模板用，避免重复输出参数）
// 例: shards:uid@uint64 | buff_data | uid@uint64 → HGet(uid) 而非 HGet(uid, uid)
func (m *RedisHash) IsShardFieldSame() bool {
	if m.ShardField == nil || len(m.Fields) == 0 {
		return false
	}
	return strings.EqualFold(m.ShardField.Name, m.Fields[0].Name)
}

// ClientExpr 返回内联客户端获取表达式（直接嵌入每个CRUD函数体内）
func (m *RedisString) ClientExpr() string {
	switch m.DbType {
	case DbTypeGlobal:
		if m.ShardField != nil {
			return fmt.Sprintf(`redispool.GetByName(%s)`, m.ShardField.Name)
		}
		return fmt.Sprintf(`redispool.GetByName("%s")`, m.DbName)
	case DbTypeShards:
		return fmt.Sprintf(`redispool.GetByUid(%s)`, m.ShardField.Name)
	default:
		return fmt.Sprintf(`redispool.GetByName("%s")`, m.DbName)
	}
}
func (m *RedisHash) ClientExpr() string {
	switch m.DbType {
	case DbTypeGlobal:
		if m.ShardField != nil {
			return fmt.Sprintf(`redispool.GetByName(%s)`, m.ShardField.Name)
		}
		return fmt.Sprintf(`redispool.GetByName("%s")`, m.DbName)
	case DbTypeShards:
		return fmt.Sprintf(`redispool.GetByUid(%s)`, m.ShardField.Name)
	default:
		return fmt.Sprintf(`redispool.GetByName("%s")`, m.DbName)
	}
}

// BatchClientExpr 批量操作（MSet/MGet）的客户端获取表达式，分片模式使用传入的 shardId
func (m *RedisString) BatchClientExpr() string {
	switch m.DbType {
	case DbTypeGlobal:
		if m.ShardField != nil {
			return fmt.Sprintf(`redispool.GetByName(%s)`, m.ShardField.Name)
		}
		return fmt.Sprintf(`redispool.GetByName("%s")`, m.DbName)
	case DbTypeShards:
		return `redispool.GetById(shardId)`
	default:
		return fmt.Sprintf(`redispool.GetByName("%s")`, m.DbName)
	}
}
func (m *RedisHash) BatchClientExpr() string {
	switch m.DbType {
	case DbTypeGlobal:
		if m.ShardField != nil {
			return fmt.Sprintf(`redispool.GetByName(%s)`, m.ShardField.Name)
		}
		return fmt.Sprintf(`redispool.GetByName("%s")`, m.DbName)
	case DbTypeShards:
		return `redispool.GetById(shardId)`
	default:
		return fmt.Sprintf(`redispool.GetByName("%s")`, m.DbName)
	}
}

// DbErrorHint 错误提示中的数据库标识
func (m *RedisString) DbErrorHint() string {
	switch m.DbType {
	case DbTypeGlobal:
		if m.ShardField != nil {
			return m.ShardField.Name + "对应"
		}
		return m.DbName
	case DbTypeShards:
		return m.ShardField.Name + "分片"
	default:
		return m.DbName
	}
}
func (m *RedisHash) DbErrorHint() string {
	switch m.DbType {
	case DbTypeGlobal:
		if m.ShardField != nil {
			return m.ShardField.Name + "对应"
		}
		return m.DbName
	case DbTypeShards:
		return m.ShardField.Name + "分片"
	default:
		return m.DbName
	}
}
