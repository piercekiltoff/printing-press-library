// Copyright 2026 trevin-chow. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"math"
	"testing"
)

func TestComputeStack_NoInputs(t *testing.T) {
	got := computeStack(stackable{}, 45.00, false)
	if got.RecommendedCouponID != nil {
		t.Errorf("no coupons: want nil ID, got %v", got.RecommendedCouponID)
	}
	if got.FinalTotal != 45.00 {
		t.Errorf("no inputs: want final 45.00, got %v", got.FinalTotal)
	}
	if got.TotalSavings != 0 {
		t.Errorf("no inputs: want 0 savings, got %v", got.TotalSavings)
	}
}

func TestComputeStack_PicksHighestEligibleCoupon(t *testing.T) {
	s := stackable{
		Coupons: []couponShape{
			{ID: 1, Value: 5, MinOrder: 0},
			{ID: 2, Value: 10, MinOrder: 50}, // not eligible at 45
			{ID: 3, Value: 8, MinOrder: 30},
		},
		Credit: 0,
	}
	got := computeStack(s, 45.00, false)
	if got.RecommendedCouponID != 3 {
		t.Errorf("want coupon 3 (8.00 with MinOrder 30), got %v", got.RecommendedCouponID)
	}
	if got.CouponValue != 8.00 {
		t.Errorf("coupon value: want 8.00, got %v", got.CouponValue)
	}
	if got.FinalTotal != 37.00 {
		t.Errorf("final total: want 37.00, got %v", got.FinalTotal)
	}
}

func TestComputeStack_AppliesCreditAfterCoupon(t *testing.T) {
	s := stackable{
		Coupons: []couponShape{{ID: "abc", Value: 5, MinOrder: 0}},
		Credit:  20,
	}
	got := computeStack(s, 45.00, false)
	if got.CouponValue != 5 {
		t.Errorf("coupon value: %v", got.CouponValue)
	}
	if got.CreditUsed != 20 {
		t.Errorf("credit used: %v", got.CreditUsed)
	}
	if got.FinalTotal != 20.00 {
		t.Errorf("final total: %v", got.FinalTotal)
	}
	// Invariant: total_savings + final_total == orderTotal
	if math.Abs(got.TotalSavings+got.FinalTotal-45.00) > 0.01 {
		t.Errorf("invariant broken: savings %v + final %v != 45.00", got.TotalSavings, got.FinalTotal)
	}
}

func TestComputeStack_CreditCappedAtRemaining(t *testing.T) {
	// $40 credit on a $30 cart should only consume $30.
	s := stackable{Credit: 40}
	got := computeStack(s, 30.00, false)
	if got.CreditUsed != 30 {
		t.Errorf("credit should cap at order total: got %v", got.CreditUsed)
	}
	if got.FinalTotal != 0 {
		t.Errorf("expected 0 final, got %v", got.FinalTotal)
	}
}

func TestComputeStack_ExperimentalWarning(t *testing.T) {
	got := computeStack(stackable{}, 25.00, true)
	if got.Warning == "" {
		t.Errorf("experimental mode should set warning")
	}
}

func TestPickBestCoupon_RespectsMinOrder(t *testing.T) {
	cs := []couponShape{
		{ID: 1, Value: 20, MinOrder: 100},
		{ID: 2, Value: 5, MinOrder: 0},
	}
	idx := pickBestCoupon(cs, 45.00)
	if idx != 1 {
		t.Errorf("want index 1 (under-MinOrder filter), got %d", idx)
	}
}

func TestPickBestCoupon_NoMatch(t *testing.T) {
	cs := []couponShape{
		{ID: 1, Value: 20, MinOrder: 100},
	}
	if got := pickBestCoupon(cs, 45.00); got != -1 {
		t.Errorf("want -1, got %d", got)
	}
}
