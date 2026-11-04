package taskruntime

import (
	"context"
	"errors"
	"strings"

	"github.com/leothevan2444/moji/internal/indexer"
)

func (s *Service) resolveIndexedResource(ctx context.Context, handle string) (downloadedTorrentMetadata, string, error) {
	return s.resolveIndexedArtifact(ctx, handle, func(work context.Context) (indexer.Artifact, error) {
		if s.resourceFetcher == nil {
			return indexer.Artifact{}, errors.New("indexer resource fetcher unavailable")
		}
		return s.resourceFetcher.FetchReference(work, strings.TrimPrefix(handle, "indexer-resource:"))
	})
}

func (s *Service) resolveIndexedArtifact(ctx context.Context, handle string, acquire func(context.Context) (indexer.Artifact, error)) (downloadedTorrentMetadata, string, error) {
	key := normalizeTorrentInspectionCacheKey(handle, "")
	s.inspectionCacheMu.Lock()
	entry, ok := s.inspectionCache[key]
	s.inspectionCacheMu.Unlock()
	if ok && entry.expiresAt.After(s.now().UTC()) {
		if entry.metadata != nil {
			return *entry.metadata, "", nil
		}
		if entry.magnet != "" {
			return downloadedTorrentMetadata{}, entry.magnet, nil
		}
	}
	artifact, err := acquire(ctx)
	if err != nil {
		return downloadedTorrentMetadata{}, "", err
	}
	if artifact.Magnet != "" {
		if normalizeMagnetURI(artifact.Magnet) == "" {
			return downloadedTorrentMetadata{}, "", errors.New("invalid indexer magnet")
		}
		s.storeCachedArtifact(handle, nil, artifact.Magnet)
		return downloadedTorrentMetadata{}, artifact.Magnet, nil
	}
	if len(artifact.TorrentBytes) == 0 || len(artifact.TorrentBytes) > MaxTorrentBytes {
		return downloadedTorrentMetadata{}, "", errors.New("invalid indexer torrent size")
	}
	metadata, err := parseTorrentMetadata(artifact.TorrentBytes)
	if err != nil {
		return downloadedTorrentMetadata{}, "", err
	}
	s.storeCachedArtifact(handle, &metadata, "")
	return metadata, "", nil
}

func (s *Service) storeCachedArtifact(key string, metadata *downloadedTorrentMetadata, magnet string) {
	next := cachedTorrentInspection{metadata: metadata, magnet: magnet, expiresAt: s.now().UTC().Add(torrentInspectionCacheTTL)}
	if metadata != nil {
		next.inspection = inspectTorrentMetadata(*metadata)
	}
	s.storeInspectionEntry(key, next)
}

func (s *Service) storeInspectionEntry(key string, next cachedTorrentInspection) {
	s.inspectionCacheMu.Lock()
	defer s.inspectionCacheMu.Unlock()
	size := func(e cachedTorrentInspection) int {
		n := len(e.magnet) + 1024
		if e.metadata != nil {
			n += len(e.metadata.Data) + len(e.metadata.Name)
			for _, paths := range [][]string{e.metadata.Paths, e.metadata.IdentityNames, e.metadata.IdentityPaths} {
				for _, path := range paths {
					n += len(path) + 16
				}
			}
		}
		return n
	}
	used := 0
	for id, e := range s.inspectionCache {
		if !e.expiresAt.After(s.now().UTC()) {
			delete(s.inspectionCache, id)
		} else {
			used += size(e)
		}
	}
	for len(s.inspectionCache) >= 256 || used+size(next) > 32<<20 {
		oldest := ""
		for id, e := range s.inspectionCache {
			if oldest == "" || e.expiresAt.Before(s.inspectionCache[oldest].expiresAt) {
				oldest = id
			}
		}
		if oldest == "" {
			return
		}
		used -= size(s.inspectionCache[oldest])
		delete(s.inspectionCache, oldest)
	}
	s.inspectionCache[normalizeTorrentInspectionCacheKey(key, "")] = next
}
