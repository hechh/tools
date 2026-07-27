# pbtool — .pb.go 辅助方法生成器

## 概述

pbtool 扫描 `.pb.go` 文件，自动为 protobuf message 生成三类辅助方法：

- **`*Rsp` 类型** → `SetRspHead` / `GetRspHead`（响应头快捷操作）
- **其他所有类型**（排除 `*Rsp`/`*Req`/`*Config`/`*ConfigAry`/`*ConfigS`） → `ToDB` / `FromDB`（proto 序列化/反序列化）
- **`*Config` 类型**（配置表） → `XxxConfigS` 只读包装体 + Getter 方法

输出文件：`{src目录}/common.gen.pb.go`

---

## 生成的方法

### SetRspHead / GetRspHead

为所有以 `Rsp` 结尾且包含 `.RspHead` 类型字段的 struct 生成：

```go
func (d *XxxRsp) SetRspHead(code int32, msg string) {
    d.Head = &packet.RspHead{Code: code, Msg: msg}
}
func (d *XxxRsp) GetRspHead() (int32, string) {
    return d.Head.Code, d.Head.Msg
}
```

工具通过检测字段类型名是否以 `.RspHead` 结尾来自动匹配字段（不限字段名）。

### ToDB / FromDB

为所有**非** `*Rsp`/`*Req`/`*Config`/`*ConfigAry`/`*ConfigS` 的 struct 生成：

```go
func (d *Xxx) ToDB() ([]byte, error) {
    if d == nil { return nil, nil }
    return proto.Marshal(d)
}
func (d *Xxx) FromDB(val []byte) error {
    if len(val) <= 0 { return nil }
    return proto.Unmarshal(val, d)
}
```

### ConfigS 只读包装

为所有以 `Config` 结尾（但不以 `ConfigAry` 或 `ConfigS` 结尾）的配置 struct 生成只读包装体：

```go
type XxxConfigS struct {
    inner *XxxConfig
}
```

并生成所有导出字段的 Getter 方法：

- **普通字段**（基本类型、枚举等）：直接返回 `s.inner.FieldName`
- **`*Reward` 字段**：返回 `s.inner.FieldName.CloneVT()` 深拷贝
- **`[]*Reward` 字段**：返回新的切片，每个元素通过 `CloneVT()` 深拷贝

```go
// 示例：普通字段
func (s *PveRoomConfigS) GetRoomId() uint32 {
    return s.inner.RoomId
}

// 示例：*Reward 字段 — 深拷贝
func (s *PveRoomConfigS) GetEntryFee() *Reward {
    return s.inner.EntryFee.CloneVT()
}

// 示例：[]*Reward 字段 — 深拷贝
func (s *PveRankRewardConfigS) GetRankPrizes() []*Reward {
    if s.inner.RankPrizes == nil { return nil }
    rets := make([]*Reward, len(s.inner.RankPrizes))
    for i, v := range s.inner.RankPrizes {
        rets[i] = v.CloneVT()
    }
    return rets
}
```

**设计意图**：配置表数据在内存中是全局共享的只读缓存。通过 `ConfigS` 包装，Reward 类型字段自动进行 `CloneVT` 深拷贝，确保业务层无论如何修改返回值，都不会污染全局配置缓存。

---

## 使用方法

```bash
# 直接运行
go run server/framework/tools/pbtool/ -src=server/common/pb

# 通过 Makefile（make pb 自动包含此步骤）
make pb
```

| 参数 | 说明 | 示例 |
|------|------|------|
| `-src` | `.pb.go` 文件所在目录 | `server/common/pb` |

> `make pb` 的执行顺序：**先 `protoc` 生成 `.pb.go` → 再 `pbtool` 生成 `common.gen.pb.go`**

---

## 排除规则

以下后缀的 struct **不会**生成 `ToDB`/`FromDB`：

| 后缀 | 原因 |
|------|------|
| `Rsp` | 响应体，不直接存 DB |
| `Req` | 请求体，不直接存 DB |
| `Config` | 配置表（由 xlsxtool 管理，生成 ConfigS 包装） |
| `ConfigAry` | 配置表数组（由 xlsxtool 管理） |
| `ConfigS` | 只读包装体（pbtool 自己生成，避免重复） |

`SetRspHead`/`GetRspHead` **仅**为 `*Rsp` 类型生成。

`ConfigS` 只读包装 **仅**为 `*Config` 类型生成（排除 `ConfigAry`）。

---

## 常见问题

| 问题 | 解决 |
|------|------|
| 新增 Rsp 后没有 SetRspHead | 确保 Rsp struct 中包含 `packet.RspHead` 类型字段；运行 `make pb` |
| 新增 Config 后没有 ConfigS | 运行 `make pb`，pbtool 会自动为所有 Config 生成只读包装 |
| ToDB/FromDB 未生成 | 检查 struct 名称是否以 `Rsp`/`Req`/`Config`/`ConfigAry`/`ConfigS` 结尾（这些会被跳过） |
| common.gen.pb.go 过期 | 重新运行 `make pb`，pbtool 每次都会覆盖生成 |
| ConfigS 的 Get 方法没有 CloneVT | 检查字段类型名是否为 `Reward`（裸名）或 `pkg.Reward`（选择器名）；pbtool 会自动识别两种形式 |
