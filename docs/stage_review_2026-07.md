# Moji Stage Review (2026-07)

## Purpose

This document captures the current-stage product and engineering assessment of Moji after reviewing:

- current backend entrypoints, GraphQL schema, resolvers, and services;
- current web UI structure and user flows;
- task / subscription persistence and runtime state design;
- the clarified product constraints confirmed by the project owner.

This is not a generic code-style review. The goal is to decide:

- what Moji currently is;
- where it is already strong;
- what is incomplete or risky;
- what should be built next.

## Product Constraints Confirmed

The following decisions are treated as product truths for this stage:

- Moji defaults to running in a trusted internal network.
- A valid code / 番号 is a hard product rule, not an implementation accident.
- On the performer page, selecting scenes is expected to lead to batch download.
- Subscription is primarily for automatic download.
- Manual review is only a fallback for cases Moji cannot judge reliably.
- Stash may contain hundreds, thousands, or tens of thousands of performers and scenes.
- Extreme cases may require Moji to handle tens of thousands of concurrent tasks.
- Task deletion does not currently require audit trail or recycle-bin behavior.
- Subtitle generation remains a future feature, not a current-stage priority.
- Subscription truth should continue to live in Stash `custom_fields`.
- Logs and task events should evolve toward self-service troubleshooting and bug reporting.

## Overall Judgment

Moji is currently in the "mid-stage, integrated prototype moving toward internal-use readiness" phase.

It is already beyond a backend-only spike:

- the core runtime exists;
- GraphQL contracts exist;
- the React web UI is already structured around the real product surface;
- SQLite-backed task and subscription persistence are in place;
- automated tests pass;
- the frontend production build passes.

The strongest current advantage is that Moji already has a real end-to-end automation skeleton:

- discover content;
- create tasks;
- download through qBittorrent;
- map / transfer into the Stash-visible namespace;
- trigger scan;
- keep subscription state tied to Stash.

The biggest current risk is not "it does not run". The real risk is that product expectations are now clearer and larger than several current implementation seams:

- performer-driven batch download is not closed yet;
- large-scale performer and subscription flows still rely on in-memory patterns;
- task and troubleshooting data exist technically, but are not yet fully productized for user self-service.

## What Moji Is And Is Not

Moji is a Stash companion automation service.

It is not trying to replace Stash as the user's long-term home. The intended product model is:

- users configure Moji;
- users subscribe to performers and rely on automated discovery / download / ingest;
- users spend most of their time in Stash, not in Moji.

This has two direct implications:

1. Moji should optimize for background reliability and low operator burden more than for deep day-to-day browsing features.
2. Moji pages should primarily exist to configure, monitor, troubleshoot, and intervene in automation flows.

## Current Strengths

### 1. Core automation chain already exists

The backend runtime wires together:

- Jackett search;
- qBittorrent submission and progress sync;
- Stash scan integration;
- subscription polling;
- SQLite-backed task persistence.

Relevant files:

- [cmd/moji/main.go](/home/wangqi/services/moji/cmd/moji/main.go:127)
- [internal/taskruntime/service.go](/home/wangqi/services/moji/internal/taskruntime/service.go:508)
- [internal/taskruntime/stash_scan.go](/home/wangqi/services/moji/internal/taskruntime/stash_scan.go:29)
- [internal/subscription/service.go](/home/wangqi/services/moji/internal/subscription/service.go:85)

### 2. The product surface is already synchronized across backend and web UI

Moji is no longer "API first, UI later" or "UI mock, backend later". The current system already has:

- GraphQL schema types;
- resolvers;
- frontend queries and mutations;
- concrete pages for home, discovery, tasks, subscription, and settings.

Relevant files:

- [graphql/moji/types/task.graphql](/home/wangqi/services/moji/graphql/moji/types/task.graphql:1)
- [graphql/moji/types/subscription.graphql](/home/wangqi/services/moji/graphql/moji/types/subscription.graphql:1)
- [web/src/App.tsx](/home/wangqi/services/moji/web/src/App.tsx:1)

### 3. Runtime configuration is already designed for live edits

The config store and connection providers already support live runtime reconfiguration for Stash, Jackett, and qBittorrent without requiring restarts in many cases.

