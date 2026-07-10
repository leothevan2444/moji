package subscription

import (
	"github.com/leothevan2444/moji/internal/taskflow"
)

// newDefaultTaskCreator keeps subscription's default wiring behind a small local
// seam so the domain layer does not need to assemble taskflow inputs inline.
func newDefaultTaskCreator(taskRuntime TaskRuntime, stashbox *stashboxRegistry) TaskCreator {
	creator := taskflow.NewService(taskRuntime)
	creator.SetDiscoveredSceneResolver(NewDiscoveredSceneResolver(stashbox))
	return creator
}
