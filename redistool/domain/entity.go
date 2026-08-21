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
	UseCache   bool     // 是否生成 GetByCache/Change/Read 缓存函数
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
	UseCache   bool     // 是否生成 HGetByCache/Change/Read 缓存函数
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

// UsesCache 是否生成缓存函数
func (m *RedisString) UsesCache() bool { return m.UseCache }
func (m *RedisHash) UsesCache() bool   { return m.UseCache }

// CacheFlag 返回缓存标记位常量名
// :cache 规则按 db 类型区分：global→GLOBAL_CACHE_FLAG, shards→SHARDS_CACHE_FLAG
// 非 :cache 规则叠加 TEMP_CAHCE_FLAG：global→TEMP|GLOBAL, shards→TEMP|SHARDS
func (m *RedisString) CacheFlag() string {
	if m.UseCache {
		switch m.DbType {
		case DbTypeGlobal:
			return "redispool.GLOBAL_FLAG"
		case DbTypeShards:
			return "redispool.SHARDS_FLAG"
		default:
			return "redispool.GLOBAL_FLAG"
		}
	}
	switch m.DbType {
	case DbTypeGlobal:
		return "redispool.TEMP_FLAG | redispool.GLOBAL_FLAG"
	case DbTypeShards:
		return "redispool.TEMP_FLAG | redispool.SHARDS_FLAG"
	default:
		return "redispool.TEMP_FLAG | redispool.GLOBAL_FLAG"
	}
}
func (m *RedisHash) CacheFlag() string {
	if m.UseCache {
		switch m.DbType {
		case DbTypeGlobal:
			return "redispool.GLOBAL_FLAG"
		case DbTypeShards:
			return "redispool.SHARDS_FLAG"
		default:
			return "redispool.GLOBAL_FLAG"
		}
	}
	switch m.DbType {
	case DbTypeGlobal:
		return "redispool.TEMP_FLAG | redispool.GLOBAL_FLAG"
	case DbTypeShards:
		return "redispool.TEMP_FLAG | redispool.SHARDS_FLAG"
	default:
		return "redispool.TEMP_FLAG | redispool.GLOBAL_FLAG"
	}
}

// HasDbConst DbName 是否为固定字符串（需要生成 DBNAME 常量）
// 重构后数据库名统一通过 database 包常量（REDIS_GLOBAL/REDIS_PLAYER）或字面量引用，不再生成 DBNAME 常量
func (m *RedisHash) HasDbConst() bool {
	return false
}

// HasDbConst DbName 是否为固定字符串（需要生成 DBNAME 常量）
// 重构后数据库名统一通过 database 包常量（REDIS_GLOBAL/REDIS_PLAYER）或字面量引用，不再生成 DBNAME 常量
func (m *RedisString) HasDbConst() bool {
	return false
}

// NeedsDatabaseImport 生成的代码是否需要引入 richgame/pkg/database 包
// shards→database.REDIS_PLAYER、global 常量→database.REDIS_GLOBAL 会引用该包
func (m *RedisHash) NeedsDatabaseImport() bool {
	return m.DbType == DbTypeShards || (m.DbType == DbTypeGlobal && m.ShardField == nil)
}

// NeedsDatabaseImport 生成的代码是否需要引入 richgame/pkg/database 包
// shards→database.REDIS_PLAYER、global 常量→database.REDIS_GLOBAL 会引用该包
func (m *RedisString) NeedsDatabaseImport() bool {
	return m.DbType == DbTypeShards || (m.DbType == DbTypeGlobal && m.ShardField == nil)
}

// ClientCallExpr 返回内联客户端获取表达式
// redispool 只负责按名称连接 Redis 服务，业务分片由中间件处理
func (m *RedisString) ClientCallExpr() string {
	switch m.DbType {
	case DbTypeGlobal:
		if m.ShardField != nil {
			return fmt.Sprintf(`redispool.Get(%s)`, m.ShardField.Name)
		}
		return `redispool.Get(database.REDIS_GLOBAL)`
	case DbTypeShards:
		return `redispool.Get(database.REDIS_PLAYER)`
	default:
		return fmt.Sprintf(`redispool.Get("%s")`, m.DbName)
	}
}

