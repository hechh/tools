package infra

import (
	"reflect"
	"testing"
	"time"

	"github.com/hechh/tools/xlsxtool/domain"
)

// TestConvertBasic 测试内置标量类型转换
func TestConvertBasic(t *testing.T) {
	cases := []struct {
		origin, val string
		want        any
	}{
		{"int32", "123", int32(123)},
		{"int", "-5", int32(-5)},
		{"int8", "8", int32(8)},
		{"int16", "16", int32(16)},
		{"int64", "123456789", int64(123456789)},
		{"uint32", "7", uint32(7)},
		{"uint", "8", uint32(8)},
		{"uint8", "9", uint32(9)},
		{"uint16", "10", uint32(10)},
		{"uint64", "11", uint64(11)},
		{"float", "1.5", float32(1.5)},
		{"float32", "2.5", float32(2.5)},
		{"double", "3.25", float64(3.25)},
		{"float64", "4.5", float64(4.5)},
		{"bool", "true", true},
		{"bool", "false", false},
		{"string", "abc", "abc"},
	}
	for _, c := range cases {
		got, err := domain.Convert(c.origin, c.val)
		if err != nil {
			t.Fatalf("%s(%s) 转换失败: %v", c.origin, c.val, err)
		}
		if got != c.want {
			t.Fatalf("%s(%s) 期望 %v(%T), 实际 %v(%T)", c.origin, c.val, c.want, c.want, got, got)
		}
	}
}

// TestConvertTimestamp 测试日期字符串 → Unix 秒
func TestConvertTimestamp(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 3, 31, 12, 0, 0, 0, loc).Unix()

	got, err := domain.Convert("timestamp", "2026-03-31 12:00:00")
	if err != nil {
		t.Fatalf("timestamp 转换失败: %v", err)
	}
	if got != want {
		t.Fatalf("timestamp 期望 %d, 实际 %v", want, got)
	}
}

// TestConvertRange 测试区间类型（Range64/Range32，逗号与竖线分隔）
func TestConvertRange(t *testing.T) {
	cases := []struct {
		origin, val string
		want        any
	}{
		{"Range64", "1,100", map[string]any{"Min": int64(1), "Max": int64(100)}},
		{"Range64", "1|100", map[string]any{"Min": int64(1), "Max": int64(100)}},
		{"Range32", "5,9", map[string]any{"Min": int32(5), "Max": int32(9)}},
		{"Range32", "5|9", map[string]any{"Min": int32(5), "Max": int32(9)}},
	}
	for _, c := range cases {
		got, err := domain.Convert(c.origin, c.val)
		if err != nil {
			t.Fatalf("%s(%s) 转换失败: %v", c.origin, c.val, err)
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Fatalf("%s(%s) 期望 %v, 实际 %v", c.origin, c.val, c.want, got)
		}
	}
}

// TestConvertReward 测试奖励类型（2/3 参数模式，PropType 为枚举）
func TestConvertReward(t *testing.T) {
	// 注册测试用 PropType 枚举转换器（实际由 GenJSON 注册）
	domain.RegisterConvertor("PropType", func(val string) (any, error) {
		switch val {
		case "金币":
			return 1, nil
		case "钻石":
			return 2, nil
		}
		return 0, nil
	}, "PropType")

	// 2 参数模式：PropType, Incr（不产 PropId）
	got, err := domain.Convert("Reward", "金币,100")
	if err != nil {
		t.Fatalf("Reward 转换失败: %v", err)
	}
	want := map[string]any{"PropType": 1, "Incr": int64(100)}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Reward(2参数) 期望 %v, 实际 %v", want, got)
	}

	// 3 参数模式：PropType, PropId, Incr
	got, err = domain.Convert("Reward", "钻石,1001,200")
	if err != nil {
		t.Fatalf("Reward 转换失败: %v", err)
	}
	want = map[string]any{"PropType": 2, "PropId": uint32(1001), "Incr": int64(200)}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Reward(3参数) 期望 %v, 实际 %v", want, got)
	}
}

// TestConvertUnregistered 测试未注册类型报错
func TestConvertUnregistered(t *testing.T) {
	if _, err := domain.Convert("NotExistType", "1"); err == nil {
		t.Fatal("未注册类型应返回错误")
	}
}