Relevant files:

- [internal/config/store.go](/home/wangqi/services/moji/internal/config/store.go:27)
- [internal/tracker/jackett_service.go](/home/wangqi/services/moji/internal/tracker/jackett_service.go:11)
- [pkg/qbittorrent/client.go](/home/wangqi/services/moji/pkg/qbittorrent/client.go:18)

This is a meaningful long-term advantage because Moji is fundamentally an operator-facing automation tool.

### 4. Persistence has already moved beyond toy storage

The project has already switched from temporary JSON thinking to SQLite-backed storage for tasks and subscriptions.

Relevant files:

- [internal/taskruntime/sqlite_store.go](/home/wangqi/services/moji/internal/taskruntime/sqlite_store.go:13)
- [internal/subscription/sqlite_store.go](/home/wangqi/services/moji/internal/subscription/sqlite_store.go:15)
- [docs/task_storage_sqlite_design.md](/home/wangqi/services/moji/docs/task_storage_sqlite_design.md:1)

This is the right direction for:

- task volume;
- failure history;
- future pagination;
- user-facing troubleshooting.

## Core Pain Points

### 1. Performer-page batch download is an explicit product gap

#### Problem

The performer detail page already supports per-scene selection, "select current page", and "clear selection", but there is no action that turns the selection into batch downloads.

Relevant files:

- [web/src/pages/SubscriptionPage.tsx](/home/wangqi/services/moji/web/src/pages/SubscriptionPage.tsx:337)
- [web/src/App.tsx](/home/wangqi/services/moji/web/src/App.tsx:538)

#### Impact

- The UI suggests a batch workflow that does not exist.
- The performer page cannot yet serve the product expectation confirmed by the owner.
- Users must route through discovery or manual workarounds instead of acting from the performer workflow directly.

#### Root Cause

The frontend has already introduced scene-selection state, but backend and UI did not yet add the corresponding batch queue contract and mutation flow.

#### Risk

High.

#### Priority

P0.

### 2. Subscription is correctly biased toward auto-download, but the fallback path is still under-explained

#### Problem

The release policy system already classifies releases and assigns decisions such as downloaded, queued, or blocked, but the current product surface still treats the fallback path more like state than like a user-facing explanation and optimization target.

Relevant files:

- [internal/subscription/release_policy.go](/home/wangqi/services/moji/internal/subscription/release_policy.go:1)
- [internal/subscription/service.go](/home/wangqi/services/moji/internal/subscription/service.go:362)
- [web/src/pages/SubscriptionPage.tsx](/home/wangqi/services/moji/web/src/pages/SubscriptionPage.tsx:24)

#### Impact

- Users can see that Moji chose review / block / queue, but the product does not yet strongly guide them on how to reduce ambiguous cases.
- The system has rules, but not yet enough product instrumentation around rule misses and fallback causes.

#### Root Cause

The decision engine exists, but the user-facing loop for "why did this fall back" and "how do we reduce that class of fallbacks" is still shallow.

#### Risk

High.

#### Priority

P1.

### 3. Large-scale performer and subscription flows do not yet match the target scale

#### Problem

The current performer list flow loads all Stash performers, filters in memory, sorts in memory, and then paginates in memory.

Relevant files:

- [internal/subscription/service.go](/home/wangqi/services/moji/internal/subscription/service.go:107)
- [internal/graphqlapi/subscription.resolvers.go](/home/wangqi/services/moji/internal/graphqlapi/subscription.resolvers.go:71)

`RefreshAll` also iterates subscribed performers sequentially.

Relevant file:

- [internal/subscription/service.go](/home/wangqi/services/moji/internal/subscription/service.go:341)

#### Impact

- This will degrade sharply as Stash libraries grow.
- It creates avoidable memory, latency, and polling pressure.
- It will become a practical bottleneck well before the stated target scale.

#### Root Cause

The current implementation is still optimized for correctness and local simplicity rather than large-library query efficiency.

#### Risk

High.

#### Priority

P1.

### 4. Task self-service troubleshooting is structurally possible but not yet surfaced enough

#### Problem

Moji already records task state transitions and stores enough task snapshot data to support meaningful troubleshooting, but the user-facing model is still mostly "current status plus logs".

