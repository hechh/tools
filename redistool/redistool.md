# redistool — Redis 数据访问层代码生成器

## 概述

redistool 扫描 `.pb.go` 中 `@dbtool` 注解，自动生成 Redis String/Hash 的 CRUD 函数及可选的 Cache 层。核心理念：**proto 中声明 → 工具生成 → 业务直接调用**。

---

## @dbtool 注解语法

```
// @dbtool:<类型>|<DbSpec>|<Key格式>|<Field格式(仅Hash)>
```

### 类型标识

| 注解前缀 | 数据类型 | Cache |
|---------|---------|-------|
| `@dbtool:string` | Redis String | 无 |
| `@dbtool:string:cache` | Redis String | GetByCache / SetByCache |
| `@dbtool:hash` | Redis Hash | 无 |
| `@dbtool:hash:cache` | Redis Hash | HGetByCache / HSetByCache |

### DbSpec — 数据库规格

| 格式 | 说明 | 生成的 redispool 调用 |
|------|------|----------------------|
| `"Name"` | 静态数据库名 | `GetByName("Name")` |
| `global:ConstName` | 常量引用，需在 `database` 包存在 | `GetByName(database.ConstName)` |
| `global:Name@string` | 动态名称，参数类型必须为 `string` | `GetByName(Name)` |
| `shards:field@type` | 分片路由（**必须声明路由字段**） | `GetByUid(field)`，批量用 `GetById(shardId)` |

> **参数去重**：当 `shards:field@type` 的 field 与 Keys[0] 或 Fields[0] 同名（忽略大小写）时，生成的函数签名中只输出一次该参数，如 `Get(uid)` 而非 `Get(uid, uid)`。

### Key/Field 格式串

| 格式 | 示例 | 解析为 |
|------|------|--------|
| `prefix:Name@type` | `user_info:uid@uint64` | 格式串 `"user_info:%d"`，字段 `[{uid, uint64}]` |
| `prefix:N1@t1,N2@t2` | `data:uid@uint64,SubId@int32` | 格式串 `"data:%d:%d"`，字段 `[{uid, uint64}, {SubId, int32}]` |
| 纯字符串（无 `@`） | `user_profile` | 格式串 `"user_profile"`，无动态字段 |

类型→占位符映射：
- `string` → `%s`
- 其他（`int64`/`uint64`/`int32`/`uint32`/`pb.RoomType` 等） → `%d`

### 格式校验

占位符数量与参数字段数量不一致时，**跳过该规则并打印警告**（不中断流程）：
```
[redistool] 跳过无效规则: @dbtool:xxx (结构体 Xxx): 格式串 "xxx:%d:%s" 需要 2 个参数但声明了 1 个
```

---

## 生成代码

### 文件输出

`{dstDir}/{snake_case(结构体名)}/{结构体名}.gen.go`，文件头含 `本代码由redistool工具生成，请勿手动修改`。

### String 生成的函数

| 函数 | 说明 |
|------|------|
| `GetKey(...)` | 构造 Redis Key |
| `Get(...)` | 读取 → crypto.Unmarshal → 返回 `*pb.Xxx`（优先走 ctx 缓存） |
| `Read(...)` | 🔴 纯缓存读取，不访问 Redis；缓存未命中直接 panic（必须先 Get/Change） |
| `Set(..., val, expiration)` | crypto.Marshal → 写入，expiration=0 永不过期 |
| `Change(...)` | 标记数据已变更（`ctx.Change(key)`），配合 `Set(nil)` 实现变更追踪回写 |
| `Del(...)` | 删除 key |
| `Expire(..., expiration)` | 设置过期 |
| `Remove([shardId,] keys ...)` | 批量删除 key，支持可变参数一次性删除多个 |
| `MSet(shardId, vals)` / `MGet(shardId, keys...)` | 批量读写（仅 shards，参数含 `shardId`） |
| `GetByCache(ctx, ...)` / `SetByCache(ctx, ...)` | 缓存穿透读/写（仅 `:cache`） |

### Hash 生成的函数

| 函数 | 说明 |
|------|------|
| `GetKey(...)` + `GetField(...)` | 构造 Redis Key 和 Hash Field |
| `HGet(...)` / `HSet(..., data)` / `HDel(...)` | 单 field 读写删 |
| `Read(...)` | 🔴 纯缓存读取，不访问 Redis；缓存未命中直接 panic（必须先 HGet/Change） |
| `Change(...)` | 标记单 field 数据已变更（`ctx.Change(key+field)`），配合 `HSet(nil)` 实现变更追踪 |
| `HGetAll(...)` | 读取整个 hash |
| `HLen(...)` | 获取 hash 字段数量 |
| `HMGet(..., fields...)` / `HMSet(..., data)` | 批量 field 读写 |
| `HRemove(..., fields ...)` | 批量删除 field，支持可变参数一次性删除多个 |
| `HExpire(..., expiration)` | hash key 过期 |
| `HGetByCache(ctx, ...)` / `HSetByCache(ctx, ...)` | 缓存穿透读/写（仅 `:cache`） |

### 变更追踪与缓存读取机制

所有 ctx 版本的生成函数共享同一套请求级缓存（`ctx.GetCache` / `ctx.SetCache` / `ctx.Change` / `ctx.IsChanged`）：

