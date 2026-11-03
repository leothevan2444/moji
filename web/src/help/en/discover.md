# Discover

Discover creates download tasks from either structured StashBox metadata or resources from built-in or external indexers. Press `/` to focus the search field. Search history is stored in the browser and can be removed from the dropdown.

## StashBox search

Use StashBox for release codes, titles, or performer names. Moji queries configured StashBox endpoints in the priority saved under **Settings > Stash**. It stops at the first endpoint that returns results; the source badge identifies that endpoint.

Results may include the release code, studio, performers, date, duration, image, and source page. **Add to task queue** resolves the selected scene and creates a standard task. A usable release code is required for automatic torrent sourcing. Duplicate release codes, torrents, and codes already present in Stash are rejected.

## Resource search

Choose **Indexers** to search enabled built-in and Jackett sites. You can select sites, sort and paginate. Changing the site selection requires an explicit search.

The page URL restores search progress and results. Refreshing or returning reads the existing search; closing the panel does not cancel it. Cancelling stops unfinished work and retains existing results. Results expire after 30 minutes. A service restart marks unfinished searches as interrupted; start a new search to continue.

Select fast or file rules and click **Apply preview** when needed. File previews may acquire torrents, within the configured inspection budget. Rules run over the complete candidate set before paging. Reloading the page does not automatically inspect files again.

Choose **Create task**, review or enter the release code, then confirm. Moji saves the selected source and lets the task acquire and validate the resource. Acquisition failure does not silently select another resource. Missing sizes or seeder counts remain unknown. Detail links use only safe detail URLs supplied by the site.

Choose **Get download link** to acquire a resource without creating a task. Once ready, use **Download torrent** or the magnet link. Prepared download entries remain available for five minutes. Request a new one when needed. This action may consume a site download quota.

## Upcoming releases

Below Search, upcoming releases found for subscribed performers are ordered by expected date. **View all** provides code/title search, performer filters and 7/30-day windows. Filters are preserved in the URL.

Unreleased films do not create download tasks or search indexers. On the release date, Moji checks current metadata and policy before downloading, requesting review or skipping the release. Changed dates reschedule checks; unavailable sources retain their last confirmed information. Records without any currently subscribed performer are hidden and paused.

Open a film for a player-style detail view, source link and refresh action. Trailers are not connected yet. Updating before release does not create a task; updating on or after release can start processing according to policy. Manual updates remain available when automatic checks are disabled. This list contains releases found by subscription checks, rather than a guaranteed complete catalog. Run a performer check if necessary.

Dates follow the Moji runtime system timezone by default, including a container’s TZ. Override it under **Settings > System > General** or with YAML `system.release_timezone`. An empty value or `auto` follows the system. Browser timezones do not change eligibility. Indexer results and code-only inputs do not provide a trusted release date and continue to use the resources explicitly submitted by the user.
