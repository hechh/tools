package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hechh/tools/jsontool/domain"
	"github.com/hechh/tools/jsontool/infra"

	"github.com/xuri/excelize/v2"
)

// mockReward 模拟 pb.Reward 的 json tag（PascalCase），验证 JSON 可倒回 pb 结构
type mockReward struct {
	PropType int32  `json:"PropType,omitempty"`
	PropId   uint32 `json:"PropId,omitempty"`
	Incr     int64  `json:"Incr,omitempty"`
}

// mockPveRoomConfig 模拟 pb.PveRoomConfig
type mockPveRoomConfig struct {
	RoomId     uint32        `json:"RoomId,omitempty"`
	MaxPlayers int32         `json:"MaxPlayers,omitempty"`
	EntryFee   *mockReward   `json:"EntryFee,omitempty"`
	RewardList []*mockReward `json:"RewardList,omitempty"`
	StartTime  int64         `json:"StartTime,omitempty"`
	Rate       []int32       `json:"Rate,omitempty"`
}

// mockPveRoomConfigAry 模拟 pb.PveRoomConfigAry
type mockPveRoomConfigAry struct {
	Ary []*mockPveRoomConfig `json:"Ary,omitempty"`
}

func saveTestFile(dir, filename string, data []byte) error {
	return os.WriteFile(filepath.Join(dir, filename), data, 0o644)
}

// buildTestXlsx 构造测试 xlsx：生成表指令 + 枚举 + 行式结构表
func buildTestXlsx(t *testing.T, dir string) {
	t.Helper()
	f := excelize.NewFile()

	if _, err := f.NewSheet("生成表"); err != nil {
		t.Fatal(err)
	}
	f.SetSheetRow("生成表", "A1", &[]interface{}{"@enum|EnumSheet"})
	f.SetSheetRow("生成表", "A2", &[]interface{}{"@struct|房间配置@PveRoomConfig|map:RoomId|group:MaxPlayers"})
	f.DeleteSheet("Sheet1")

	if _, err := f.NewSheet("EnumSheet"); err != nil {
		t.Fatal(err)
	}
	f.SetSheetRow("EnumSheet", "A1", &[]interface{}{"E|金币|PropType|PT_Coin|1"})
	f.SetSheetRow("EnumSheet", "A2", &[]interface{}{"E|钻石|PropType|PT_Diamond|2"})

	if _, err := f.NewSheet("房间配置"); err != nil {
		t.Fatal(err)
	}
	f.SetSheetRow("房间配置", "A1", &[]interface{}{"RoomId", "MaxPlayers", "EntryFee", "RewardList", "StartTime", "Rate"})
	f.SetSheetRow("房间配置", "A2", &[]interface{}{"uint32", "int32", "Reward", "[]Reward", "timestamp", "[]int32"})
	f.SetSheetRow("房间配置", "A3", &[]interface{}{"房间ID", "最大玩家数", "报名费", "奖励列表", "开始时间", "掉率"})
	f.SetSheetRow("房间配置", "A4", &[]interface{}{"5001", "5", "金币,50", "金币,100|钻石,200", "2026-03-31 12:00:00", "1|2|3"})
	f.SetSheetRow("房间配置", "A5", &[]interface{}{"5002", "5", "钻石,100", "金币,50", "2026-04-01 12:00:00", "4|5|6"})

	if err := f.SaveAs(filepath.Join(dir, "test.xlsx")); err != nil {
		t.Fatal(err)
	}
}

// parseTestXlsx 完整解析测试 xlsx → ParseContext
func parseTestXlsx(t *testing.T, dir string) *domain.ParseContext {
	t.Helper()
	tables := infra.ReadTables(dir)
	if len(tables) != 2 {
		t.Fatalf("期望2个表(1枚举+1结构), 实际%d", len(tables))
	}
	ctx := domain.NewParseContext()
	for _, tb := range tables {
		ctx.ParseTable(tb)
	}
	return ctx
}

