package graphqlapi

import (
	"context"

	"github.com/leothevan2444/moji/internal/film"
	"github.com/leothevan2444/moji/internal/graphqlapi/model"
	"github.com/leothevan2444/moji/internal/taskflow"
	"github.com/leothevan2444/moji/internal/taskruntime"
)

type filmAdmission interface {
	ResolveFilm(context.Context, string, string) (film.Snapshot, error)
	ConfirmFilm(context.Context, taskflow.ConfirmFilmInput) (*taskruntime.Task, error)
}

func filmToModel(v *film.Snapshot) *model.TaskFilm {
	if v == nil {
		return nil
	}
	names := []string{}
	for _, p := range v.Performers {
		names = append(names, p.Name)
	}
	return &model.TaskFilm{Endpoint: v.Endpoint, SceneID: v.SceneID, Code: v.Code, Title: v.Title, Date: v.Date, Studio: v.Studio.Name, Performers: names, ImageURL: v.ImageURL, URL: v.URL, DurationSeconds: v.DurationSeconds, ConfirmationDigest: v.Digest()}
}

func filmText(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
