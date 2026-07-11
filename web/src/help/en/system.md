# System

System settings control destructive task deletion and Moji's upstream image cache. Review the active deletion policy before removing a task.

## Task deletion

The policy can delete only the Moji task record, also remove the corresponding qBittorrent item, or remove both the torrent item and its downloaded files. It is evaluated when a task is deleted and does not apply retroactively. The delete confirmation shows the current effect.

## Image cache and diagnostics

Upstream images are always proxied by Moji. Disabling disk caching prevents new cache writes, but existing cached files can still be read. Capacity accepts 64–20480 MB and retention accepts 1–365 days. Cleanup removes old entries and trims an oversized cache. Clearing the cache deletes local image files but preserves source registrations so images can be fetched again.

Logs can be filtered by level, refreshed, copied, or downloaded. Use them after checking the service status and task error. About shows the running Moji version.