// TestGenJSON 集成测试：xlsx → JSON 结构断言
func TestGenJSON(t *testing.T) {
	dir := t.TempDir()
	buildTestXlsx(t, dir)
	ctx := parseTestXlsx(t, dir)

	out := t.TempDir()
	if err := GenJSON(ctx, out, saveTestFile); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(out, "PveRoomConfig.json"))
	if err != nil {
		t.Fatalf("读取生成文件失败: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("JSON 解析失败: %v", err)
	}
	ary, ok := got["Ary"].([]any)
	if !ok || len(ary) != 2 {
		t.Fatalf("Ary 结构错误: %+v", got)
	}
	row0 := ary[0].(map[string]any)
	if row0["RoomId"] != float64(5001) {
		t.Fatalf("RoomId 期望 5001, 实际 %v", row0["RoomId"])
	}
	if row0["MaxPlayers"] != float64(5) {
		t.Fatalf("MaxPlayers 期望 5, 实际 %v", row0["MaxPlayers"])
	}
	entryFee := row0["EntryFee"].(map[string]any)
	if entryFee["PropType"] != float64(1) || entryFee["Incr"] != float64(50) {
		t.Fatalf("EntryFee 期望 {PropType:1,Incr:50}, 实际 %v", entryFee)
	}
	// 2 参数 Reward 不产 PropId 键
	if _, ok := entryFee["PropId"]; ok {
		t.Fatalf("2参数Reward不应有PropId键: %v", entryFee)
	}
	rewardList := row0["RewardList"].([]any)
	if len(rewardList) != 2 {
		t.Fatalf("RewardList 期望2项, 实际 %d", len(rewardList))
	}
	if rewardList[0].(map[string]any)["PropType"] != float64(1) ||
		rewardList[1].(map[string]any)["PropType"] != float64(2) ||
		rewardList[1].(map[string]any)["Incr"] != float64(200) {
		t.Fatalf("RewardList 解析错误: %v", rewardList)
	}
	rate := row0["Rate"].([]any)
	if len(rate) != 3 || rate[2] != float64(3) {
		t.Fatalf("Rate 期望 [1,2,3], 实际 %v", rate)
	}
}

// TestJSONRoundTrip 倒回验证：生成的 JSON 可 json.Unmarshal 进模拟 pb 结构
func TestJSONRoundTrip(t *testing.T) {
	dir := t.TempDir()
	buildTestXlsx(t, dir)
	ctx := parseTestXlsx(t, dir)

	out := t.TempDir()
	if err := GenJSON(ctx, out, saveTestFile); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(out, "PveRoomConfig.json"))
	if err != nil {
		t.Fatalf("读取生成文件失败: %v", err)
	}

	var got mockPveRoomConfigAry
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("倒回 pb 结构失败: %v", err)
	}
	if len(got.Ary) != 2 {
		t.Fatalf("Ary 期望2条, 实际%d", len(got.Ary))
	}
	row0 := got.Ary[0]
	if row0.RoomId != 5001 || row0.MaxPlayers != 5 {
		t.Fatalf("标量倒回错误: %+v", row0)
	}
	if row0.EntryFee == nil || row0.EntryFee.PropType != 1 || row0.EntryFee.Incr != 50 || row0.EntryFee.PropId != 0 {
		t.Fatalf("EntryFee 倒回错误(PropId应为0): %+v", row0.EntryFee)
	}
	if len(row0.RewardList) != 2 ||
		row0.RewardList[0].PropType != 1 || row0.RewardList[0].Incr != 100 ||
		row0.RewardList[1].PropType != 2 || row0.RewardList[1].Incr != 200 {
		t.Fatalf("RewardList 倒回错误: %+v", row0.RewardList)
	}
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatal(err)
	}
	wantStart := time.Date(2026, 3, 31, 12, 0, 0, 0, loc).Unix()
	if row0.StartTime != wantStart {
		t.Fatalf("StartTime 期望 %d, 实际 %d", wantStart, row0.StartTime)
	}
	if len(row0.Rate) != 3 || row0.Rate[2] != 3 {
		t.Fatalf("Rate 倒回错误: %v", row0.Rate)
	}
}