- **`Get` / `HGet`**：优先从 `ctx.GetCache(key)` 读取；未命中则查 Redis 并回写缓存
- **`Read`**：🔴 仅从 `ctx.GetCache(key)` 读取，**绝不访问 Redis**。缓存未命中时 `mlog.Panicf` 终止流程——调用方必须确保 `Get`/`HGet` 已先行加载数据
- **`Change`**：调用 `ctx.Change(key)` 标记该 key 已变更，用于后续 `Set(nil)` / `HSet(nil)` 变更追踪回写
- **`Set(nil)` / `HSet(nil)`**：仅当 `ctx.IsChanged(key)` 为 true 时才从缓存取数据写回 Redis，未标记变更则跳过

典型业务调用流程：
```go
// 1. 加载数据到缓存
data, _ := user_data.Get(ctx)       // Redis → cache

// 2. 修改数据
data.Star += 100

// 3. 标记变更
user_data.Change(ctx)               // ctx.Change(key)

// 4. 快速读取（不访问 Redis）
current := user_data.Read(ctx)      // 仅从缓存读，必定命中

// 5. 统一回写
user_data.Set(ctx, nil, 0)          // 仅当 IsChanged 时才真正写 Redis
```

- **`GetByCache`**：`ctx.GetCache(key)` 命中直接返回；未命中调 `Get()` 后 `ctx.SetCache(key, data)`
- **`SetByCache`**：`ctx.GetCache(key)` 有值则调 `Set()` 刷新 Redis
- `uid` 参数自动替换为 `ctx.GetUid()`；Hash 的 cache key = `GetKey(...) + GetField(...)`

### 生成的依赖

`core/define`、`core/library/crypto`、`core/pkg/mlog`、`core/pkg/redispool`、`richgame/server/common/pb`、`richgame/server/common/database`（仅 `global:ConstName`）、`google.golang.org/protobuf/proto`

---

## 使用方法

### 命令

```bash
# 直接运行
go run server/framework/tools/redistool/ -src=server/common/pb -dst=server/common/redis

# 通过 Makefile（推荐，会先清理旧生成文件）
make redistool
```

### 新增 Redis 模型的步骤

1. 在 proto 文件中定义 message，上方加 `@dbtool` 注解
2. `make pb` — 重新生成 `.pb.go`（**必须先执行，redistool 解析 Go 代码而非 proto**）
3. `make redistool` — 生成 Redis 访问层
4. 业务代码中 `import` 生成的包，调用生成函数

---

## 示例

### String + shards + cache

```protobuf
// @dbtool:string:cache|shards:uid@uint64|user_data:uid@uint64
message UserData { uint64 Uid = 1; int64 Star = 2; }
```
```go
// 生成
func GetKey(ctx define.IContext) string                        // "user_data:{uid}"
func Get(ctx define.IContext) (*pb.UserData, error)            // 优先缓存，未命中查 Redis
func Read(ctx define.IContext) *pb.UserData                    // 纯缓存读取，未命中 panic
func Set(ctx define.IContext, val *pb.UserData, expiration time.Duration) error
func Change(ctx define.IContext)                               // ctx.Change(key) 标记变更

// 调用
data, _ := user_data.Get(ctx)   // 加载
data.Star += 100
user_data.Change(ctx)           // 标记变更
cur := user_data.Read(ctx)      // 快速读取（必定命中）
user_data.Set(ctx, nil, 0)      // 回写（仅变更时）
```

### String + global 常量

```protobuf
// @dbtool:string|global:REDIS_PLAYER_CACHE|admin_session:Username@string
message AdminSession { string Username = 1; }
```
```go
func GetKey(Username string) string                     // "admin_session:{Username}"
func Get(Username string) (*pb.AdminSession, error)      // GetByName(database.REDIS_PLAYER_CACHE)
```

### Hash + shards

```protobuf
// @dbtool:hash|shards:uid@uint64|prize_record:uid@uint64|ContestId@string
message PrizeRecordData { string ContestId = 1; uint64 Uid = 2; }
```
```go
func GetKey(uid uint64) string          // "prize_record:{uid}"
func GetField(ContestId string) string  // "{ContestId}"
func HGet(uid uint64, ContestId string) (*pb.PrizeRecordData, error)
func HGetAll(uid uint64) (map[string]*pb.PrizeRecordData, error)
```

### Hash + global + 静态 key

```protobuf
// @dbtool:hash|global:REDIS_PLAYER_CACHE|user_profile|uid@uint64
message UserProfile { uint64 Uid = 1; string Name = 2; }
```
```go
func GetKey() string                     // "user_profile"（无参数）
func GetField(uid uint64) string         // "{uid}"
func HGetAll() (map[string]*pb.UserProfile, error)
```

---

## 常见问题

| 问题 | 原因 | 解决 |
|------|------|------|
| 注解写了不生成 | 未先 `make pb`；或 `-src` 目录不对；或注解格式错误 | 确认 `make pb` 已执行；检查 `[redistool] 跳过无效规则` 日志 |
| `格式串需要 N 个参数但声明了 M 个` | 占位符数量与 `Name@type` 数量不一致 | 对齐 `%s`/`%d` 数量与参数数量 |
| `Hash 类型必须使用4段格式` | Hash 注解只写了 3 段 | 补全：`@dbtool:hash\|DbSpec\|keyFmt\|fieldFmt` |
| `shards 模式缺少有效的参数声明` | 写了 `shards` 但没路由字段 | 改为 `shards:uid@uint64` |
