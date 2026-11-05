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
function homeTask(id: string, stage: string, status = "RUNNING", extra: Record<string, unknown> = {}) {
  return { version:"00000000000000000001",revision:1,stashSceneId: (extra.stashSceneIds as string[] | undefined)?.[0] ?? null, id, code: id, source: "SUBSCRIPTION", stage, stageStatus: status, stageLabel: stage, stageStatusLabel: status,
    candidate: { title: id, tracker: "", infoHash: "", link: "", magnetUri: "", size: 0, seeders: 0, peers: 0 }, files: [], torrentUrl: "", progress: 0, stashSceneIds: [], createdAt: "2026-01-14T12:00:00Z", updatedAt: now, ...extra };
}
export const homeOverview = {
  total: 26, attention: 3, active: 8, waiting: 12, completedLast24Hours: 3,
  attentionTasks: [
    homeTask("MOJI-101", "DOWNLOADING", "BLOCKED", { stageErrorCode: "FILE_SELECTION_REQUIRED", files: [{ index: 0, path: "MOJI-101.mp4", size: 1000, kind: "UNKNOWN_VIDEO", part: 0, selected: false, reason: "IDENTITY_UNCERTAIN" }] }),
    homeTask("MOJI-102", "SOURCING", "BLOCKED", { stageErrorCode: "SOURCING_FAILED" }),
    homeTask("MOJI-103", "SCANNING", "BLOCKED", { stageErrorCode: "STASH_IDENTIFY_FAILED" }),
  ],
  activeTasks: [homeTask("MOJI-201", "DOWNLOADING", "RUNNING", { progress: .62 }), homeTask("MOJI-202", "TRANSFERRING"), homeTask("MOJI-203", "SCANNING", "RUNNING", { stashIngestPhase: "IDENTIFY" })],
  recentTasks: [homeTask("MOJI-301", "COMPLETED", "DONE", { ingestCompletedAt: "2026-01-15T11:50:00Z", stashSceneIds: ["301"] }), homeTask("MOJI-302", "COMPLETED", "DONE", { ingestCompletedAt: "2026-01-15T11:25:00Z", stashSceneIds: ["302"] })],
};
// Execute the real queries against the schema so missing non-null fields fail
// the test instead of silently producing an error page or a loading skeleton.
export const downloaderSnapshot = {
    checkedAt: now, connected: true, stale: false, storageAvailable: true, freeSpace: 230000000000,
    downloadSpeed: 12500000, uploadSpeed: 2400000, downloadLimit: 0, uploadLimit: 0,
    connectionStatus: "connected", dhtNodes: 230, completedBytes: 19440000000, remainingBytes: 4560000000,
    total: 2, attentionCount: 0, policy: { enabled: false, ratio: 2, seedHours: 72, matchAll: true, deleteFiles: true, afterIngest: false },
    torrents: [
      { hash: "a".repeat(40), name: "MOJI-301 - Completed and seeding", state: "uploading", present: true,
        taskIds: ["MOJI-301"], taskCodes: ["MOJI-301"], allCompleted: true, canDeleteFiles: true,
        progress: 1, downloadSpeed: 0, uploadSpeed: 2400000, size: 12000000000, completedBytes: 12000000000,
        downloaded: 12000000000, uploaded: 32000000000, ratio: 2.67, seedingSeconds: 90000, eta: 0, seeds: 8, peers: 12,
        availability: 3.4, savePath: "/downloads/MOJI-301", category: "moji", tags: "subscription",
        lastSeenAt: now, removedAt: "", removalReason: "", cleanupError: false, needsAttention: false },
      { hash: "b".repeat(40), name: "An independently added qB torrent", state: "downloading", present: true,
        taskIds: [], taskCodes: [], allCompleted: false, canDeleteFiles: false,
        progress: .62, downloadSpeed: 12500000, uploadSpeed: 0, size: 12000000000, completedBytes: 7440000000,
        downloaded: 7440000000, uploaded: 2000000000, ratio: .27, seedingSeconds: 0, eta: 0, seeds: 0, peers: 0,
        availability: 0, savePath: "/downloads/MOJI-201", category: "moji", tags: "subscription",
        lastSeenAt: now, removedAt: "", removalReason: "", cleanupError: false, needsAttention: false }
    ]
  };