// TestGenJSONCol 集成测试：@struct:col 列式表（每行一个字段，GetCols 转置）
func TestGenJSONCol(t *testing.T) {
	dir := t.TempDir()
	f := excelize.NewFile()

	if _, err := f.NewSheet("生成表"); err != nil {
		t.Fatal(err)
	}
	f.SetSheetRow("生成表", "A1", &[]interface{}{"@enum|EnumSheet"})
	f.SetSheetRow("生成表", "A2", &[]interface{}{"@struct:col|列式配置@ColConfig|map:RoomId"})
	f.DeleteSheet("Sheet1")

	if _, err := f.NewSheet("EnumSheet"); err != nil {
		t.Fatal(err)
	}
	f.SetSheetRow("EnumSheet", "A1", &[]interface{}{"E|金币|PropType|PT_Coin|1"})

	// 列式布局：每行一个字段（A=字段名、B=类型、C=描述、D起=数据）
	if _, err := f.NewSheet("列式配置"); err != nil {
		t.Fatal(err)
	}
	f.SetSheetRow("列式配置", "A1", &[]interface{}{"RoomId", "uint32", "房间ID", "5001", "5002"})
	f.SetSheetRow("列式配置", "A2", &[]interface{}{"MaxPlayers", "int32", "最大玩家数", "5", "5"})
	f.SetSheetRow("列式配置", "A3", &[]interface{}{"EntryFee", "Reward", "报名费", "金币,50", "钻石,100"})

	if err := f.SaveAs(filepath.Join(dir, "test.xlsx")); err != nil {
		t.Fatal(err)
	}

	ctx := parseTestXlsx(t, dir)
	out := t.TempDir()
	if err := GenJSON(ctx, out, saveTestFile); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(out, "ColConfig.json"))
	if err != nil {
		t.Fatalf("读取生成文件失败: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	ary := got["Ary"].([]any)
	if len(ary) != 2 {
		t.Fatalf("Ary 期望2条, 实际%d", len(ary))
	}
	row0 := ary[0].(map[string]any)
	if row0["RoomId"] != float64(5001) || row0["MaxPlayers"] != float64(5) {
		t.Fatalf("列式表标量解析错误: %v", row0)
	}
	entryFee := row0["EntryFee"].(map[string]any)
	if entryFee["PropType"] != float64(1) || entryFee["Incr"] != float64(50) {
		t.Fatalf("列式表 EntryFee 错误: %v", entryFee)
	}
}

// TestGenJSONEnumFallback 测试枚举未命中时兜底为 0
func TestGenJSONEnumFallback(t *testing.T) {
	dir := t.TempDir()
	f := excelize.NewFile()

	if _, err := f.NewSheet("生成表"); err != nil {
		t.Fatal(err)
	}
	f.SetSheetRow("生成表", "A1", &[]interface{}{"@enum|EnumSheet"})
	f.SetSheetRow("生成表", "A2", &[]interface{}{"@struct|测试表@TestConfig|map:Id"})
	f.DeleteSheet("Sheet1")

	if _, err := f.NewSheet("EnumSheet"); err != nil {
		t.Fatal(err)
	}
	f.SetSheetRow("EnumSheet", "A1", &[]interface{}{"E|金币|PropType|PT_Coin|1"})

	if _, err := f.NewSheet("测试表"); err != nil {
		t.Fatal(err)
	}
	f.SetSheetRow("测试表", "A1", &[]interface{}{"Id", "Type"})
	f.SetSheetRow("测试表", "A2", &[]interface{}{"int32", "PropType"})
	f.SetSheetRow("测试表", "A3", &[]interface{}{"ID", "类型"})
	f.SetSheetRow("测试表", "A4", &[]interface{}{"1", "未知枚举"})

	if err := f.SaveAs(filepath.Join(dir, "test.xlsx")); err != nil {
		t.Fatal(err)
	}

	ctx := parseTestXlsx(t, dir)
	out := t.TempDir()
	if err := GenJSON(ctx, out, saveTestFile); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(out, "TestConfig.json"))
	if err != nil {
		t.Fatalf("读取生成文件失败: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	row0 := got["Ary"].([]any)[0].(map[string]any)
	if row0["Type"] != float64(0) {
		t.Fatalf("枚举未命中应兜底0, 实际 %v", row0["Type"])
	}
}

// TestGenJSONNilConvert 测试转换结果为 nil 的字段/元素处理（与 xlsxtool 行为一致）：
// 空 Reward 字段省略、空 []Reward 输出 []、空 []枚举 输出 [0]
func TestGenJSONNilConvert(t *testing.T) {
	dir := t.TempDir()
	f := excelize.NewFile()

	if _, err := f.NewSheet("生成表"); err != nil {
		t.Fatal(err)
	}
	f.SetSheetRow("生成表", "A1", &[]interface{}{"@enum|EnumSheet"})
	f.SetSheetRow("生成表", "A2", &[]interface{}{"@struct|空值表@NilTestConfig|map:Id"})
	f.DeleteSheet("Sheet1")

	if _, err := f.NewSheet("EnumSheet"); err != nil {
		t.Fatal(err)
	}
	f.SetSheetRow("EnumSheet", "A1", &[]interface{}{"E|新手|SegmentRule|ActiveNewbie|1"})

	if _, err := f.NewSheet("空值表"); err != nil {
		t.Fatal(err)
	}
	f.SetSheetRow("空值表", "A1", &[]interface{}{"Id", "Reward", "Rewards", "Rules", "Note"})
	f.SetSheetRow("空值表", "A2", &[]interface{}{"int32", "Reward", "[]Reward", "[]SegmentRule", "string"})
	f.SetSheetRow("空值表", "A3", &[]interface{}{"ID", "奖励", "奖励列表", "规则列表", "备注"})
	f.SetSheetRow("空值表", "A4", &[]interface{}{"1", "", "", "", "备注"})

	if err := f.SaveAs(filepath.Join(dir, "test.xlsx")); err != nil {
		t.Fatal(err)
	}

	ctx := parseTestXlsx(t, dir)
	out := t.TempDir()
	if err := GenJSON(ctx, out, saveTestFile); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(out, "NilTestConfig.json"))
	if err != nil {
		t.Fatalf("读取生成文件失败: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	row0 := got["Ary"].([]any)[0].(map[string]any)
	if _, ok := row0["Reward"]; ok {
		t.Fatalf("空 Reward 字段应省略, 实际 %v", row0["Reward"])
	}
	rewards := row0["Rewards"].([]any)
	if len(rewards) != 0 {
		t.Fatalf("空 []Reward 应输出空数组, 实际 %v", rewards)
	}
	rules := row0["Rules"].([]any)
	if len(rules) != 1 || rules[0] != float64(0) {
		t.Fatalf("空 []枚举 应输出 [0], 实际 %v", rules)
	}
}
