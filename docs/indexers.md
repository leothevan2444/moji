# Indexer setup and recovery

Moji can search through built-in indexers, an external Jackett installation, or
both. Configure them in **Settings → Indexers**. qBittorrent performs downloads;
Stash handles the media library and ingest.

## Add built-in sites

1. Open **Definition library** and choose **Download definitions**. Moji downloads
   the official Prowlarr Cardigann v11 library and validates it before activation.
2. Open **Sites → Add site** and find a definition. A public definition not yet
   added and without a login flow offers **Add and enable**, using its defaults.
   Use the gear button to configure a mirror, network profile or site settings
   before adding it.
3. Adding saves a configuration; the site's switch controls whether searches
   use it. Test the configuration when needed. Authentication is tested by
   default; search and resource acquisition are separate choices in the test.
4. Arrange the site order. Moji searches in that order and applies its existing
   resource-selection rules before creating or continuing a download task.

Already-added definitions remain visible with their status. **Add another
configuration** supports a separate account or mirror. The site type appears as
a colored label: public green, private red and semi-private yellow.

**Log in to site** appears only when the rule declares a login flow. It lets
Moji submit the configured credentials and handle a declared challenge before
searching; it does not open the website or download a resource. The resulting
session is reused for searches and resource acquisition. A public rule can
still declare a login flow, so availability is based on the rule itself.

The application starts without a definition library or enabled sites. Reading
Home or opening settings does not install definitions. Sites are not enabled
automatically after installation. A definition's presence means Moji can load
its rule; the site's availability, credentials and challenges still determine
whether a search succeeds.

The library contains declarative definitions. Sites that require a dedicated
Jackett/Prowlarr parser are not provided by the built-in engine. Configure an
external Jackett source for those sites if needed.

When a definition supplies a hash and title for generating a magnet, Moji adds
Jackett's public tracker defaults. Magnets supplied directly by a site keep their
original parameters. A hash-only result from a private site does not acquire a
public magnet fallback.

## Update definitions independently

Definitions are downloaded at runtime and stored with Moji's persistent data;
they are not bundled into the source or image. A library update does not require
rebuilding Moji. Changes that require new interpreter capabilities still require
a compatible Moji version.

For an installed library, the default policy checks for updates every 24 hours
and applies them manually. **Check for updates** discovers a version;
**Update definitions** downloads and validates it. Automatic application is a
separate option in the library settings. First installation always requires an
explicit download action.

Progress and the final result survive page refresh. A failed or cancelled update
keeps the current working library. Updates also check compatibility with enabled
site configurations before activation. They do not reselect resources that have
already been accepted for a task.

**Roll back definitions** selects the previous locally installed version and
checks it before activation. **Repair definitions** restores missing or damaged
files for the selected version. These operations concern the rule library;
reverting Moji itself requires matching configuration and database backups.

Persist the entire Docker `/data` mount, including the database generations and
their indexer bundles. See [database backups and recovery](database-operations.md)
before changing an existing installation. Do not copy new YAML files over an
active library directory.

## Use external Jackett

Open **External Jackett**, enter its service URL and API key, then enable the
connection. A Jackett dashboard password is not needed for searches or catalog
status. Configure the remote sites in Jackett itself; refresh the Jackett catalog
in Moji after making remote changes.

Enabled external sites appear alongside built-in sites and can share their
order. Home counts enabled built-in and Jackett sites together. An unavailable catalog retains the last confirmed state;
it is not treated as a confirmed empty list.

New installations leave Jackett disabled. Reading supported older configurations
preserves their existing Jackett enablement and order rather than switching
them to built-in sources. The current YAML layout is `config_version: 3`;
[the packaged example](../docker/config.example.yaml) shows the indexer sections.

## Network, login and cache

**Network & cache** manages reusable proxy/browser-verification profiles and
request budgets. Direct connections require no external service. FlareSolverr is
optional and is used for recognized authentication/search challenges; Moji does
not delegate torrent acquisition or automatically replay grabs through it.

Some sites require a manual login challenge. Use the site's login action, answer
the image or text challenge, and submit it before expiry. Reopening the login
dialog restores a still-pending challenge. A cancelled or expired login requires
a new attempt.

Site credentials are not read back in plaintext. Leave an existing secret blank
to retain it; use the explicit clear control to remove it. Advanced trusted
origins authorize the named additional origins for that site's requests and
cookies. They do not grant access to private-network destinations.

Built-in search caching defaults to five minutes for successful results and
30 seconds for confirmed empty results, with a 32 MiB limit. Failed or incomplete
searches do not become cached empty results. Jackett searches use Jackett's own
cache instead of a second Moji search cache. Resource selection still runs with
the current task rules when search results are reused.

Clearing the search cache causes future searches to fetch again. It does not
remove selected resources, existing tasks, or downloads. File inspection and task
submission reuse the acquired torrent so that they operate on the same bytes.

## When an indexer needs attention

| State | Next action |
| --- | --- |
| No definition library | Download it from **Definition library**, then add sites. |
| Library update failed | Review the job details; the previous library remains active. Retry after resolving the reported cause. |
| Installed files cannot be read | Use **Repair definitions**, or review a compatible local rollback. |
| Site has not been verified | Run its test; this status is not evidence of a failed connection. |
| Login required | Check the site's saved credentials and complete any manual challenge. |
| Rate limited | Respect the displayed cooldown; repeated searches do not bypass it. |
| Search succeeded but resource acquisition failed | Review the acquisition result separately; a working search does not verify the download endpoint. |
| Jackett unavailable | Check that optional connection and its remote sites; the last confirmed catalog remains visible. |

Home's **Indexers** section shows current enabled sites and search statistics
over a selected period (24h, 72h, or two weeks; default 72h): search count, success
rate, and average search time. The retained record count covers all saved reports.
Cache hits and joined callers do not increase search counters; a successful empty
response counts as success. The measured interval is available on the metrics.
History, statistics, and recent site status are all queried from the same execution table.
All records in the last 14 days are retained without a global row limit. Each
configured site also retains its ten latest actual executions, even when older.
Removed sites lose that extra protection. Cleanup never shortens the two-week
statistics window. Search time excludes candidate inspection and acquisition;
the average uses only records with reliable separate search timing. Missing
timing records are excluded from the mean; no samples or a zero mean displays
"-" without an additional notice.

**Manage indexers** opens settings. **View indexer records** opens the same history
available in settings, with site, outcome, operation, and time filters plus text
search. Filters apply to all retained reports before pagination. Site history
icons turn red when the latest execution failed, or yellow when an earlier one
of the last ten actual searches or fetches failed. Cache hits and cancellations
do not clear this status. Execution records directly determine both recent site
status and search statistics; there are no separate recent snapshots or counters. Current enabled
site counts do not depend on the selected search period and always show a number.
When Jackett is enabled and its connection is configured, Moji loads its directory
automatically in the background after startup and on configuration changes.
Periodic checks reuse the directory cache and failure backoff; refresh failures
retain the last confirmed directory for the same connection. An empty statistics window keeps its
requested time bounds and does not imply a zero success rate. Full errors and copyable identifiers are in history;
the list does not repeat the errors on each row.

The Jackett settings panel shows the last confirmed site-list update and catalog
errors. An old directory is independent of an individual site's search outcome.
Download progress and qBittorrent controls remain in **Downloader**.
