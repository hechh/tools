# xlsxtool 使用说明

## 概述

`xlsxtool` 是 Excel 配置表 → Proto / 数据 / 查询代码的一站式生成工具。每个 `.xlsx` 文件必须包含一个 **"生成表"** Sheet。

```bash
make xlsx2proto  → ① 生成 enum.gen.proto / table.gen.proto
make pb          → ② 编译 proto → .pb.go
make xlsx2data   → ③ 生成 .conf 数据文件
make xlsx2code   → ④ 生成 .gen.go 查询代码
```

> 新增字段类型或枚举值必须从 ① 开始完整执行。

---

## 一、"生成表" Sheet 指令

```
@enum|Sheet名
@struct|Sheet名@结构体名|索引规则1|索引规则2...
@struct:col|Sheet名@结构体名|索引规则1|索引规则2...
```

| 指令 | 用途 |
|------|------|
| `@enum` | 定义枚举 |
| `@struct` | 行式配置表（字段名在首行） |
| `@struct:col` | 列式配置表（字段名在首列） |

`Sheet名@结构体名` 中，`@` 前是 Excel Sheet 名，`@` 后是 Proto message 名（PascalCase）。

---

## 二、枚举定义

### 2.1 格式

在对应 Sheet 中每行一个枚举值，5 个管道分隔字段：

```
E|查找键|枚举类型|Proto名称|数值
```

| 位置 | 字段 | 说明 |
|------|------|------|
| 1 | `E` | 固定标识 |
| 2 | 查找键 | **配置表数据中用来引用此枚举值的字符串**（`DescMap` 的 key） |
| 3 | 枚举类型 | 枚举 Proto 类型名（PascalCase，如 `PropType`） |
| 4 | Proto 名称 | 枚举值 Proto 名（全大写+下划线，如 `PT_Coin`） |
| 5 | 数值 | 整数值 |

### 2.2 示例

```
E|金币|PropType|PT_Coin|1
E|钻石|PropType|PT_Diamond|2
E|青铜|RoomType|RT_Bronze|1
E|白银|RoomType|RT_Silver|2
```

工具自动追加 `{EnumType}_None = 0` 作为默认值。一个 Sheet 可定义多种枚举类型，按枚举类型自动分组。

---

## 三、结构体定义（配置表）

### 3.1 行式布局（`@struct`）

| 行号 | 内容 | 示例 |
|------|------|------|
| 第 1 行 | 字段名（PascalCase） | `RoomId`, `MaxPlayers`, `EntryFee` |
| 第 2 行 | 字段类型 | `uint32`, `int32`, `Reward` |
| 第 3 行 | 中文描述 | `房间ID`, `最大玩家数`, `报名费` |
| 第 4 行起 | 数据行 | `5001`, `5`, `金币,50` |

### 3.2 示例

"生成表"中声明：

```
@struct|房间配置@PveRoomConfig|map:RoomId|group:MaxPlayers
```

对应 Sheet（`房间配置`）：

| RoomId | MaxPlayers | EntryFee |
|--------|-----------|-----------|
| uint32 | int32     | Reward   |
| 房间ID  | 最大玩家数  | 报名费    |
| 5001   | 5         | 金币,50   |
| 5002   | 5         | 钻石,100  |

---

## 四、字段类型系统

### 4.1 可用类型一览

| 类别 | Excel 写法 | 说明 | 数据示例 |
|------|-----------|------|----------|
| Proto 标量 | `int32`, `int64`, `uint32`, `uint64`, `float`, `double`, `bool`, `string`, `bytes` | 标准 Protobuf 类型（含 `sint*`/`fixed*`/`sfixed*` 变体） | `123`, `true` |
| 内置别名 | `int`/`int8`/`int16` → `int32`；`uint`/`uint8`/`uint16` → `uint32`；`float32` → `float`；`float64` → `double` | 均为别名，Proto 层映射到对应标量 | 同上 |
| `timestamp` | `timestamp` | 日期时间 → Unix 时间戳（int64） | `2026-03-31 12:00:00` |
| `Range32` | `Range32` | int32 区间 `{Min, Max}` | `1,100` 或 `1\|100` |
| `Range64` | `Range64` | int64 区间 `{Min, Max}` | `1,100` 或 `1\|100` |
| `Reward` | `Reward` | 奖励 `{PropType, Incr}` | `金币,1000` |
| 枚举 | `PropType` / `RoomType` 等 | 直接写枚举类型名 | `金币`（即枚举的查找键） |
| 外部 message | `&EntryFee` / `*SomeType` | `&` 或 `*` 前缀引用 `struct.proto` 中的 message | 由该类型的转换器决定 |

### 4.2 数组（repeated）

类型前加 `[]`，数据用 `|` 分隔元素：`[]int32` → `1|2|3`，`[]Reward` → `金币,100|钻石,200`。

