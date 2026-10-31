package releasedate

import (
	"testing"
	"time"
)

func TestCalendarGate(t *testing.T) {
	now := time.Date(2026, 10, 31, 16, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		date, zone string
		want       error
	}{{"2026-11-01", "Asia/Shanghai", nil}, {"2026-11-01", "UTC", ErrNotReleased}, {"2026-11-02", "Asia/Shanghai", ErrNotReleased}, {"", "UTC", ErrUnknown}, {"2026-02-30", "UTC", ErrUnknown}, {"2026-10-30T23:00:00Z", "UTC", nil}} {
		if got := Check(tc.date, now, tc.zone); got != tc.want {
			t.Errorf("%+v: %v", tc, got)
		}
	}
}
