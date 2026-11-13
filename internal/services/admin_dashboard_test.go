package services

import (
	"net/url"
	"testing"
	"time"
)

func TestDashboardPeriod(t *testing.T) {
	now := time.Date(2026, 3, 1, 22, 0, 0, 0, time.UTC) // March 2 in Nairobi.
	period, err := ParseDashboardPeriod(url.Values{}, now)
	if err != nil || period.Start != "2026-02-01" || period.End != "2026-03-02" {
		t.Fatalf("default period: %+v %v", period, err)
	}
	for _, query := range []string{
		"start=2026-02-01", "end=2026-02-01", "start=2026-02-30&end=2026-03-01",
		"start=2026-03-02&end=2026-03-01", "start=2024-01-01&end=2025-01-01",
		"start=2026-01-01&start=2026-02-01&end=2026-03-01", "start=garbage&end=garbage",
	} {
		values, _ := url.ParseQuery(query)
		if _, err := ParseDashboardPeriod(values, now); err == nil {
			t.Fatalf("accepted invalid range %q", query)
		}
	}
	values, _ := url.ParseQuery("start=2024-02-29&end=2024-02-29")
	period, err = ParseDashboardPeriod(values, now)
	if err != nil || period.StartAt.UTC().Format(time.RFC3339) != "2024-02-28T21:00:00Z" || period.EndAt.UTC().Format(time.RFC3339) != "2024-02-29T21:00:00Z" {
		t.Fatalf("inclusive local date bounds: %+v %v", period, err)
	}
}
