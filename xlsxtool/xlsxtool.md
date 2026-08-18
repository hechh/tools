# xlsxtool 使用说明

## 概述

`xlsxtool` 是 Excel 配置表 → JSON / Proto / 查询代码的一站式生成工具，是 `xlsxtool` 的**完整替代版**（data/proto/code 三命令），**data 命令不依赖 .proto / .pb.go 编译链**，输出 JSON 数据文件。JSON 结构与 protobuf message（`XxxConfigAry`）同构，运行时可用 `json.Unmarshal` / `sonic.Unmarshal` 直接倒回 pb 结构，走现有 gen.go 查询层。

> 与 `xlsxtool` 的关系：xlsxtool 是 xlsxtool 的完整替代版（data/proto/code 三命令），唯一区别是 data 命令输出 JSON（无 proto 依赖）而非 `.conf`。验证无误后将删除 xlsxtool。

## 命令（三个子命令）

```bash
xlsxtool data  -x <xlsx目录> -o <json输出目录>
xlsxtool proto -x <xlsx目录> -o <proto输出目录>
xlsxtool code  -x <xlsx目录> -o <代码输出目录>
```

| 命令 | 产物 | 说明 |
|------|------|------|
| `data` | `XxxConfig.json` | JSON 数据（无 proto 依赖，可倒回 pb） |
| `proto` | `enum.gen.proto` + `table.gen.proto` | proto 定义（与 xlsxtool 输出内容一致） |
| `code` | `common/table/<snake>/*.gen.go` | Go 查询代码（与 xlsxtool 输出一致） |

> `proto` 命令会扫描输出目录中已有的 `.proto` 文件获取 package 信息与外部类型，因此应输出到已含旧 proto 的目录（如 `public/protocol`）。

## "生成表" Sheet 指令

与 xlsxtool 完全一致：

```
@enum|Sheet名
@struct|Sheet名@结构体名|索引规则1|索引规则2...
@struct:col|Sheet名@结构体名|索引规则1|索引规则2...
```

- `@enum`：枚举定义（`E|查找键|枚举类型|Proto名称|数值` 5 段）
- `@struct`：行式配置表（字段名在首行）
- `@struct:col`：列式配置表（字段名在首列，每行一个字段）
- 省略 `@结构体名` 时 Sheet 名即结构体名
- 索引规则（map/group/range）由 `code` 命令消费构建容器

## JSON 结构规范

```json
{
  "Ary": [
    {
      "RoomId": 5001,
      "MaxPlayers": 5,
      "EntryFee": { "PropType": 1, "Incr": 50 },
      "RewardList": [ { "PropType": 1, "Incr": 100 }, { "PropType": 2, "Incr": 200 } ],
      "StartTime": 1772409600,
      "Rate": [1, 2, 3]
    }
  ]
}
```

- **键名 = Excel 第 1 行字段名原样（PascalCase）**：匹配 pb.go json tag（`json:"RoomId,omitempty"`），`json.Unmarshal`/`sonic.Unmarshal` 直接倒回 `*pb.XxxConfigAry`
- **包装 `{"Ary":[...]}`**：与 `XxxConfigAry` 同构，可喂给 gen.go `parse(ary *pb.XxxConfigAry)`
- **空字段省略键**：与 omitempty 语义一致，倒回时为零值

## 类型系统

| Excel 写法 | JSON 值 | 数据示例 |
|-----------|---------|----------|
| `int`/`int8`/`int16`/`int32` | 数字 | `123` |
| `int64` | 数字 | `123456789` |
| `uint`/`uint8`/`uint16`/`uint32` | 数字 | `7` |
| `uint64` | 数字 | `9` |
| `float`/`float32` | 数字 | `1.5` |
| `double`/`float64` | 数字 | `3.25` |
| `bool` | true/false | `true` |
| `string` | 字符串 | `abc` |
| `timestamp` | 数字（Unix 秒，Asia/Shanghai） | `2026-03-31 12:00:00` |
| `Range64` | `{"Min":1,"Max":100}` | `1,100` 或 `1\|100` |
| `Range32` | `{"Min":1,"Max":100}` | `1,100` 或 `1\|100` |
| `Reward` | `{"PropType":1,"Incr":100}`（2 参数）<br>`{"PropType":1,"PropId":1001,"Incr":100}`（3 参数） | `金币,100` / `金币,1001,100` |
| 枚举（如 `PropType`） | 数字（枚举值） | `金币`（查 @enum 查找键） |
| `[]T` | 数组（`\|` 分隔）；空元素按零值转换（枚举→0），nil 转换结果（空 Reward/Range）跳过，空数组输出 `[]` | `1\|2\|3` |

## 运行时倒回 pb 示例

```go
// 业务服务加载 JSON 数据（fwatcher 或等值 loader）
data := &pb.PveRoomConfigAry{}
if err := json.Unmarshal(jsonBytes, data); err != nil { ... }
// 交给现有 gen.go 查询层
pve_room_config.Parse(data)
```
