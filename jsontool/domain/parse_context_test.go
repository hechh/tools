package domain

import "testing"

// TestParseEnum 测试枚举解析（E| 5 段格式）
func TestParseEnum(t *testing.T) {
	ctx := NewParseContext()
	ctx.ParseTable(&Table{
		Sheet: "enum",
		Token: 1,
		Rows: [][]string{
			{"E|金币|PropType|PT_Coin|1"},
			{"E|钻石|PropType|PT_Diamond|2"},
			{"E|青铜|RoomType|RT_Bronze|1"},
		},
	})

	if len(ctx.Enums) != 2 {
		t.Fatalf("期望2个枚举类型, 实际%d", len(ctx.Enums))
	}
	propType := ctx.EnumMap["PropType"]
	if propType == nil || len(propType.Values) != 2 {
		t.Fatalf("PropType枚举解析错误")
	}
	if item := propType.DescMap["金币"]; item == nil || item.Value != 1 || item.Name != "PT_Coin" {
		t.Fatalf("金币枚举值错误: %+v", item)
	}
	if item := propType.DescMap["钻石"]; item == nil || item.Value != 2 {
		t.Fatalf("钻石枚举值错误: %+v", item)
	}
}

// TestParseStruct 测试行式结构体解析（含索引规则）
func TestParseStruct(t *testing.T) {
	ctx := NewParseContext()
	ctx.ParseTable(&Table{
		Sheet: "房间配置",
		Type:  "PveRoomConfig",
		Token: 2,
		Rules: []string{"map:RoomId", "group:MaxPlayers"},
		Rows: [][]string{
			{"RoomId", "MaxPlayers", "EntryFee"},
			{"uint32", "int32", "Reward"},
			{"房间ID", "最大玩家数", "报名费"},
			{"5001", "5", "金币,50"},
			{"5002", "5", "钻石,100"},
		},
	})

	st := ctx.StructMap["PveRoomConfig"]
	if st == nil {
		t.Fatal("结构体未解析")
	}
	if len(st.FieldList) != 3 {
		t.Fatalf("期望3个字段, 实际%d", len(st.FieldList))
	}
	if st.FieldList[0].Name != "RoomId" || st.FieldList[0].OriginType != "uint32" {
		t.Fatalf("字段0解析错误: %+v", st.FieldList[0])
	}
	if len(st.IndexList) != 2 {
		t.Fatalf("期望2个索引, 实际%d", len(st.IndexList))
	}
	if st.IndexList[0].Type != "map" || st.IndexList[0].Name != "RoomId" {
		t.Fatalf("索引0错误: %+v", st.IndexList[0])
	}
	if len(st.Rows) != 2 {
		t.Fatalf("期望2行数据, 实际%d", len(st.Rows))
	}
}

// TestParseStructCol 测试列式结构体解析（@struct:col）
func TestParseStructCol(t *testing.T) {
	ctx := NewParseContext()
	ctx.ParseTable(&Table{
		Sheet: "列式配置",
		Type:  "ColConfig",
		Token: 3,
		Rows: [][]string{
			{"RoomId", "MaxPlayers", "EntryFee"},
			{"uint32", "int32", "Reward"},
			{"房间ID", "最大玩家数", "报名费"},
			{"5001", "5", "金币,50"},
			{"5002", "5", "钻石,100"},
		},
	})

	st := ctx.StructMap["ColConfig"]
	if st == nil {
		t.Fatal("结构体未解析")
	}
	if len(st.FieldList) != 3 {
		t.Fatalf("期望3个字段, 实际%d", len(st.FieldList))
	}
	if st.FieldList[1].Name != "MaxPlayers" || st.FieldList[1].OriginType != "int32" {
		t.Fatalf("字段1解析错误: %+v", st.FieldList[1])
	}
	if len(st.Rows) != 2 {
		t.Fatalf("期望2行数据, 实际%d", len(st.Rows))
	}
	if st.Rows[0][0] != "5001" {
		t.Fatalf("数据行错误: %+v", st.Rows[0])
	}
}

// TestParseStructCompositeIndex 测试组合索引规则（group:A@map:B）
func TestParseStructCompositeIndex(t *testing.T) {
	ctx := NewParseContext()
	ctx.ParseTable(&Table{
		Sheet: "关卡奖励",
		Type:  "StageRewardConfig",
		Token: 2,
		Rules: []string{"group:StageType@map:SubId"},
		Rows: [][]string{
			{"StageType", "SubId", "Reward"},
			{"int32", "int32", "Reward"},
			{"关卡类型", "子ID", "奖励"},
			{"1", "10", "金币,50"},
		},
	})

	st := ctx.StructMap["StageRewardConfig"]
	if st == nil {
		t.Fatal("结构体未解析")
	}
	if len(st.IndexList) != 1 {
		t.Fatalf("期望1个索引, 实际%d", len(st.IndexList))
	}
	idx := st.IndexList[0]
	if idx.Type != "group" || idx.Name != "StageType" {
		t.Fatalf("一级索引错误: %+v", idx)
	}
	if idx.Next == nil || idx.Next.Type != "map" || idx.Next.Name != "SubId" {
		t.Fatalf("二级索引错误: %+v", idx.Next)
	}
}

// TestParseStructSkipEmptyType 测试空类型列跳过
func TestParseStructSkipEmptyType(t *testing.T) {
	ctx := NewParseContext()
	ctx.ParseTable(&Table{
		Sheet: "配置",
		Type:  "TestConfig",
		Token: 2,
		Rows: [][]string{
			{"Id", "", "Name"},
			{"int32", "", "string"},
			{"ID", "", "名称"},
			{"1", "", "abc"},
		},
	})

	st := ctx.StructMap["TestConfig"]
	if st == nil {
		t.Fatal("结构体未解析")
	}
	if len(st.FieldList) != 2 {
		t.Fatalf("期望2个字段(空类型列跳过), 实际%d", len(st.FieldList))
	}
}
