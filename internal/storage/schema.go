package storage

import _ "embed"

//go:embed schema/core.sql
var coreSchema string

//go:embed schema/core_v2.sql
var coreV2 string

//go:embed schema/core_v3.sql
var coreV3 string

//go:embed schema/core_v4.sql
var coreV4 string

//go:embed schema/core_v5.sql
var coreV5 string

//go:embed schema/core_v6.sql
var coreV6 string

//go:embed schema/runtime.sql
var runtimeSchema string

//go:embed schema/runtime_v2.sql
var runtimeV2 string

//go:embed schema/runtime_v3.sql
var runtimeV3 string

//go:embed schema/runtime_v4.sql
var runtimeV4 string

//go:embed schema/runtime_v5.sql
var runtimeV5 string

//go:embed schema/runtime_v6.sql
var runtimeV6 string

//go:embed schema/cache.sql
var cacheSchema string

func CoreMigrations() []Migration {
	return []Migration{{1, "normalized durable workflow", coreSchema}, {2, "durable resolved download location", coreV2}, {3, "upcoming subscription releases", coreV3}, {4, "runtime indexer definition bundles", coreV4}, {5, "durable selected indexer provenance", coreV5}, {6, "accepted indexer resource snapshots", coreV6}}
}
func RuntimeMigrations() []Migration {
	return []Migration{{1, "replaceable observations", runtimeSchema}, {2, "isolated indexer sessions", runtimeV2}, {3, "server-owned indexer resources", runtimeV3}, {4, "scoped indexer observations", runtimeV4}, {5, "persistent interactive torrent searches", runtimeV5}, {6, "persistent origin egress cooldowns", runtimeV6}}
}
func CacheMigrations() []Migration { return []Migration{{1, "external catalog cache", cacheSchema}} }
func DefaultManagerOptions(directory string) ManagerOptions {
	return ManagerOptions{Directory: directory, CoreMigrations: CoreMigrations(), RuntimeMigrations: RuntimeMigrations(), CacheMigrations: CacheMigrations()}
}
