package graphqlapi

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/leothevan2444/moji/internal/mediafiles"
	"github.com/leothevan2444/moji/internal/taskruntime"
)

type fileResolutionRuntime struct {
	fakeTaskRuntime
	indices []int
}

func (f *fileResolutionRuntime) ResolveTaskFiles(_ context.Context, id string, indices []int) (*taskruntime.Task, error) {
	f.indices = append([]int(nil), indices...)
	plan := mediafiles.Plan{Files: []mediafiles.File{{Index: 7, Path: "unknown.mkv", Kind: mediafiles.Main, Part: 1, Selected: true, Reason: "USER_SELECTED"}}, Confirmed: true}
	data, _ := json.Marshal(plan)
	return &taskruntime.Task{ID: id, Code: "OFJE-329", Stage: taskruntime.TaskStageDownloading, StageStatus: taskruntime.TaskStageStatusRunning, FilePlanJSON: string(data)}, nil
}
func TestFileSelectionGraphQLPreservesOrderAndExposesPlan(t *testing.T) {
	runtime := &fileResolutionRuntime{}
	resolver := NewResolver(nil, nil, nil, nil, "test")
	resolver.TaskRuntime = runtime
	var response struct {
		Data struct {
			ResolveTaskFiles struct {
				ID    string
				Files []struct {
					Index    int
					Kind     string
					Part     int
					Selected bool
				}
			}
		}
		Errors []struct{ Message string }
	}
	executeGraphQLInto(t, resolver, `mutation { resolveTaskFiles(id:"task-1",videoIndices:[12,7]) { id files {index kind part selected} } }`, &response)
	if len(response.Errors) > 0 {
		t.Fatalf("GraphQL failed: %+v", response.Errors)
	}
	if !reflect.DeepEqual(runtime.indices, []int{12, 7}) {
		t.Fatal("transport changed the merge order")
	}
	if response.Data.ResolveTaskFiles.ID != "task-1" || len(response.Data.ResolveTaskFiles.Files) != 1 || !response.Data.ResolveTaskFiles.Files[0].Selected {
		t.Fatalf("file plan missing: %+v", response)
	}
}
func TestIngestSubtitleSettingGraphQLContract(t *testing.T) {
	editor := &fakeSettingsEditor{updateIngestSnapshot: &SettingsSnapshot{Ingest: IngestSettingsSnapshot{KeepSubtitles: true, DeliveryMode: "TRANSFER"}}}
	resolver := NewResolver(nil, nil, nil, nil, "test")
	resolver.SettingsEditor = editor
	var response struct {
		Data struct {
			UpdateIngestSettings struct{ Ingest struct{ KeepSubtitles bool } }
		}
		Errors []struct{ Message string }
	}
	executeGraphQLInto(t, resolver, `mutation { updateIngestSettings(input:{keepSubtitles:true,deliveryMode:"TRANSFER",downloads:{qbRoot:"/q",mojiRoot:"/m"},library:{mojiRoot:"/library",stashRoot:"/stash"},transfer:{action:"COPY"}}){ingest{keepSubtitles}} }`, &response)
	if len(response.Errors) > 0 || !response.Data.UpdateIngestSettings.Ingest.KeepSubtitles || !editor.ingestInput.KeepSubtitles {
		t.Fatalf("subtitle setting lost: %+v %+v", response, editor.ingestInput)
	}
}
