# Home

Home is the operational summary. It shows whether Moji can currently use its dependencies, how content will enter Stash, and which tasks need attention.

## Service cards

Each Stash, Jackett, and qBittorrent card distinguishes **configured** from **ready**. Configured means the minimum connection fields are present. Ready means a recent backend probe succeeded. A failed or stale probe makes the service unavailable and the card shows the latest error.

Stash statistics include its version, scene count, and pending Moji scans. Jackett shows configured indexers and the latest search condition. qBittorrent shows transfer rates, active torrents, connection state, and alternative speed-limit state.

## Ingest and attention items

The ingest card summarizes the active delivery mode and paths. Treat an incomplete card as a blocker: downloads may finish, but Moji cannot safely produce a Stash scan path.

Attention items contain blocked tasks and downloads awaiting a Stash scan. Open a card for exact paths, upstream state, and the available retry, resolve, or scan action. Home shows only a short list; **Tasks** is authoritative for the full history.
