package app

import (
	"bytes"
	"strings"
	"testing"
	"text/template"

	"github.com/hechh/tools/xlsxtool/domain"
	"github.com/hechh/tools/xlsxtool/internal"

	"github.com/iancoleman/strcase"
)

func newTestTpl(t *testing.T) *template.Template {
	t.Helper()
	tpl, err := template.New("templ").Funcs(template.FuncMap{
		"ToSnake":                     strcase.ToSnake,
		"ToSnakePkg":                  func(s string) string { return strcase.ToSnake(s) },
		"ToLowerCamel":                strcase.ToLowerCamel,
		"containerType":               containerType,
		"keyExpr":                     keyExpr,
		"rangeSearchExpr":             rangeSearchExpr,
		"rangeFieldNames":             rangeFieldNames,
		"compositeSearchExpr":         compositeSearchExpr,
		"compositeFieldNames":         compositeFieldNames,
		"compositeInnerContainerType": compositeInnerContainerType,
		"compositeContainerType":      compositeContainerType,
	}).Parse(internal.ConfigCodeTempl)
	if err != nil {
		t.Fatalf("parse template: %v", err)
	}
	return tpl
}

func execTpl(t *testing.T, tpl *template.Template, st *domain.Struct) string {
	t.Helper()
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, st); err != nil {
		t.Fatalf("execute template: %v", err)
	}
	return buf.String()
}

// TestRangeIndexMultiKey 测试多key range索引
func TestRangeIndexMultiKey(t *testing.T) {
	tpl := newTestTpl(t)
	st := &domain.Struct{
		Type: "SlotBetConfig",
		FieldList: []*domain.Field{
			{Name: "Level", Type: "int32"},
			{Name: "EnergyLowerValue", Type: "int64"},
		},
		IndexList: []*domain.Index{
			{Type: "map", Name: "Level", List: []*domain.Field{{Name: "Level", Type: "int32"}}},
			{Type: "range", Name: "LevelEnergyLowerValue", List: []*domain.Field{
				{Name: "Level", Type: "int32"},
				{Name: "EnergyLowerValue", Type: "int64"},
			}},
		},
	}

	output := execTpl(t, tpl, st)

	// Data struct 不包含 range 字段
	if strings.Contains(output, "levelEnergyLowerValue map[") {
		t.Error("Data struct should not contain range index field")
	}
	// Range 函数
	if !strings.Contains(output, "func RangeLevelEnergyLowerValue(level int32, energyLowerValue int64)") {
		t.Error("unexpected Range function signature")
	}
	// sort.Search
	if !strings.Contains(output, "search_level := sort.Search(len(list), func(i int) bool") {
		t.Error("missing level sort.Search")
	}
	if !strings.Contains(output, "search_energyLowerValue := sort.Search(len(list), func(i int) bool") {
		t.Error("missing energyLowerValue sort.Search")
	}
	// util.Min
	if !strings.Contains(output, "idx := util.Min(search_level, search_energyLowerValue) - 1") {
		t.Error("missing util.Min call")
	}
	// 边界保护
	if !strings.Contains(output, "if len(list) == 0") {
		t.Error("missing empty list check")
	}
	if !strings.Contains(output, "if idx < 0") {
		t.Error("missing idx < 0 check")
	}
}

// TestRangeIndexSingleKey 测试单key range索引
func TestRangeIndexSingleKey(t *testing.T) {
	tpl := newTestTpl(t)
	st := &domain.Struct{
		Type: "TestConfig",
		FieldList: []*domain.Field{
			{Name: "Level", Type: "int32"},
		},
		IndexList: []*domain.Index{
			{Type: "range", Name: "Level", List: []*domain.Field{{Name: "Level", Type: "int32"}}},
		},
	}

	output := execTpl(t, tpl, st)

	// 直接用 search_level - 1，不用 util.Min
	if !strings.Contains(output, "idx := search_level - 1") {
		t.Error("single key should use 'search_level - 1' instead of util.Min")
	}
	if strings.Contains(output, "util.Min") {
		t.Error("single key range should not use util.Min")
	}
}

// ============== 组合规则测试 ==============

// TestGroupRangeComposite 测试 group@range 组合规则
func TestGroupRangeComposite(t *testing.T) {
	tpl := newTestTpl(t)
	st := &domain.Struct{
		Type: "SymbolLayerConfig",
		FieldList: []*domain.Field{
			{Name: "ConfigId", Type: "int32"},
			{Name: "Level", Type: "int32"},
			{Name: "Energy", Type: "int32"},
		},
		IndexList: []*domain.Index{
			{
				Type: "group",
				Name: "ConfigId",
				List: []*domain.Field{{Name: "ConfigId", Type: "int32"}},
				Next: &domain.Index{
					Type: "range",
					Name: "Level,Energy",
					List: []*domain.Field{
						{Name: "Level", Type: "int32"},
						{Name: "Energy", Type: "int32"},
					},
				},
			},
		},
	}

	output := execTpl(t, tpl, st)

	// 基础 group 方法仍然生成
	if !strings.Contains(output, "func GGetConfigId(configId int32) []*pb.SymbolLayerConfig") {
		t.Error("missing base GGet method")
	}
	if !strings.Contains(output, "func GWalkConfigId") {
		t.Error("missing GWalk method")
	}
	// 组合方法名和签名
	if !strings.Contains(output, "func GGetConfigIdRangeLevelEnergy(configId int32, level int32, energy int32)") {
		t.Errorf("unexpected composite method signature, got:\n%s", output)
	}
	// items 从 map 取出
	if !strings.Contains(output, "items, ok := data.configId[configId]") {
		t.Error("missing items extraction from group map")
	}
	// sort.Search 在 items 上操作
	if !strings.Contains(output, "search_level := sort.Search(len(items), func(i int) bool") {
		t.Error("missing items-based sort.Search for level")
	}
	if !strings.Contains(output, "search_energy := sort.Search(len(items), func(i int) bool") {
		t.Error("missing items-based sort.Search for energy")
	}
	// util.Min 多key
	if !strings.Contains(output, "idx := util.Min(search_level, search_energy) - 1") {
		t.Error("missing util.Min for multi-key range")
	}
	// 边界保护
	if !strings.Contains(output, "if !ok {\n\t\treturn nil\n\t}") {
		t.Error("missing nil check for ok")
	}
	if !strings.Contains(output, "if idx < 0") {
		t.Error("missing idx < 0 check")
	}
	if !strings.Contains(output, "return proto.Clone(items[idx]).(*pb.SymbolLayerConfig)") {
		t.Error("missing return proto.Clone(items[idx])")
	}
}

