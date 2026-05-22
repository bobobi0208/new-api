package service

import (
	"testing"

	"github.com/QuantumNous/new-api/model"
)

func TestApplyTier_Basic(t *testing.T) {
	tiers := []model.CommissionTier{
		{From: 0, RateBP: 1000},   // 10%
		{From: 100, RateBP: 2000}, // 20% beyond 100
	}
	// 50 quota 全部落 0-100 区间，应得 5
	if got := ApplyTier(50, tiers); got != 5 {
		t.Fatalf("expect 5, got %d", got)
	}
	// 100 quota: 100*0.10 = 10
	if got := ApplyTier(100, tiers); got != 10 {
		t.Fatalf("expect 10, got %d", got)
	}
	// 200 quota: 100*0.10 + 100*0.20 = 30
	if got := ApplyTier(200, tiers); got != 30 {
		t.Fatalf("expect 30, got %d", got)
	}
}

func TestApplyTier_EmptyAndNegative(t *testing.T) {
	if got := ApplyTier(0, DefaultCommissionTiers); got != 0 {
		t.Fatalf("zero consume should produce zero, got %d", got)
	}
	if got := ApplyTier(-5, DefaultCommissionTiers); got != 0 {
		t.Fatalf("negative consume should produce zero, got %d", got)
	}
	if got := ApplyTier(1000, nil); got != 0 {
		t.Fatalf("nil tiers should produce zero, got %d", got)
	}
	if got := ApplyTier(1000, []model.CommissionTier{}); got != 0 {
		t.Fatalf("empty tiers should produce zero, got %d", got)
	}
}

func TestApplyTier_Capped(t *testing.T) {
	// 阶梯里有 9999bp（99.99%）也应被压到 CommissionMaxRateBP=2000bp
	tiers := []model.CommissionTier{
		{From: 0, RateBP: 9999},
	}
	if got := ApplyTier(100, tiers); got != 20 {
		t.Fatalf("expect capped at 20%% -> 20, got %d", got)
	}
}

func TestApplyTier_Unsorted(t *testing.T) {
	// 入参乱序，函数内部会排序
	tiers := []model.CommissionTier{
		{From: 100, RateBP: 2000},
		{From: 0, RateBP: 1000},
	}
	if got := ApplyTier(200, tiers); got != 30 {
		t.Fatalf("expect 30 even when unsorted, got %d", got)
	}
}

func TestApplyTier_DefaultTiers(t *testing.T) {
	// $50,000 = 25,000,000,000 quota
	// 落档：
	//   0 - 10000*500000 = 5e9       @ 10%  -> 500_000_000
	//   5e9 - 1.5e10                 @ 12%  -> 1_200_000_000
	//   1.5e10 - 2.5e10              @ 15%  -> 1_500_000_000
	// 合计 3_200_000_000 quota
	got := ApplyTier(25_000_000_000, DefaultCommissionTiers)
	if got != 3_200_000_000 {
		t.Fatalf("expect 3_200_000_000, got %d", got)
	}
}

func TestApplyTier_DefaultTiers_Top(t *testing.T) {
	// $200,000 = 1e11 quota
	// 落档：
	//   0 - 5e9              @ 10%  -> 5e8
	//   5e9 - 1.5e10         @ 12%  -> 1.2e9
	//   1.5e10 - 5e10        @ 15%  -> 5.25e9
	//   5e10 - 1e11          @ 18%  -> 9e9
	// 合计 1.595e10
	got := ApplyTier(100_000_000_000, DefaultCommissionTiers)
	expect := int64(500_000_000 + 1_200_000_000 + 5_250_000_000 + 9_000_000_000)
	if int64(got) != expect {
		t.Fatalf("expect %d, got %d", expect, got)
	}
}

func TestMonthRange_Valid(t *testing.T) {
	start, end, err := MonthRange("2026-05")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if end <= start {
		t.Fatalf("end %d must be after start %d", end, start)
	}
	// 一个月差应该是 28-31 天
	d := (end - start) / 86400
	if d < 28 || d > 31 {
		t.Fatalf("month span looks wrong: %d days", d)
	}
}

func TestMonthRange_Invalid(t *testing.T) {
	cases := []string{"", "2026", "2026-5", "2026/05", "abcd-ef"}
	for _, s := range cases {
		if _, _, err := MonthRange(s); err == nil {
			t.Fatalf("expected error for input %q", s)
		}
	}
}