---

## 五、枚举在配置表中的引用

1. **字段直接是枚举类型**：第 2 行写枚举类型名（如 `RoomType`），数据行写枚举的**查找键**（如 `青铜`）。
2. **枚举数组**：`[]PropType`，数据用 `|` 分隔（`金币|钻石`）。
3. **自定义类型中包含枚举**：如 `Reward` 内部有 `PropType` 字段，数据格式由 `Reward` 的转换器决定（`金币,1000`），转换器内部通过 `domain.Convert("PropType", "金币", ...)` 递归解析。

> Excel 中定义的枚举由 `gen_data.go` 自动注册转换器，无需手动处理。

---

## 六、新增自定义数据类型

### 6.1 步骤

1. 在 `public/protocol/struct.proto` 中定义 message。
2. 在 `server/framework/tools/xlsxtool/infra/builtin.go` 的 `init()` 中注册转换器。
3. 在 Excel 第 2 行使用注册时指定的类型名。

### 6.2 注册范式

```go
domain.RegisterConvertor(
    "ProtoMessage名",     // target: Proto 类型名
    func(val string, field protoreflect.FieldDescriptor, msg *dynamicpb.Message) error {
        // 解析 val → 构建子 message → 写入 msg
        // 子字段通过 domain.Convert("基础类型", 子值, 子field, 子msg) 递归转换
    },
    "Proto类型",           // protoType: 供 proto 生成使用
    "Excel中写的类型名",    // origins...: Excel 第 2 行可用的名称
)
```

### 6.3 转换函数要点

- 用 `dynamicpb.NewMessage(field.Message())` 创建子 message
- 子字段调用 `domain.Convert("int64"/"int32"/"string"/枚举类型名, 子值, ...)` 复用已有转换器
- 根据 `field.IsList()` 决定 `Append` 还是 `Set`

> 完整代码示例参考 `infra/builtin.go` 中的 `Reward`、`Range64` 实现或 `.agents/rules/config-conventions.md`。

---

## 七、索引规则

### 7.1 基础规则

| 规则 | 格式 | 查询方法 | 适用场景 |
|------|------|----------|----------|
| `map` | `map:字段名` | `MGet字段名(key) → *pb.Xxx` | 主键/唯一键精确查找 |
| `group` | `group:字段名` | `GGet字段名(key) → []*pb.Xxx` | 非唯一键分组 |
| `range` | `range:字段名1,字段名2` | `Range字段名(args) → *pb.Xxx` | 等级/区间匹配（二分查找，返回 ≤ 给定值的最大记录） |

`range` **要求数据行按 range 字段升序**。多字段时 key 用逗号分隔（`range:Level,Energy`）。

### 7.2 组合规则

用 `@` 连接两层索引：

| 组合 | 格式 | 查询方法 | 含义 |
|------|------|----------|------|
| `map@map` | `map:A@map:B` | `MGetAB(a, b) → *pb.Xxx` | 两级精确查找 |
| `map@group` | `map:A@group:B` | `MGetAGroupB(a, b) → []*pb.Xxx` | 先精确再分组 |
| `group@range` | `group:A@range:B` | `GGetARangeB(a, b) → *pb.Xxx` | 先分组再二分查找 |
| `group@map` | `group:A@map:B` | `GGetAMapB(a, b) → *pb.Xxx` | 先分组再精确查找 |

示例：`@struct|关卡奖励@StageRewardConfig|group:StageType@map:SubId`

```go
func GGetStageType(stageType int32) []*pb.StageRewardConfig           // 基础 group
func GGetStageTypeMapSubId(stageType int32, subId int32) *pb.StageRewardConfig  // 组合
```

### 7.3 通用查询方法（自动生成）

| 方法 | 说明 |
|------|------|
| `SGet(pos int) *pb.Xxx` | 按位置获取单条 |
| `LGet() []*pb.Xxx` | 获取全部（深拷贝） |
| `Walk(func(*pb.Xxx) bool)` | 遍历 |

---

## 八、命令参数

```bash
xlsxtool proto -x <xlsx目录> -o <proto输出目录>
xlsxtool data  -x <xlsx目录> -p <proto目录> -o <数据输出目录> [-i <额外import路径>]
xlsxtool code  -x <xlsx目录> -p <proto目录> -o <代码输出目录> [-i <额外import路径>]
```

---

## 九、产物一览

| 命令 | 产物 | 位置 |
|------|------|------|
| `proto` | `enum.gen.proto`, `table.gen.proto`（含 `XxxAry` 包装 message） | `public/protocol/` |
| `data` | `XxxConfig.conf`（prototext 格式） | `public/data/` |
| `code` | `XxxConfig.gen.go`（含 `parse`/`SGet`/`LGet`/`Walk`/`Change` + 索引方法） | `server/common/table/{snake_name}/` |