// TestGroupMapComposite 测试 group@map 组合规则
func TestGroupMapComposite(t *testing.T) {
	tpl := newTestTpl(t)
	st := &domain.Struct{
		Type: "StageRewardConfig",
		FieldList: []*domain.Field{
			{Name: "StageType", Type: "int32"},
			{Name: "SubId", Type: "int32"},
		},
		IndexList: []*domain.Index{
			{
				Type: "group",
				Name: "StageType",
				List: []*domain.Field{{Name: "StageType", Type: "int32"}},
				Next: &domain.Index{
					Type: "map",
					Name: "SubId",
					List: []*domain.Field{{Name: "SubId", Type: "int32"}},
				},
			},
		},
	}

	output := execTpl(t, tpl, st)

	// 嵌套 map 容器声明（使用 Map 后缀字段名避免与基础容器冲突）
	if !strings.Contains(output, "stageTypeMap map[int32]map[int32]*pb.StageRewardConfig") {
		t.Errorf("missing nested map container declaration")
	}
	// parse 初始化嵌套 map（使用 Map 后缀字段名）
	if !strings.Contains(output, `data.stageTypeMap[item.StageType] = make(map[int32]*pb.StageRewardConfig)`) {
		t.Error("missing nested map init in parse")
	}
	// 数据填充使用双层 key
	if !strings.Contains(output, "data.stageTypeMap[item.StageType][item.SubId]") {
		t.Error("missing double key data insertion")
	}
	// 组合方法
	if !strings.Contains(output, "func GGetStageTypeMapSubId(stageType int32, subId int32)") {
		t.Errorf("missing GGetStageTypeMapSubId method")
	}
	// 方法内使用嵌套map字段名
	if !strings.Contains(output, "innerMap, ok := data.stageTypeMap[stageType]") {
		t.Error("missing innerMap extraction with correct field name")
	}
	if !strings.Contains(output, "return proto.Clone(item).(*pb.StageRewardConfig)") {
		t.Error("missing proto.Clone return")
	}
}

// TestMapMapComposite 测试 map@map 组合规则
func TestMapMapComposite(t *testing.T) {
	tpl := newTestTpl(t)
	st := &domain.Struct{
		Type: "ItemDropConfig",
		FieldList: []*domain.Field{
			{Name: "SceneId", Type: "int32"},
			{Name: "ItemId", Type: "int32"},
		},
		IndexList: []*domain.Index{
			{
				Type: "map",
				Name: "SceneId",
				List: []*domain.Field{{Name: "SceneId", Type: "int32"}},
				Next: &domain.Index{
					Type: "map",
					Name: "ItemId",
					List: []*domain.Field{{Name: "ItemId", Type: "int32"}},
				},
			},
		},
	}

	output := execTpl(t, tpl, st)

	// 嵌套容器（使用 Map 后缀字段名）
	if !strings.Contains(output, "sceneIdMap map[int32]map[int32]*pb.ItemDropConfig") {
		t.Errorf("missing nested map container")
	}
	// MGet 方法
	if !strings.Contains(output, "func MGetSceneIdMapItemId(sceneId int32, itemId int32)") {
		t.Errorf("missing MGetSceneIdMapItemId method")
	}
	// 两级查找（使用 Map 后缀字段名）
	if !strings.Contains(output, "innerMap := data.sceneIdMap[sceneId]") {
		t.Error("missing outer map lookup with correct field name")
	}
	if !strings.Contains(output, "if innerMap == nil") {
		t.Error("missing nil check for innerMap")
	}
	if !strings.Contains(output, "return proto.Clone(item).(*pb.ItemDropConfig)") {
		t.Error("missing proto.Clone return")
	}
}

// TestMapGroupComposite 测试 map@group 组合规则
func TestMapGroupComposite(t *testing.T) {
	tpl := newTestTpl(t)
	st := &domain.Struct{
		Type: "MonsterWaveConfig",
		FieldList: []*domain.Field{
			{Name: "MapId", Type: "int32"},
			{Name: "WaveType", Type: "int32"},
		},
		IndexList: []*domain.Index{
			{
				Type: "map",
				Name: "MapId",
				List: []*domain.Field{{Name: "MapId", Type: "int32"}},
				Next: &domain.Index{
					Type: "group",
					Name: "WaveType",
					List: []*domain.Field{{Name: "WaveType", Type: "int32"}},
				},
			},
		},
	}

	output := execTpl(t, tpl, st)

	// map@group 返回分组列表
	if !strings.Contains(output, "func MGetMapIdGroupWaveType(mapId int32, waveType int32) []*pb.MonsterWaveConfig") {
		t.Errorf("missing MGetMapIdGroupWaveType method")
	}
}
