# Discover

Discover creates download tasks from either structured StashBox metadata or raw Jackett results. Press `/` to focus the search field. Search history is stored in the browser and can be removed from the dropdown.

## StashBox search

Use StashBox for release codes, titles, or performer names. Moji queries configured StashBox endpoints in the priority saved under **Settings > Stash**. It stops at the first endpoint that returns results; the source badge identifies that endpoint.

Results may include the release code, studio, performers, date, duration, image, and source page. **Add to task queue** resolves the selected scene and creates a standard task. A usable release code is required for automatic torrent sourcing. Duplicate release codes, torrents, and codes already present in Stash are rejected.

## Jackett search

Jackett is the fallback and power-user view. Results are torrent candidates, not verified scene metadata. You can filter by configured indexers and sort or paginate the result drawer. Indexers that Jackett reports as unconfigured cannot be selected.

**Create task** uses the chosen result directly, bypassing automatic candidate choice for that task. Review the title, tracker, size, seeders, and source before submitting it. Magnet and original-download links are exposed separately when available.

## Upcoming releases

Below Search, upcoming releases found for subscribed performers are ordered by expected date. **View all** provides code/title search, performer filters and 7/30-day windows. Filters are preserved in the URL.

Unreleased films do not create download tasks or search indexers. On the release date, Moji checks current metadata and policy before downloading, requesting review or skipping the release. Changed dates reschedule checks; unavailable sources retain their last confirmed information. Records without any currently subscribed performer are hidden and paused.

Open a film for a player-style detail view, source link and refresh action. Trailers are not connected yet. Updating before release does not create a task; updating on or after release can start processing according to policy. Manual updates remain available when automatic checks are disabled. This list contains releases found by subscription checks, rather than a guaranteed complete catalog. Run a performer check if necessary.

Dates follow the Moji runtime system timezone by default, including a container’s TZ. Override it under **Settings > System > General** or with YAML `system.release_timezone`. An empty value or `auto` follows the system. Browser timezones do not change eligibility. Raw Jackett results and code-only inputs do not provide a trusted release date and continue to use the resources explicitly submitted by the user.
