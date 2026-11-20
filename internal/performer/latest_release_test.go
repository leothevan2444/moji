package performer

import (
	stashboxgraphql "github.com/leothevan2444/moji/pkg/stashbox/graphql"
	"testing"
	"time"
)

func TestLatestReleaseUsesValidSourceDates(t *testing.T) {
	str := func(s string) *string { return &s }
	svc := &Service{timezoneProvider: func() string { return "Asia/Shanghai" }}
	now := time.Date(2026, 11, 9, 17, 0, 0, 0, time.UTC)
	scenes := []*stashboxgraphql.SceneFragment{
		{Title: str("old"), Date: str("2026-09-01")},
		{Title: str("future"), Date: str("2026-11-11")},
		{Title: str("unknown"), Date: str("invalid")}, nil,
		{Code: str("ABC-010"), Title: str("released today in configured timezone"), Date: str("2026-11-10")},
		{Title: str("older at end"), Date: str("2026-10-01")},
	}
	got := svc.latestSourceRelease(scenes, now)
	if got == nil || got.Code != "ABC-010" || got.Date != "2026-11-10" {
		t.Fatalf("latest=%+v", got)
	}
	if got := svc.latestSourceRelease(scenes[1:4], now); got != nil {
		t.Fatalf("future/unknown dates became releases: %+v", got)
	}
}