export const indexerLibrary = {
  streamId: "visual-indexer", activeRevision: "a".repeat(40), previousRevision: "", availableRevision: "a".repeat(40), revision: 1,
  available: true, lastCheckedAt: now, checkErrorCode: "", errorCode: "", job: null,
  installed: [{ revision: "a".repeat(40), schemaVersion: 11, engineVersion: "cardigann-v11-1", definitionCount: 579, installedAt: now }]
};
export const indexerInstances = {
  healthErrorCode: "", revision: "config-1", ruleRevision: "a".repeat(40), ruleErrorCode: "", catalogErrorCode: "", jackettEnabled: true, order: ["builtin-one", "jackett/onejav"], networkProfiles: ["direct"],
  items: [
    {description:"",loginSupported:false,health:{search:null,fetch:null}, id: "builtin-one", name: "Sukebei", kind: "BUILTIN", enabled: true, definitionId: "sukebei", baseUrl: "", siteUrl: "https://sukebei.nyaa.si/", privacy: "public", language: "en-US", networkProfile: "direct", trustedOrigins: [], ruleDigest: "digest", errorCode: "", modes: ["search"], settings: [
      { name: "filter", type: "select", label: "Filter", secret: false, configured: false, value: null, defaultValue: { string: "0", boolean: null, number: null }, options: [{ value: "0", label: "All" }, { value: "1", label: "Trusted only" }] },
      { name: "cookie", type: "text", label: "Cookie", secret: true, configured: true, value: null, defaultValue: null, options: [] }
    ] },
    {description:"",loginSupported:false,health:{search:null,fetch:null}, id: "jackett/onejav", name: "OneJAV", kind: "JACKETT", enabled: true, definitionId: "", baseUrl: "", siteUrl: "https://onejav.com/", privacy: "public", language: "en-US", networkProfile: "", trustedOrigins: [], ruleDigest: "", errorCode: "", modes: ["search"], settings: [] }
  ]
};
export const searchCandidate = { id:"candidate-one",instanceId:"builtin-one",instanceName:"Sukebei",kind:"BUILTIN",title:"MOJI-102 Long resource title with release and encoding information",detailsUrl:"https://example.test/details?id=1",infoHash:null,size:null,seeders:null,peers:null,publishedAt:null,categories:[] };
export const torrentSearchSnapshot = {id:"search-one",state:"COMPLETED",revision:1,keywords:"MOJI-102",taskId:null,createdAt:now,updatedAt:now,expiresAt:"2099-01-15T12:00:00Z",finishedAt:now,cancelRequested:false,partial:false,truncated:false,errorCode:null,correlationId:"search-correlation",total:1,offset:0,limit:10,
 sites:[{instanceId:"builtin-one",name:"Sukebei",kind:"BUILTIN",state:"COMPLETED",results:1,cacheHit:true,fetchedAt:now,elapsedMs:20,partial:false,truncated:false,errorCode:null,correlationId:null,retryAt:null}],candidates:[searchCandidate]};