Relevant files:

- [internal/taskruntime/sqlite_store.go](/home/wangqi/services/moji/internal/taskruntime/sqlite_store.go:21)
- [internal/taskruntime/task_stage.go](/home/wangqi/services/moji/internal/taskruntime/task_stage.go:1)
- [graphql/moji/types/task.graphql](/home/wangqi/services/moji/graphql/moji/types/task.graphql:1)

#### Impact

- Users cannot yet fully self-diagnose why a task blocked, retried, or changed stage.
- Bug reports will rely too much on raw logs instead of structured task history.
- The system is more debuggable internally than externally.

#### Root Cause

The storage and runtime model were built first; the productized read model for timelines and failure narratives has not yet caught up.

#### Risk

Medium-high.

#### Priority

P1-P2.

### 5. The web data model is heavier than necessary for page-by-page usage

#### Problem

The dashboard hook issues a large query that pulls health, stats, settings, status, and tasks in one shot.

Relevant files:

- [web/src/hooks/useDashboard.ts](/home/wangqi/services/moji/web/src/hooks/useDashboard.ts:27)
- [web/src/graphql/dashboard.graphql](/home/wangqi/services/moji/web/src/graphql/dashboard.graphql:1)

#### Impact

- Pages share a broad data dependency surface.
- Performance tuning becomes harder later.
- Sensitive config data is requested more often than strictly necessary.

#### Root Cause

The current UI favors convenience and coherence over page-specific read models.

#### Risk

Medium.

#### Priority

P2.

### 6. Some product-facing surfaces are still ahead of their actual functional closure

#### Problem

The discovery page still contains a recommendation placeholder, and the README still advertises subtitle generation even though it is not yet part of the current product baseline.

Relevant files:

- [web/src/pages/DiscoveryPage.tsx](/home/wangqi/services/moji/web/src/pages/DiscoveryPage.tsx:124)
- [README.md](/home/wangqi/services/moji/README.md:1)

#### Impact

- Users may overestimate what is production-ready.
- Development attention risks drifting toward future-facing surfaces before current loops are complete.

#### Root Cause

The product surface has advanced quickly, but some roadmap-facing messaging has not yet been tightened around current-stage priorities.

#### Risk

Medium.

#### Priority

P2.

## Product And User Experience Assessment

## Core User Flows

### Flow 1: Configure Moji and leave it running

This is the most important long-term product loop.

Current status:

- reasonably strong configuration UX;
- service status cards exist;
- settings persistence exists;
- runtime health snapshots exist.

Relevant files:

- [web/src/pages/HomePage.tsx](/home/wangqi/services/moji/web/src/pages/HomePage.tsx:1)
- [web/src/components/settings/SettingsPanel.tsx](/home/wangqi/services/moji/web/src/components/settings/SettingsPanel.tsx:1)
- [internal/stats/collector.go](/home/wangqi/services/moji/internal/stats/collector.go:1)

Main gap:

- when something breaks, the product still leans more on operator inference than on guided resolution.

### Flow 2: Discover content and queue it

Current status:

- StashBox discovery exists;
- Jackett fallback search exists;
- queueing discovered scenes exists.

Relevant files:

- [graphql/moji/types/search.graphql](/home/wangqi/services/moji/graphql/moji/types/search.graphql:1)
- [internal/graphqlapi/search.resolvers.go](/home/wangqi/services/moji/internal/graphqlapi/search.resolvers.go:14)

Main gap:

- the discovery page still mixes real workflow with placeholder future surface;
- the product should increasingly treat discovery as a supporting entrypoint, not the central one.

### Flow 3: Subscribe to performers and trust auto-download

This is the defining product loop.

Current status:

- subscribe / unsubscribe exists;
- release polling exists;
- release classification exists;
- auto-download can create tasks;
- Stash remains source of truth for subscription state.

Relevant files:

- [internal/subscription/service.go](/home/wangqi/services/moji/internal/subscription/service.go:132)
- [internal/subscription/release_policy.go](/home/wangqi/services/moji/internal/subscription/release_policy.go:1)

Main gaps:

