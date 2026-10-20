package graphqlapi

import (
	"io"

	"github.com/99designs/gqlgen/graphql"
	"github.com/leothevan2444/moji/internal/taskruntime"
)

func readSourcingTorrentUpload(upload *graphql.Upload) ([]byte, error) {
	if upload == nil || upload.File == nil || upload.Size > taskruntime.MaxTorrentBytes {
		return nil, taskruntime.ErrInvalidTorrentInput
	}
	data, err := io.ReadAll(io.LimitReader(upload.File, taskruntime.MaxTorrentBytes+1))
	if err != nil || len(data) == 0 || len(data) > taskruntime.MaxTorrentBytes {
		return nil, taskruntime.ErrInvalidTorrentInput
	}
	return data, nil
}
