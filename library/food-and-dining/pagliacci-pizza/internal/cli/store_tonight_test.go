// Copyright 2026 trevin-chow. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"encoding/json"
	"testing"
	"time"
)

func TestParseClockTime_Valid(t *testing.T) {
	loc, _ := time.LoadLocation("America/Los_Angeles")
	day := time.Date(2026, 4, 25, 0, 0, 0, 0, loc)
	cases := []struct {
		in   string
		hour int
		min  int
	}{
		{"11:00am", 11, 0},
		{"10:00pm", 22, 0},
		{"4:30pm", 16, 30},
	}
	for _, c := range cases {
		got, err := parseClockTime(c.in, day, loc)
		if err != nil {
			t.Errorf("parseClockTime(%q) error: %v", c.in, err)
			continue
		}
		if got.Hour() != c.hour || got.Minute() != c.min {
			t.Errorf("parseClockTime(%q) = %02d:%02d, want %02d:%02d", c.in, got.Hour(), got.Minute(), c.hour, c.min)
		}
	}
}

func TestParseClockTime_Empty(t *testing.T) {
	loc, _ := time.LoadLocation("America/Los_Angeles")
	if _, err := parseClockTime("", time.Now(), loc); err == nil {
		t.Errorf("expected error on empty input")
	}
}

func TestIsOpenNow(t *testing.T) {
	loc, _ := time.LoadLocation("America/Los_Angeles")
	now := time.Date(2026, 4, 25, 18, 0, 0, 0, loc) // 6 PM PT
	if !isOpenNow("11:00am", "10:00pm", now, loc) {
		t.Errorf("6pm should be open between 11am-10pm")
	}
	if isOpenNow("11:00am", "5:00pm", now, loc) {
		t.Errorf("6pm should NOT be open after 5pm close")
	}
	if isOpenNow("7:00pm", "10:00pm", now, loc) {
		t.Errorf("6pm should NOT be open before 7pm")
	}
}

func TestHoursForToday_FindsCorrectDay(t *testing.T) {
	storeRec := map[string]any{
		"Hours": []any{
			map[string]any{"Day": float64(0), "Open": "11:00am", "Close": "10:00pm"},
			map[string]any{"Day": float64(1), "Open": "11:00am", "Close": "11:00pm"},
		},
	}
	loc, _ := time.LoadLocation("America/Los_Angeles")
	// Sunday => Day 0
	sun := time.Date(2026, 4, 26, 12, 0, 0, 0, loc)
	o, c, ok := hoursForToday(storeRec, sun)
	if !ok || o != "11:00am" || c != "10:00pm" {
		t.Errorf("Sunday: got %q-%q ok=%v", o, c, ok)
	}
}

func TestExtractDeliveryMinutes(t *testing.T) {
	if got := extractDeliveryMinutes(json.RawMessage(`{"Delivery":"30"}`)); got != 30 {
		t.Errorf("string '30' -> %d", got)
	}
	if got := extractDeliveryMinutes(json.RawMessage(`{"Delivery":45}`)); got != 45 {
		t.Errorf("number 45 -> %d", got)
	}
	if got := extractDeliveryMinutes(json.RawMessage(`{}`)); got != 0 {
		t.Errorf("missing -> %d", got)
	}
}

func TestFindStoreInTimeWindowDays(t *testing.T) {
	loc, _ := time.LoadLocation("America/Los_Angeles")
	day := time.Date(2026, 4, 25, 12, 0, 0, 0, loc)
	raw := json.RawMessage(`[{"ID":20260425,"Day":"2026-04-25T00:00:00-07:00","Available":true},{"ID":20260426,"Available":false}]`)
	if !findStoreInTimeWindowDays(raw, day) {
		t.Errorf("expected today (2026-04-25) to be available")
	}
	day2 := time.Date(2026, 4, 26, 12, 0, 0, 0, loc)
	if findStoreInTimeWindowDays(raw, day2) {
		t.Errorf("expected 2026-04-26 to NOT be available")
	}
}

func TestCandidateStores_FilterByID(t *testing.T) {
	stores := json.RawMessage(`[
		{"ID":1,"Name":"A"},{"ID":2,"Name":"B"},{"ID":3,"Name":"C"}
	]`)
	got := candidateStores(stores, map[int]bool{2: true})
	if len(got) != 1 || extractInt(got[0], "ID") != 2 {
		t.Errorf("filter: got %+v", got)
	}
	all := candidateStores(stores, nil)
	if len(all) != 3 {
		t.Errorf("nil filter: want all 3, got %d", len(all))
	}
}
