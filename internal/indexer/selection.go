package indexer

import (
	"encoding/json"
	"time"
)

// SelectionCandidate is the projection consumed by Moji's existing ranking
// rules. Counts use their established -1 unknown sentinel at this boundary;
// Release retains nullable source observations. ResourceID is server-owned.
type SelectionCandidate struct {
	Source       *Provenance `json:"Source,omitempty"`
	Title        string      `json:"Title"`
	Tracker      string      `json:"Tracker"`
	TrackerID    string      `json:"TrackerId"`
	CategoryDesc string      `json:"CategoryDesc"`
	GUID         string      `json:"Guid"`
	Link         string      `json:"Link"`
	Details      string      `json:"Details"`
	PublishDate  string      `json:"PublishDate"`
	Category     []int       `json:"Category"`
	Size         int64       `json:"Size"`
	Seeders      int         `json:"Seeders"`
	Peers        int         `json:"Peers"`
	InfoHash     string      `json:"InfoHash"`
	MagnetURI    string      `json:"MagnetUri"`
	ResourceID   string      `json:"ResourceID,omitempty"`
}

// Provenance identifies the exact source of selected bytes. Credentials and
// signed locators are intentionally absent from this durable audit record.
type Provenance struct {
	Version                                                                                int
	Kind                                                                                   Kind
	InstanceID, ResourceID, BundleRevision, DefinitionID, DefinitionDigest, ConfigRevision string
}

func (r *SelectionCandidate) UnmarshalJSON(data []byte) error {
	type plain SelectionCandidate
	decoded := struct {
		*plain
		Seeders *int `json:"Seeders"`
	}{plain: (*plain)(r)}
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	r.Seeders = -1
	if decoded.Seeders != nil && *decoded.Seeders >= 0 {
		r.Seeders = *decoded.Seeders
	}
	return nil
}

func SelectionFromRelease(release Release) SelectionCandidate {
	out := SelectionCandidate{Title: release.Title, Tracker: release.InstanceName, TrackerID: release.InstanceID, GUID: release.GUID, Details: release.DetailsURL, Category: append([]int(nil), release.Categories...), Size: -1, Seeders: -1, Peers: -1, InfoHash: release.InfoHash, ResourceID: release.Resource.ID}
	if release.Size != nil {
		out.Size = *release.Size
	}
	if release.Seeders != nil {
		out.Seeders = *release.Seeders
	}
	if release.Peers != nil {
		out.Peers = *release.Peers
	}
	if release.PublishedAt != nil {
		out.PublishDate = release.PublishedAt.Format(time.RFC3339Nano)
	}
	ref := release.Resource
	out.Source = &Provenance{Version: 1, Kind: ref.Kind, InstanceID: ref.InstanceID, ResourceID: ref.ID, BundleRevision: ref.BundleRevision, DefinitionID: ref.DefinitionID, DefinitionDigest: ref.DefinitionDigest, ConfigRevision: ref.ConfigRevision}
	return out
}
