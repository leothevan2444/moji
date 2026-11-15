package metadata

import (
	"time"

	"github.com/leothevan2444/moji/internal/film"
	stashboxgraphql "github.com/leothevan2444/moji/pkg/stashbox/graphql"
)

// FilmSnapshot preserves a catalog observation for immediate task admission.
// Callers must not deserialize this transient observation from client input.
func FilmSnapshot(endpoint string, scene *stashboxgraphql.SceneFragment) *film.Snapshot {
	if scene == nil {
		return nil
	}
	value := func(v *string) string {
		if v == nil {
			return ""
		}
		return *v
	}
	v := &film.Snapshot{Endpoint: film.Endpoint(endpoint), SceneID: scene.ID, Code: value(scene.Code), Title: value(scene.Title), Date: value(scene.Date), DurationSeconds: scene.Duration, ConfirmedAt: time.Now().UTC()}
	if scene.Studio != nil {
		v.Studio = film.Person{ID: scene.Studio.ID, Name: scene.Studio.Name}
	}
	for _, p := range scene.Performers {
		if p != nil && p.Performer != nil {
			v.Performers = append(v.Performers, film.Person{ID: p.Performer.ID, Name: p.Performer.Name})
		}
	}
	if len(scene.Images) > 0 && scene.Images[0] != nil {
		v.ImageURL = scene.Images[0].URL
	}
	if len(scene.Urls) > 0 && scene.Urls[0] != nil {
		v.URL = scene.Urls[0].URL
	}
	return v
}