const rootValue = {
  testIndexer: ({input}:{input:{instanceId?:string;expectedRevision:string;search:boolean;download:boolean}})=>({id:"site-test",instanceId:input.instanceId??"draft-only",kind:input.instanceId?.startsWith("jackett/")?"JACKETT":"BUILTIN",configRevision:input.expectedRevision,ruleRevision:"a".repeat(40),startedAt:now,finishedAt:now,results:input.search?1:0,partial:false,truncated:false,stages:[{name:"CONFIGURATION",state:"PASSED",errorCode:"",elapsedMs:1},{name:"AUTHENTICATION",state:input.instanceId?.startsWith("jackett/")?"NOT_OBSERVED":"NOT_REQUIRED",errorCode:"",elapsedMs:0},{name:"SEARCH",state:input.search?"PASSED":"NOT_REQUESTED",errorCode:"",elapsedMs:2},{name:"DOWNLOAD",state:input.download?"PASSED":"NOT_REQUESTED",errorCode:"",elapsedMs:3}]}),

  indexerDefinitionPolicy: {revision:"config-one",source:"prowlarr",schemaVersion:11,checkIntervalHours:24,autoUpdate:false},
  updateIndexerDefinitionPolicy: ({input}:{input:{checkIntervalHours?:number;autoUpdate?:boolean}})=>({revision:"config-two",source:"prowlarr",schemaVersion:11,checkIntervalHours:input.checkIntervalHours??24,autoUpdate:input.autoUpdate??false}),
  prepareTorrentDownload: ()=>({kind:"TORRENT",downloadUrl:"/api/indexers/downloads/"+"d".repeat(64),expiresAt:new Date(Date.now()+300000).toISOString()}),
  torrentSearch: ({id,offset=0,limit=100}:{id:string;offset:number;limit:number})=>({...torrentSearchSnapshot,id,offset,limit,taskId:id.startsWith("task-search-")?id.slice(12):null}),
  startTorrentSearch: ({input}:{input:{keywords:string;taskId?:string}})=>({...torrentSearchSnapshot,id:input.taskId?`task-search-${input.taskId}`:"search-one",taskId:input.taskId??null,keywords:input.keywords,state:"QUEUED",total:0,candidates:[]}),
  resolveTaskSourcingCandidate: ({input}:{input:{taskId:string;expectedRevision:number}})=>homeTask(input.taskId,"SOURCING","PENDING",{revision:input.expectedRevision+1,sourcingNextCheckAt:null}),
  cancelTorrentSearch: ({id}:{id:string})=>({...torrentSearchSnapshot,id,state:"CANCELLED",cancelRequested:true}),
  previewTorrentSelection: ({input}:{input:{searchId:string;offset?:number;limit?:number;applyFastRules:boolean;applyFileRules:boolean}})=>({searchId:input.searchId,searchRevision:1,total:1,offset:input.offset??0,limit:input.limit??100,candidates:[searchCandidate],meta:{appliedFastRules:input.applyFastRules,appliedFileRules:input.applyFileRules,inspectedCount:input.applyFileRules?1:0,inspectableCount:input.applyFileRules?1:0,inspectionCandidateLimit:input.applyFileRules?5:0}}),
  queueTorrentCandidate: homeTask("queued-candidate","SOURCING","PENDING"),
  torrentSearchEvents: ({id}:{id:string})=>({streamId:"search-stream",sequence:0,searchId:id}),

  indexerNetworkSettings: {revision:"config-1",requestTimeoutSeconds:30,siteTimeoutSeconds:120,challengeTimeoutSeconds:90,profiles:[{id:"direct",proxyConfigured:false,flareSolverrConfigured:false,usedBy:1}]},
  indexerCacheStatus: {revision:"config-1",enabled:true,ttlSeconds:300,emptyTtlSeconds:30,maxSizeMb:32,entries:0,usedBytes:0,activeRequests:0,activeWaiters:0},
  clearIndexerSearchCache: {revision:"config-1",enabled:true,ttlSeconds:300,emptyTtlSeconds:30,maxSizeMb:32,entries:0,usedBytes:0,activeRequests:0,activeWaiters:0},
  indexerEvents: {streamId:"visual-indexers",sequence:0,kind:"READY"},
  indexerDefinitionStatus: indexerLibrary,
  indexerDefinitionEvents: indexerLibrary,
  checkIndexerDefinitionUpdate: indexerLibrary,
  indexerInstances,
  indexerDefinitions: { revision: "a".repeat(40), totalCount: 1, offset: 0, limit: 30, items: [{ id: "sukebei", name: "Sukebei", description: "Public torrent index", categories: ["XXX", "XXX/DVD"], language: "en-US", privacy: "public", loginSupported: false, digest: "digest", links: ["https://sukebei.nyaa.si/"], modes: ["search"], warnings: [], settings: indexerInstances.items[0].settings }] },
  setIndexerEnabled: indexerInstances,
  refreshJackettIndexers: indexerInstances,
  reorderIndexers: indexerInstances,
 upcomingReleases: () => ({items:[],performers:[],totalCount:0,page:1,pageSize:24,totalPages:0,subscribedCount:2,automatic:true,timezone:"Asia/Shanghai",today:"2026-10-31"}),
  version: "0.0.0-visual",
  testServiceConnection: {code:"OK"},
  refreshDownloader: true,
  setDownloaderSpeedLimit: true,
  controlTorrent: true,
  updateDownloaderPolicy: ({input}: {input: Record<string,unknown>}) => ({...downloaderSnapshot.policy,...input}),
  downloader: downloaderSnapshot,
  homeIndexerSummary: {definitionJob:null,definitionErrorCode:"",revision:"indexers-config-1",total:2,builtinEnabled:1,builtinUnverified:1, builtinSearchSucceeded:0,healthErrorCode:"",attention:0,issues:[],jackett:{sites:1,stale:false,updatedAt:now,errorCode:""}},
  homeOverview,
  homeSubscriptionSummary: { total: 24, attention: 1, unchecked: 0, lastCheckedAt: "2026-01-15T11:30:00Z", attentionPerformerIds: ["p1"] },
  tasks: [],
  task: ({ id }: { id: string }) => [...homeOverview.attentionTasks, ...homeOverview.activeTasks, ...homeOverview.recentTasks].find(task => task.id === id),
  taskStashScenes: [],
  discoverScenes: { items: [], usedStashBox: null, fallbackCount: 0, searchedQuery: "" },
  dashboardStats: { total: 0, active: 0, completed: 0, downloading: 0, pendingScans: 0, failed: 0 },
  releaseReviews: ({ input }: { input: { status?: string; page?: number } }) => ({
    totalCount: 2, pendingCount: 2, failedCount: 0, page: input.page ?? 1, pageSize: 24, totalPages: 1,
    performers: [{ id: "review-actor", name: "Example performer", pendingCount: 2 }],
    items: [
      { key: "review-1", code: "ABCVR-123", title: "A release recorded for manual review", date: "2026-01-12", url: "https://example.com/releases/1", imageUrl: null, source: "stash-box:https://example.com/graphql", vr: true, status: "PENDING", reasonCode: "vr_behavior_review", error: null, taskId: null, seenAt: now, reviewedAt: null, performerNames: ["Example performer"], performers: [{ id: "review-actor", name: "Example performer" }] },
      { key: "review-2", code: "ABC-456", title: "A second release awaiting a download decision", date: null, url: null, imageUrl: null, source: "stash-box:https://example.com/graphql", vr: false, status: "PENDING", reasonCode: "group_behavior_review", error: null, taskId: null, seenAt: now, reviewedAt: null, performerNames: ["Example performer", "Another performer"], performers: [{ id: "review-actor", name: "Example performer" }] }
    ]
  }),
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
        soloBehavior: "DOWNLOAD", groupBehavior: "REVIEW", compilationBehavior: "BLOCK", vrBehavior: "REVIEW",
        maxGroupPerformerCount: 5, releaseDateRange: "ALL",
      },
      torrentSelection: { enabled: false, inspectionCandidateLimit: 5, fastRules: [], torrentRules: [] },
    },
    system: { releaseTimezone:"auto",effectiveReleaseTimezone:"Asia/Shanghai",
      developerMode: false,
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
  homeSubscriptionChecks: { enabled: true, nextRunAt: "2026-01-15T18:00:00Z", activeCount: 0, latestScheduled: null, activeRuns: [], recentRuns: [] },
  subscriptionCheckActivity: { activeCount: 0, allRunId: null, performers: [] },
  subscriptionCheck: null,
  performerWorkspace: {
    performers: {
      items: [], page: 1, pageSize: 24, totalCount: 0, totalPages: 1,
      hasPrevPage: false, hasNextPage: false,
    },
    subscribedPerformers: [],
  },
};