- scale bottlenecks in performer and refresh flows;
- fallback explanations are not yet productized enough;
- performer scene selection does not yet result in batch queueing.

### Flow 4: Handle failures without leaving the product

Current status:

- task stages are modeled well;
- retry exists;
- scan triggering exists;
- logs can be surfaced.

Relevant files:

- [internal/taskruntime/task_stage.go](/home/wangqi/services/moji/internal/taskruntime/task_stage.go:1)
- [internal/graphqlapi/task.resolvers.go](/home/wangqi/services/moji/internal/graphqlapi/task.resolvers.go:11)

Main gap:

- the product still needs a stronger task-event / explanation layer so users can self-diagnose instead of reading raw logs.

## Business Logic Assessment

### What is solid

- `番号/code` is consistently treated as a hard identity gate for task creation.
- task stages are explicit and map well to user-understandable automation phases:
  - sourcing;
  - downloading;
  - pending ingest;
  - transferring;
  - scanning;
  - completed.
- task dedupe is already expressed across code and torrent identity.
- subscription state ownership is properly split:
  - Stash `custom_fields` for subscribed / not subscribed;
  - Moji-local store for runtime state and release processing.

Relevant files:

- [internal/taskruntime/task_identity.go](/home/wangqi/services/moji/internal/taskruntime/task_identity.go:107)
- [internal/taskruntime/task_stage.go](/home/wangqi/services/moji/internal/taskruntime/task_stage.go:5)
- [internal/subscription/sqlite_store.go](/home/wangqi/services/moji/internal/subscription/sqlite_store.go:24)

### What is still weak

- performer-scene selection has no business operation behind it yet, so part of the subscription UI has no real behavior.
- automatic download decisions are rule-driven, but there is not yet enough feedback structure around decision misses.
- some list flows are only locally correct because they assume moderate volumes, not the target volumes now stated by the owner.

## Architecture Assessment

### What is working well

- module boundaries are mostly reasonable:
  - `taskruntime` owns tasks and ingest flow;
  - `subscription` owns performer/release logic;
  - `stashsync` owns Stash integration semantics;
  - `graphqlapi` maps runtime services to product contracts.

- dynamic config provider patterns are strong and should be preserved.

- SQLite-backed persistence is the correct base for the next stage.

### What is likely to bottleneck next

- in-memory performer querying and pagination;
- sequential subscription refresh at scale;
- oversized frontend read models;
- lack of productized structured event history for large task volumes.

### Overdesign vs underdesign

The project is not suffering from heavy overdesign.

The current issue is more often underdeveloped read models and user operations on top of a decent domain core:

- the backend model often knows more than the UI exposes;
- the UI sometimes suggests more capability than the backend currently offers.

## Maintainability Assessment

### Strengths

- naming is mostly coherent around product concepts.
- task and subscription state are represented with recognizable domain language.
- test coverage is present across key backend packages.
- the system is still understandable without deep framework magic.

### Weaknesses

- some user-facing flows span many files and layers before becoming visibly closed.
- the current frontend root state in `App.tsx` is becoming large and central.
- new developers may struggle to see which product flows are truly complete versus partly staged.

Relevant files:

- [web/src/App.tsx](/home/wangqi/services/moji/web/src/App.tsx:1)
- [internal/graphqlapi/resolver.go](/home/wangqi/services/moji/internal/graphqlapi/resolver.go:1)

## Recommended Next-Stage Plan

### Immediate Fixes

1. Add performer-page batch download as a first-class flow.
2. Remove or downgrade misleading placeholder surfaces where the action does not yet exist.
3. Tighten README wording so future features are clearly marked as future-stage capabilities.
4. Define the product contract for "manual review fallback" so every fallback reason is explainable and reducible.

### Short-Term Optimization

1. Add batch queue GraphQL contract for selected performer scenes.
2. Reuse the standard task pipeline for performer-page batch downloads instead of introducing a separate task type.
3. Add structured task timeline / event read model for the task detail UI.
4. Add user-facing troubleshooting copy:
   - why a task blocked;
   - what the system tried;
   - what the user should fix next.
5. Split oversized page queries into narrower read models where it improves clarity or performance.

### Mid-Term Refactor

