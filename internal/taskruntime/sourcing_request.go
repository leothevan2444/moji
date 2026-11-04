package taskruntime

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/leothevan2444/moji/internal/tracker"
)

const maxSourcingRequestBytes = 64 << 10

// Version 1 persists qualified instance identities. Unversioned payloads were
// written before builtin sources existed: every tracker is a Jackett ID,
// including strings which happen to look like a builtin UUID.
type sourcingRequestPayload struct {
	Version int `json:"version"`
	DownloadRequest
}

func marshalSourcingRequest(request DownloadRequest) ([]byte, error) {
	// Keep the existing input bridge at admission, never infer the meaning of
	// an already persisted versioned identity while resuming a task.
	request.Trackers = tracker.InstanceIDsFromLegacyTrackers(request.Trackers)
	if err := validateSourcingInstanceIDs(request.Trackers); err != nil {
		return nil, err
	}
	data, err := json.Marshal(sourcingRequestPayload{Version: 1, DownloadRequest: request})
	if err != nil {
		return nil, err
	}
	if len(data) > maxSourcingRequestBytes {
		return nil, fmt.Errorf("sourcing request exceeds byte budget")
	}
	return data, nil
}

func unmarshalSourcingRequest(data []byte) (DownloadRequest, error) {
	var payload struct {
		Version json.RawMessage `json:"version"`
		DownloadRequest
	}
	trimmed := bytes.TrimSpace(data)
	if len(data) > maxSourcingRequestBytes || len(trimmed) == 0 || trimmed[0] != '{' {
		return DownloadRequest{}, fmt.Errorf("invalid sourcing request object")
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return DownloadRequest{}, err
	}
	if payload.Version == nil {
		for i, id := range payload.Trackers {
			payload.Trackers[i] = "jackett/" + id
		}
	} else {
		var version int
		if err := json.Unmarshal(payload.Version, &version); err != nil || version != 1 {
			return DownloadRequest{}, fmt.Errorf("unsupported sourcing request version")
		}
		if err := validateSourcingInstanceIDs(payload.Trackers); err != nil {
			return DownloadRequest{}, err
		}
	}
	return payload.DownloadRequest, nil
}

func validateSourcingInstanceIDs(ids []string) error {
	for _, id := range ids {
		if strings.HasPrefix(id, "jackett/") && len(id) > len("jackett/") {
			continue
		}
		if parsed, err := uuid.Parse(id); err != nil || parsed.String() != id {
			return fmt.Errorf("invalid sourcing instance identity")
		}
	}
	return nil
}
