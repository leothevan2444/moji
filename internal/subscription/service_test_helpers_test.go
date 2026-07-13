package subscription

import (
	"github.com/leothevan2444/moji/internal/metadata"
	"github.com/leothevan2444/moji/internal/taskflow"
)

func newServiceForTest(stash StashClient, registry *metadata.Registry, taskRuntime taskflow.TaskRuntime, store Store) (*Service, error) {
	loader, _ := any(stash).(metadata.EndpointLoader)
	return NewService(stash, metadata.NewService(loader, registry), taskflow.NewService(taskRuntime), store)
}
