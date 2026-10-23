import { readFileSync, readdirSync } from "node:fs";
import { buildSchema, graphqlSync } from "graphql";
import { expect, test as base } from "@playwright/test";

const schemaDir = new URL("../../../graphql/moji/", import.meta.url);
const typesDir = new URL("types/", schemaDir);
const schema = buildSchema([
  readFileSync(new URL("schema.graphql", schemaDir), "utf8"),
  ...readdirSync(typesDir).filter((name) => name.endsWith(".graphql"))
    .sort().map((name) => readFileSync(new URL(name, typesDir), "utf8")),
].join("\n"));

const now = "2026-01-15T12:00:00Z";
// Execute the real queries against the schema so missing non-null fields fail
// the test instead of silently producing an error page or a loading skeleton.
const rootValue = {
  version: "0.0.0-visual",
  tasks: [],
  dashboardStats: { total: 0, active: 0, completed: 0, downloading: 0, pendingScans: 0, failed: 0 },
  settings: {
    stash: { configured: true, url: "http://stash.test", apiKeyConfigured: false, apiKey: "" },
    jackett: {
      configured: true, url: "http://jackett.test", apiKeyConfigured: false,
      apiKey: "", passwordConfigured: false, password: "",
    },
    qbittorrent: {
      configured: true, url: "http://qbittorrent.test", username: "moji",
      usernameConfigured: true, passwordConfigured: false, password: "",
      defaultSavePath: "/downloads", category: "moji", tags: "",
    },
    ingest: {
      generateTranscodes: false,
      keepSubtitles: false,
      downloads: { qbRoot: "/downloads", mojiRoot: "/downloads" },
      library: { mojiRoot: "/library", stashRoot: "/library" },
      transfer: { action: "COPY" },
    },
    automation: {
      taskProcessingEnabled: true,
      taskProgressSyncIntervalSeconds: 60, subscriptionPollIntervalHours: 6,
      sourcingRetryIntervalHours: 6, stashBoxEndpoints: [],
      subscriptionReleasePolicy: {
        soloBehavior: "DOWNLOAD", groupBehavior: "REVIEW", compilationBehavior: "BLOCK",
        maxGroupPerformerCount: 5, releaseDateRange: "ALL",
      },
      torrentSelection: { enabled: false, inspectionCandidateLimit: 5, fastRules: [], torrentRules: [] },
    },
    system: {
      taskDeletePolicy: "KEEP_ONLY",
      imageCache: { enabled: true, maxSizeMb: 1024, retentionDays: 30 },
      stashBoxDataCache: { ttlHours: 24 },
    },
  },
  settingsStatus: {
    stash: { configured: true, ready: true },
    jackett: { configured: true, ready: true },
    qbittorrent: { configured: true, ready: true },
    ingest: { configured: true },
    automation: {
      taskProgressSyncIntervalSeconds: 60, taskProgressSyncEnabled: false,
      subscriptionPollIntervalHours: 6, subscriptionPollEnabled: false,
    },
    stashBox: { stashBoxes: [], stashBoxesLoaded: true, stashBoxesLoadError: null },
    stashLibraries: [{ path: "/library" }],
    stashLibrariesLoadError: null,
    stashStats: { version: "0.28.0", sceneCount: 0, pendingMojiScanCount: 0, lastError: null, okAt: now },
    jackettStats: {
      indexerCount: 2, configuredIndexerCount: 2, lastIndexerLatencyMs: 0,
      lastIndexerError: null, lastIndexerSearchAt: null, lastError: null, okAt: now,
    },
    qbittorrentStats: {
      downloadSpeed: 0, uploadSpeed: 0, activeTorrentCount: 0, connectionStatus: "connected",
      altSpeedLimitEnabled: false, lastError: null, okAt: now,
    },
    imageCache: {
      usedBytes: 0, entryCount: 0, cacheDirectory: "/data/images", lastCleanupAt: null, lastError: null,
    },
    stashBoxDataCache: {
      usedBytes: 0, sceneCount: 0, performerCount: 0, snapshotCount: 0,
      databasePath: "/data/stashbox.db", lastCleanupAt: null, lastError: null,
    },
  },
  performerWorkspace: {
    performers: {
      items: [], page: 1, pageSize: 24, totalCount: 0, totalPages: 1,
      hasPrevPage: false, hasNextPage: false,
    },
    subscribedPerformers: [],
  },
};

export const test = base.extend({
  page: async ({ page }, use) => {
    const errors: string[] = [];
    page.on("pageerror", (error) => errors.push(error.message));
    page.on("console", (message) => {
      if (message.type() === "error") errors.push(message.text());
    });
    await page.clock.setFixedTime(new Date(now));
    await page.route("**/graphql", async (route) => {
      const request = route.request().postDataJSON();
      const result = graphqlSync({
        schema, rootValue, source: request.query,
        variableValues: request.variables, operationName: request.operationName,
      });
      errors.push(...(result.errors ?? []).map((error) => error.message));
      await route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify(result) });
    });
    // Keep GraphQL subscriptions connected without a live backend or retry loop.
    await page.routeWebSocket("**/graphql", (socket) => {
      socket.onMessage((data) => {
        const message = JSON.parse(String(data));
        if (message.type === "connection_init") socket.send(JSON.stringify({ type: "connection_ack" }));
        if (message.type === "ping") socket.send(JSON.stringify({ type: "pong" }));
      });
    });
    await use(page);
    expect(errors, "Page and GraphQL fixture errors must not become screenshot baselines").toEqual([]);
  },
});
