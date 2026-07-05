package main

import (
	"os"
	"testing"
	"time"

	"coffee-ranker/entity"
)

// 2時前は当日の2時を次回実行時刻にすることを検証する。
func TestNextDailyBatchTime_BeforeBatchHour(t *testing.T) {
	location := mustLoadLocation(t, "Asia/Tokyo")
	now := time.Date(2026, 7, 6, 1, 0, 0, 0, location)
	want := time.Date(2026, 7, 6, 2, 0, 0, 0, location)

	got := nextDailyBatchTime(now, entity.RankingBatchHour)
	if !got.Equal(want) {
		t.Fatalf("nextDailyBatchTime() = %s, want %s", got.Format(time.RFC3339), want.Format(time.RFC3339))
	}
}

// 2時ちょうどは当日分を過ぎた扱いにし、翌日の2時を次回実行時刻にすることを検証する。
func TestNextDailyBatchTime_AtBatchHour(t *testing.T) {
	location := mustLoadLocation(t, "Asia/Tokyo")
	now := time.Date(2026, 7, 6, 2, 0, 0, 0, location)
	want := time.Date(2026, 7, 7, 2, 0, 0, 0, location)

	got := nextDailyBatchTime(now, entity.RankingBatchHour)
	if !got.Equal(want) {
		t.Fatalf("nextDailyBatchTime() = %s, want %s", got.Format(time.RFC3339), want.Format(time.RFC3339))
	}
}

// 2時を過ぎている場合は翌日の2時を次回実行時刻にすることを検証する。
func TestNextDailyBatchTime_AfterBatchHour(t *testing.T) {
	location := mustLoadLocation(t, "Asia/Tokyo")
	now := time.Date(2026, 7, 6, 3, 0, 0, 0, location)
	want := time.Date(2026, 7, 7, 2, 0, 0, 0, location)

	got := nextDailyBatchTime(now, entity.RankingBatchHour)
	if !got.Equal(want) {
		t.Fatalf("nextDailyBatchTime() = %s, want %s", got.Format(time.RFC3339), want.Format(time.RFC3339))
	}
}

// BATCH_TIMEZONEが未指定の場合にAsia/Tokyoを使うことを検証する。
func TestBatchLocation_DefaultsToAsiaTokyo(t *testing.T) {
	oldValue, existed := os.LookupEnv("BATCH_TIMEZONE")
	if err := os.Unsetenv("BATCH_TIMEZONE"); err != nil {
		t.Fatalf("unset BATCH_TIMEZONE: %v", err)
	}
	t.Cleanup(func() {
		if existed {
			_ = os.Setenv("BATCH_TIMEZONE", oldValue)
			return
		}
		_ = os.Unsetenv("BATCH_TIMEZONE")
	})

	got := batchLocation()
	if got.String() != "Asia/Tokyo" {
		t.Fatalf("batchLocation() = %s, want Asia/Tokyo", got.String())
	}
}

// 不正なhourはRankingBatchHourへ戻して次回実行時刻を計算することを検証する。
func TestNextDailyBatchTime_InvalidHourFallsBackToRankingBatchHour(t *testing.T) {
	location := mustLoadLocation(t, "Asia/Tokyo")
	now := time.Date(2026, 7, 6, 1, 0, 0, 0, location)
	want := time.Date(2026, 7, 6, entity.RankingBatchHour, 0, 0, 0, location)

	cases := []struct {
		name string
		hour int
	}{
		{name: "negative", hour: -1},
		{name: "over 23", hour: 24},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := nextDailyBatchTime(now, tt.hour)
			if !got.Equal(want) {
				t.Fatalf("nextDailyBatchTime() = %s, want %s", got.Format(time.RFC3339), want.Format(time.RFC3339))
			}
		})
	}
}

func mustLoadLocation(t *testing.T, name string) *time.Location {
	t.Helper()

	location, err := time.LoadLocation(name)
	if err != nil {
		t.Fatalf("load location %s: %v", name, err)
	}
	return location
}