1. Replace all-performer in-memory pagination with scalable query and paging behavior.
2. Redesign subscription refresh for controlled concurrency, progress visibility, and backpressure.
3. Add scalable task listing / filtering / paging paths for high-volume task centers.
4. Turn fallback decision telemetry into a product loop:
   - count fallback causes;
   - surface common ambiguity classes;
   - improve matching / classification rules based on real misses.

### Explicit Non-Priorities For This Stage

1. Subtitle generation.
2. Recommendation system.
3. Recycle bin or audit trail for task deletion.
4. Deep security hardening beyond what fits the trusted-internal-network assumption.

## Detailed Design Direction: Performer Batch Download

This is the clearest missing product capability and should be implemented as the next major feature.

### Product Goal

From a performer detail page, the operator should be able to:

- filter scenes;
- select one or more scenes;
- queue them into the same standard Moji task pipeline used elsewhere;
- see which selected items were queued, skipped, or already present.

### Recommended Backend Contract

Add a mutation shaped around batch queueing from performer scenes.

Suggested behavior:

- input accepts a list of selected scene identities;
- backend validates each item against standard Moji code and dedupe rules;
- successful items produce normal persisted tasks;
- skipped items return structured reasons such as:
  - already in library;
  - duplicate task;
  - missing code;
  - missing StashBox scene mapping.

The result should not be a bare list of tasks. It should return a batch outcome summary fit for UI feedback.

### Recommended Domain Behavior

- do not invent a second download pipeline for performer batch actions;
- map performer-scene queueing onto the existing task creation path;
- preserve strict `code` requirements;
- preserve existing dedupe guarantees.

### Recommended UI Behavior

- keep the existing selection UI;
- add a primary action such as "批量下载所选";
- show per-batch summary:
  - queued count;
  - skipped count;
  - reason breakdown;
- allow quick jump to created tasks.

## Detailed Design Direction: Auto-Download First, Review As Fallback

Moji should treat manual review as an exception path that should shrink over time.

### What the system already has

- release classification;
- configurable behaviors for solo / group / compilation-like releases;
- date-range fallback;
- explicit decision reasons.

Relevant file:

- [internal/subscription/release_policy.go](/home/wangqi/services/moji/internal/subscription/release_policy.go:1)

### What the system still needs

- UI grouping by fallback reason;
- aggregate counters for decision reasons;
- operator visibility into "which fallback classes are most common";
- targeted optimization against real fallback clusters.

Without this, the system can make fallback decisions, but it cannot learn productively from them.

## Detailed Design Direction: Self-Service Troubleshooting

The long-term troubleshooting target should be:

- users open a task;
- users see the stage path and failure reason;
- users understand whether the issue is:
  - config;
  - upstream search;
  - duplicate;
  - qBittorrent submission;
  - path mapping;
  - transfer;
  - Stash scan;
- users know what to change next.

### Current foundation

- task stage model exists;
- task event persistence direction exists;
- logs exist;
- stage-specific error codes already exist.

Relevant files:

- [internal/taskruntime/task_stage.go](/home/wangqi/services/moji/internal/taskruntime/task_stage.go:18)
- [internal/taskruntime/sqlite_store.go](/home/wangqi/services/moji/internal/taskruntime/sqlite_store.go:21)

### Next product step

Expose a structured task timeline and pair each blocked state with:

- operator-facing explanation;
- suggested fix;
- retry affordance where valid.

## Validation Notes

The assessment above was checked against the current project state.

Verification performed:

- backend tests passed via `env GOCACHE=/tmp/moji-gocache go test ./...`
- frontend production build passed via `npm run build`

Observed note:

- the current web build emits a large chunk warning around the main JS bundle, which is not a blocker now but will matter once the UI grows further.

## Final Conclusion

Moji is already a real automation product skeleton, not a toy integration demo.

The next stage should not focus on adding more future-facing features. It should focus on making the core identity clearer and stronger:

- performer-driven batch download;
- subscription-driven automatic download with fewer fallbacks;
- scalability for large Stash libraries;
- task and failure visibility for self-service troubleshooting.

If those four directions are executed well, Moji will move from "can demonstrate the workflow" to "can be trusted to sit beside Stash and quietly do its job".
