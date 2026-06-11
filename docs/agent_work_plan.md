# Moji Agent Work Plan

## Project Positioning

- Moji is a self-hosted companion service for Stash, not a replacement for Stash.
- The current product direction is to let Moji own discovery, download orchestration, task tracking, and selected automation around Stash scans.
- Development should keep the system on a compilable, testable, no-secrets baseline before expanding broader automation loops.

## Historical Context

- The main planning thread started from Codex session `019e93aa-39bf-74c3-9de1-4875088f2b16` on 2026-06-05.
- That thread established a clear direction:
  - first restore a stable baseline;
  - then converge on Moji's own GraphQL API instead of continuing to grow ad hoc REST debug endpoints;
  - advance Web UI and backend together;
  - implement the minimum closed loop for search, download, persisted task tracking, progress sync, and Stash scan triggering;
  - expand following, settings, and richer product surfaces after the core loop is solid.
- A later session on 2026-06-08 attempted to split `graphql/moji` into multiple domain files.
- That schema split is now restored in the current worktree and committed as part of the active GraphQL cleanup line.

## Current Repository Status

### What Is Already Working

- HTTP service entrypoint is in place, with:
  - GraphQL POST endpoint at `/graphql`;
  - GraphQL Playground at `/playground`;
  - existing REST-style API routes kept for compatibility/debug;
  - Web UI routing with a clear fallback when the frontend is not built.
- Moji GraphQL already exposes the main vertical slice for the download workflow:
  - service health and version;
  - Jackett torrent search;
  - qBittorrent torrent listing;
  - persisted task lookup and task listing;
  - Stash job lookup;
  - mutations for `addTorrent`, `downloadMedia`, `syncTaskProgress`, `triggerStashScans`, and `stashMetadataScan`.
- Resolver wiring is in place and follows dependency injection through `internal/graphqlapi`.
- qBittorrent is no longer a full process-level fail-fast requirement:
  - if qBittorrent config is incomplete, related resolver capabilities are disabled instead of crashing startup for that case.
- A downloader service exists with:
  - search-to-candidate selection;
  - torrent submission;
  - persisted task creation;
  - task lookup/list;
  - progress synchronization from qBittorrent;
  - completed-task handoff into Stash scan triggering.
- JSON task storage is implemented and safe enough for the current baseline:
  - load existing tasks;
  - persist sorted tasks;
  - replace writes through a temp file + rename flow.
- Stash scan integration is abstracted behind `internal/stashsync`.
- The Web UI is no longer just a placeholder:
  - dashboard, search, and task-related views are connected to the GraphQL API;
  - the UI already reflects task progress and scan state at a basic level.
- The Moji GraphQL schema is now split by domain under `graphql/moji/types`, with a small root `schema.graphql` entry file.
- Resolver code is now split by domain as well:
  - `health.resolvers.go`;
  - `search.resolvers.go`;
  - `task.resolvers.go`;
  - `stash.resolvers.go`.
- The Settings surface now has a real backend-driven runtime snapshot:
  - Stash, Jackett, qBittorrent, Tasks, and System tabs read live GraphQL data;
  - sensitive values remain masked as configuration-presence booleans instead of raw secrets;
  - tabs that do not yet have backend support are rendered as disabled in the Web UI.

### What Is Verified

- `go test ./cmd/moji ./internal/graphqlapi ./internal/downloader ./internal/tracker` passes.
- `go test ./internal/stashsync` passes.
- GraphQL disabled-dependency behavior now has explicit coverage for qBittorrent and Stash-backed resolver paths.
- Command-layer HTTP/GraphQL regression coverage includes playground routing and disabled dependency behavior.
- `npm --prefix web run codegen` passes.
- `npm --prefix web run build` passes.
- The git worktree was clean at the time this status was recorded.

### What Is Still Incomplete

- The Web UI has clear placeholder or proto areas, especially around:
  - following;
  - parts of settings/help;
  - richer dashboard/stats surfaces.
- The current docs do not yet define the data contract for following, nor the long-term configuration UX.
- Old REST debug routes still exist and should not become the main integration path again.

## Working Rules For Future Changes

- Keep frontend and backend in the same pass when they depend on each other.
- When a UI change needs new data, add the GraphQL/backend contract first or alongside the UI.
- When a backend capability creates a new user-facing action, add the UI surface and tests in the same pass whenever practical.
- Prefer small, verifiable increments that preserve a clean baseline.
- Do not hard-code credentials. Keep secrets out of committed source and rotate old test credentials if they ever existed.
- Preserve Moji's role as a companion to Stash:
  - Moji should orchestrate and assist;
  - Stash remains the source of truth for library-side metadata and scanning workflows.

## Development Plan

### P0: Close The Current Core Loop

- (closed) Reorganize the Moji GraphQL schema by domain:
  - `health`;
  - `search`;
  - `task`;
  - `settings`;
  - `following`;
  - `stats`;
  - `common`.
- (closed) Split resolver code to match those domains so schema structure, resolver ownership, and UI areas align.
- (closed) Add tests for the remaining unverified service seams, especially:
  - `internal/stashsync`;
  - GraphQL behavior around disabled dependencies;
  - command-layer HTTP/GraphQL integration regressions.
- (closed) Normalize code generation flow in `Makefile` so GraphQL and web codegen are easy to run and reason about.
- Keep REST search routes explicitly secondary and avoid expanding them with new product behavior.

### P1: Make State And Configuration Visible

- (closed) Add GraphQL queries for runtime capability/configuration state, including:
  - whether Jackett is configured;
  - whether qBittorrent is configured/enabled;
  - whether Stash is configured/enabled;
  - task store type and path-related defaults;
  - downloader defaults such as save path, category, and tags.
- (closed) Replace static or proto Settings panels with real backend-driven state.
- (closed) Add editable settings flows for Stash, Jackett, and qBittorrent, with backend persistence and masked secret handling.
- Expand the task center so it becomes the operational control surface:
  - filtering;
  - (closed) grouped statuses;
  - (partial) clearer failure reasons;
  - (closed) manual sync actions;
  - (partial) manual Stash scan triggers where appropriate.
- Add dashboard aggregates that summarize the system rather than only listing raw tasks.

### P2: Build The Following Data Model

- Define what a "following" item is in Moji:
  - performer;
  - series;
  - studio/label;
  - or a mixed model.
- Design and implement the GraphQL contract for following before polishing the UI.
- Deliver a minimum following loop:
  - create/update/delete followed items;
  - list followed items;
  - show recent updates;
  - allow manual refresh/check.
- Only after the contract is stable, decide how much richer metadata should come from `r18dev`, `stashbox`, or other sources.

### P3: Expand Automation Carefully

- Revisit automatic download rules and filters after following exists.
- Add deduplication and safer selection logic for candidate choice.
- Define clearer scan-complete and failure-recovery behavior across download and Stash workflows.
- Treat background automation as an extension of the verified task model, not as a separate hidden system.

## Recommended Immediate Next Step

- With settings/runtime capability state and editable settings now `(closed)`, the next step is to finish the task center operational slice.
- Recommended P1 close-out order:
  - normalize and surface clearer failure reasons for download and Stash scan failures;
  - add task-scoped Stash scan actions where a global trigger is too coarse;
  - move dashboard aggregates onto a clearer product contract instead of relying only on frontend-derived list summaries.
- After that, move into following contract design instead of extending placeholder UI first.