// ClientFuncRef 返回客户端函数引用（用于 NewDataType），不含调用参数
func (m *RedisString) ClientFuncRef() string {
	switch m.DbType {
	case DbTypeGlobal:
		return `redispool.GetByName`
	case DbTypeShards:
		return `redispool.GetByUid`
	default:
		return `redispool.GetByName`
	}
}

// DataTypeFlags 返回 DataType 的标志位表达式
func (m *RedisString) DataTypeFlags() string {
	flags := `redispool.STRING_FLAG`
	switch m.DbType {
	case DbTypeGlobal:
		flags += ` | redispool.GLOBAL_FLAG`
	case DbTypeShards:
		flags += ` | redispool.SHARDS_FLAG`
	}
	flags += ` | redispool.PERMANENT_FLAG`
	return flags
}

// ClientArg 返回 DataType.GetClient() 的调用参数（有分片字段则用字段名，否则用 DBNAME 常量）
func (m *RedisString) ClientArg() string {
	if m.ShardField != nil {
		return m.ShardField.Name
	}
	return `DBNAME`
}

// ShardType 返回 DataType 泛型参数 I 的 Go 类型
func (m *RedisString) ShardType() string {
	switch m.DbType {
	case DbTypeGlobal:
		if m.ShardField != nil {
			return m.ShardField.Type
		}
		return `string`
	case DbTypeShards:
		if m.ShardField != nil {
			return m.ShardField.Type
		}
		return `uint64`
	default:
		return `string`
	}
}

// GetKeyCallArgs 生成 GetKey() 调用的参数列表
// 例: Keys=[{uid,uint64}] → "uid"
//
//	Keys=[{GameId,int32}] → "GameId"
func (m *RedisString) GetKeyCallArgs() string {
	args := make([]string, 0, len(m.Keys))
	for _, f := range m.Keys {
		args = append(args, f.Name)
	}
	return strings.Join(args, ",")
}

// GetKeyFmtArgs 生成 fmt.Sprintf 的参数列表
func (m *RedisString) GetKeyFmtArgs() string {
	if len(m.Keys) == 0 {
		return ""
	}
	args := make([]string, 0, len(m.Keys))
	for _, f := range m.Keys {
		args = append(args, f.Name)
	}
	return strings.Join(args, ",")
}

// --- RedisHash 实现 ModelWithDbInfo ---
func (m *RedisHash) GetDbType() DbType { return m.DbType }

// GetKeyCallArgs 生成 GetKey() 调用的参数列表
func (m *RedisHash) GetKeyCallArgs() string {
	args := make([]string, 0, len(m.Keys))
	for _, f := range m.Keys {
		args = append(args, f.Name)
	}
	return strings.Join(args, ",")
}

// GetFieldCallArgs 生成 GetField() 调用的参数列表
func (m *RedisHash) GetFieldCallArgs() string {
	args := make([]string, 0, len(m.Fields))
	for _, f := range m.Fields {
		args = append(args, f.Name)
	}
	return strings.Join(args, ",")
}

// GetKeyFmtArgs 生成 key 的 fmt.Sprintf 参数
func (m *RedisHash) GetKeyFmtArgs() string {
	if len(m.Keys) == 0 {
		return ""
	}
	args := make([]string, 0, len(m.Keys))
	for _, f := range m.Keys {
		args = append(args, f.Name)
	}
	return strings.Join(args, ",")
}

// GetFieldFmtArgs 生成 field 的 fmt.Sprintf 参数
func (m *RedisHash) GetFieldFmtArgs() string {
	if len(m.Fields) == 0 {
		return ""
	}
	args := make([]string, 0, len(m.Fields))
	for _, f := range m.Fields {
		args = append(args, f.Name)
	}
	return strings.Join(args, ",")
}

// ClientCallExpr 返回客户端获取表达式
// redispool 只负责按名称连接 Redis 服务，业务分片由中间件处理
func (m *RedisHash) ClientCallExpr() string {
	switch m.DbType {
	case DbTypeGlobal:
		if m.ShardField != nil {
			return fmt.Sprintf(`redispool.Get(%s)`, m.ShardField.Name)
		}
		return `redispool.Get(database.REDIS_GLOBAL)`
	case DbTypeShards:
		return `redispool.Get(database.REDIS_PLAYER)`
	default:
		return fmt.Sprintf(`redispool.Get("%s")`, m.DbName)
	}
}

