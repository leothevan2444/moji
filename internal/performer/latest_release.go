package performer

import (
	"time"

	"github.com/leothevan2444/moji/internal/releasedate"
	stashboxgraphql "github.com/leothevan2444/moji/pkg/stashbox/graphql"
)

// Use the source date before library merging or subscription policy filtering.
// An unknown date is never replaced by a discovery or library import timestamp.
func (s *Service) latestSourceRelease(scenes []*stashboxgraphql.SceneFragment, now time.Time) *LatestRelease {
	zone := ""
	if s.timezoneProvider != nil {
		zone = s.timezoneProvider()
	}
	var latest *LatestRelease
	for _, scene := range scenes {
		if scene == nil || releasedate.Check(stringValue(scene.Date), now, zone) != nil {
			continue
		}
		date, _ := releasedate.Parse(stringValue(scene.Date))
		candidate := LatestRelease{Code: stringValue(scene.Code), Title: stringValue(scene.Title), Date: date}
		if latest == nil || candidate.Date > latest.Date || (candidate.Date == latest.Date && candidate.Code+candidate.Title < latest.Code+latest.Title) {
			latest = &candidate
		}
	}
	return latest
}
