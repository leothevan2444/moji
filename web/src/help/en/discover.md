# Discover

Discover creates download tasks from either structured StashBox metadata or raw Jackett results. Press `/` to focus the search field. Search history is stored in the browser and can be removed from the dropdown.

## StashBox search

Use StashBox for release codes, titles, or performer names. Moji queries configured StashBox endpoints in the priority saved under **Settings > Automation**. It stops at the first endpoint that returns results; the source badge identifies that endpoint.

Results may include the release code, studio, performers, date, duration, image, and source page. **Add to task queue** resolves the selected scene and creates a standard task. A usable release code is required for automatic torrent sourcing. Duplicate release codes, torrents, and codes already present in Stash are rejected.

## Jackett search

Jackett is the fallback and power-user view. Results are torrent candidates, not verified scene metadata. You can filter by configured indexers and sort or paginate the result drawer. Indexers that Jackett reports as unconfigured cannot be selected.

**Create task** uses the chosen result directly, bypassing automatic candidate choice for that task. Review the title, tracker, size, seeders, and source before submitting it. Magnet and original-download links are exposed separately when available.