// ClientFuncRef 返回客户端函数引用（用于 NewDataType），不含调用参数
func (m *RedisHash) ClientFuncRef() string {
	switch m.DbType {
	case DbTypeGlobal:
		return `redispool.GetByName`
	case DbTypeShards:
		return `redispool.GetByUid`
	default:
		return `redispool.GetByName`
	}
}

// DataTypeFlags 返回 DataType 的标志位表达式
func (m *RedisHash) DataTypeFlags() string {
	flags := `redispool.HASH_FLAG`
	switch m.DbType {
	case DbTypeGlobal:
		flags += ` | redispool.GLOBAL_FLAG`
	case DbTypeShards:
		flags += ` | redispool.SHARDS_FLAG`
	}
	flags += ` | redispool.PERMANENT_FLAG`
	return flags
}

// ClientArg 返回 DataType.GetClient() 的调用参数（有分片字段则用字段名，否则用 DBNAME 常量）
func (m *RedisHash) ClientArg() string {
	if m.ShardField != nil {
		return m.ShardField.Name
	}
	return `DBNAME`
}

// ShardType 返回 DataType 泛型参数 I 的 Go 类型
func (m *RedisHash) ShardType() string {
	switch m.DbType {
	case DbTypeGlobal:
		if m.ShardField != nil {
			return m.ShardField.Type
		}
		return `string`
	case DbTypeShards:
		if m.ShardField != nil {
			return m.ShardField.Type
		}
		return `uint64`
	default:
		return `string`
	}
}

// GetHashCallArgs 生成调用 HGet/HSet 的参数列表（shard + keys + fields，去重）
func (m *RedisHash) GetHashCallArgs() string {
	var args []string
	sf := m.ShardField

	if sf != nil {
		args = append(args, sf.Name)
	}
	for _, f := range m.Keys {
		if sf == nil || !strings.EqualFold(sf.Name, f.Name) {
			args = append(args, f.Name)
		}
	}
	for _, f := range m.Fields {
		if sf == nil || !strings.EqualFold(sf.Name, f.Name) {
			args = append(args, f.Name)
		}
	}
	return strings.Join(args, ",")
}

// GetHashFuncExtraParams 返回 HGet/HSet/HDel 签名中的参数列表（shard + keys + fields，去重）
func (m *RedisHash) GetHashFuncExtraParams() []*Field {
	var result []*Field
	sf := m.ShardField

	if sf != nil {
		result = append(result, sf)
	}
	for _, f := range m.Keys {
		if sf == nil || !strings.EqualFold(sf.Name, f.Name) {
			result = append(result, f)
		}
	}
	for _, f := range m.Fields {
		if sf == nil || !strings.EqualFold(sf.Name, f.Name) {
			result = append(result, f)
		}
	}
	return result
}

// GetHashKeyParams 返回 HMGet/HMSet 签名中的参数列表（shard + keys，不含 fields，去重）
func (m *RedisHash) GetHashKeyParams() []*Field {
	var result []*Field
	sf := m.ShardField
	if sf != nil {
		result = append(result, sf)
	}
	for _, f := range m.Keys {
		if sf == nil || !strings.EqualFold(sf.Name, f.Name) {
			result = append(result, f)
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

// BatchClientExpr 批量操作（Remove）的客户端获取表达式，分片模式使用传入的 shardId
func (m *RedisString) BatchClientExpr() string {
	switch m.DbType {
	case DbTypeGlobal:
		if m.ShardField != nil {
			return fmt.Sprintf(`redispool.Get(%s)`, m.ShardField.Name)
		}
		return `redispool.Get(database.REDIS_GLOBAL)`
	case DbTypeShards:
		return `redispool.Get(database.REDIS_PLAYER)`
	default:
		return fmt.Sprintf(`redispool.Get("%s")`, m.DbName)
	}
}
func (m *RedisHash) BatchClientExpr() string {
	switch m.DbType {
	case DbTypeGlobal:
		if m.ShardField != nil {
			return fmt.Sprintf(`redispool.Get(%s)`, m.ShardField.Name)
		}
		return `redispool.Get(database.REDIS_GLOBAL)`
	case DbTypeShards:
		return `redispool.Get(database.REDIS_PLAYER)`
	default:
		return fmt.Sprintf(`redispool.Get("%s")`, m.DbName)
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