export const settingsSnapshot = rootValue.settings;

export const test = base.extend<{ graphqlOverrides: Record<string, unknown> }>({
  graphqlOverrides: [{}, { option: true }],
  page: async ({ page, graphqlOverrides }, use) => {
    const errors: string[] = [];
    page.on("pageerror", (error) => errors.push(error.message));
    page.on("console", (message) => {
      if (message.type() === "error") errors.push(message.text());
    });
    await page.clock.setFixedTime(new Date(now));
    const effective = { ...rootValue, ...graphqlOverrides } as Record<string, any>;
    const cursor = () => effective.taskSnapshotCursor?.() ?? {sequence:0,streamId:"visual"};
    const taskDefaults = (task: any) => ({version:"00000000000000000001",revision:1,stashSceneId:task.stashSceneIds?.[0] ?? null,...task});
    const taskArray = () => (typeof effective.tasks === "function" ? effective.tasks() : effective.tasks) as any[];
    effective.homeTaskSnapshot ??= { ...cursor(),...effective.homeOverview };
    effective.taskDetailSnapshot ??= ({id}:{id:string}) => ({...cursor(),task: (()=>{const task=typeof effective.task === "function" ? effective.task({id}) : effective.task;return task?taskDefaults(task):null})()});
    effective.taskSummaries ??= ({ids}:{ids:string[]}) => taskArray().filter(task=>ids.includes(task.id)).map(taskDefaults);
    const group=(task:any)=>task.stage==="COMPLETED"?"COMPLETED":task.stageStatus==="BLOCKED"?"ATTENTION":task.stage==="SOURCING"&&task.stageStatus==="PENDING"&&task.sourcingNextCheckAt?"WAITING":"ACTIVE";
    effective.taskAttentionQueue ??= () => ({...cursor(),items:taskArray().filter(task=>group(task)==="ATTENTION").sort((a,b)=>a.createdAt.localeCompare(b.createdAt)||a.id.localeCompare(b.id)).map(taskDefaults)});
    effective.taskPage ??= ({input}:{input:{page:number;pageSize:number;status:string;sort:string;search?:string;completedWithin24Hours?:boolean;ids?:string[]}}) => {
      input=Object.assign({page:1,pageSize:24,status:"ALL",sort:"CREATED"},input);
      const all=taskArray(); let items=all.filter(task=>(input.status==="ALL"||group(task)===input.status)&&(!input.ids||input.ids.includes(task.id))&&(!input.search||`${task.code} ${task.torrentName??""}`.toLowerCase().includes(input.search.toLowerCase())));
      if(input.completedWithin24Hours)items=items.filter(task=>task.stage==="COMPLETED"&&Date.parse(task.ingestCompletedAt)>=Date.parse(now)-86400000);
      items.sort((a,b)=> input.sort==="PROGRESS"?(b.progress-a.progress||a.id.localeCompare(b.id)):input.sort==="UPDATED"?b.updatedAt.localeCompare(a.updatedAt)||a.id.localeCompare(b.id):b.createdAt.localeCompare(a.createdAt)||a.id.localeCompare(b.id));
      const totalPages=Math.max(1,Math.ceil(items.length/input.pageSize)); const page=Math.min(input.page,totalPages);
      const completed=all.filter(t=>group(t)==="COMPLETED").length,attention=all.filter(t=>group(t)==="ATTENTION").length,waiting=all.filter(t=>group(t)==="WAITING").length;
      return {items:items.slice((page-1)*input.pageSize,page*input.pageSize).map(taskDefaults),page,pageSize:input.pageSize,totalPages,totalCount:items.length,...cursor(),stats:{total:all.length,active:all.length-completed,completed,attention,waiting,downloading:0,pendingScans:0,failed:attention}};
    };
    await page.addInitScript(() => {
      const originalFetch = window.fetch.bind(window);
      window.fetch = (input, init) => {
        const url = typeof input === "string" ? input : input instanceof URL ? input.href : input.url;
        if (url.includes("/graphql")) {
          const headers = new Headers(init?.headers ?? (input instanceof Request ? input.headers : undefined));
          headers.set("x-moji-test-locale", localStorage.getItem("moji:locale") ?? "auto");
          init = { ...init, headers };
        }
        return originalFetch(input, init);
      };
    });
    await page.route("**/graphql", async (route) => {
      const request = route.request().postDataJSON();
      // Locale-oriented tests use their requested browser preference unless
      // an explicit saved instance language was supplied by the test.
      const browserPreference = route.request().headers()["x-moji-test-locale"] ?? "auto";
      const settings = typeof effective.settings === "function" ? effective.settings() : effective.settings;
      const requestRoot = { ...effective, settings: { ...settings, system: { ...settings.system, language: settings.system.language ?? browserPreference } } };
      const result = graphqlSync({
        schema, rootValue: requestRoot, source: request.query,
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
        if (message.type === "subscribe" && message.payload?.query?.includes("subscription TaskEvents")) socket.send(JSON.stringify({id:message.id,type:"next",payload:{data:{taskEvents:{__typename:"TaskEvent",type:"READY",sequence:0,streamId:"visual",taskId:"",listChanged:false,task:null,dashboardStats:null}}}}));
        if (message.type === "ping") socket.send(JSON.stringify({ type: "pong" }));
        if (message.type === "subscribe" && message.payload?.query?.includes("subscription TorrentSearchEvents")) {
          const result = graphqlSync({schema,rootValue:{...rootValue,...graphqlOverrides},source:message.payload.query,variableValues:message.payload.variables,operationName:message.payload.operationName});
          errors.push(...(result.errors??[]).map(error=>error.message));socket.send(JSON.stringify({id:message.id,type:"next",payload:result}));
        }
        if (message.type === "subscribe" && message.payload?.query?.includes("subscription IndexerEvents")) {
          const result = graphqlSync({ schema, rootValue: { ...rootValue, ...graphqlOverrides }, source: message.payload.query, variableValues: message.payload.variables, operationName: message.payload.operationName });
          errors.push(...(result.errors ?? []).map(error => error.message));
          socket.send(JSON.stringify({ id: message.id, type: "next", payload: result }));
        }
        if (message.type === "subscribe" && message.payload?.query?.includes("subscription DatabasePerformanceDocument")) {
          const result = graphqlSync({ schema, rootValue: { ...rootValue, ...graphqlOverrides }, source: message.payload.query, variableValues: message.payload.variables, operationName: message.payload.operationName });
          errors.push(...(result.errors ?? []).map(error => error.message));
          socket.send(JSON.stringify({ id: message.id, type: "next", payload: result }));
        }
        if (message.type === "subscribe" && (message.payload?.query?.includes("subscription IndexerDefinitionEvents") || message.payload?.query?.includes("subscription HomeIndexerDefinitionEvents"))) {
          const result = graphqlSync({ schema, rootValue: { ...rootValue, ...graphqlOverrides }, source: message.payload.query, variableValues: message.payload.variables, operationName: message.payload.operationName });
          errors.push(...(result.errors ?? []).map(error => error.message));
          socket.send(JSON.stringify({ id: message.id, type: "next", payload: result }));
        }
      });
    });
    await use(page);
    expect(errors, "Page and GraphQL fixture errors must not become screenshot baselines").toEqual([]);
  },
});
