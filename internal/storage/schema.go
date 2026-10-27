package storage

import _ "embed"

//go:embed schema/core.sql
var coreSchema string

//go:embed schema/core_v2.sql
var coreV2 string

//go:embed schema/runtime.sql
var runtimeSchema string

//go:embed schema/cache.sql
var cacheSchema string

func CoreMigrations() []Migration {
	return []Migration{{1, "normalized durable workflow", coreSchema}, {2, "durable resolved download location", coreV2}}
}
func RuntimeMigrations() []Migration {
	return []Migration{{1, "replaceable observations", runtimeSchema}}
}
func CacheMigrations() []Migration { return []Migration{{1, "external catalog cache", cacheSchema}} }
func DefaultManagerOptions(directory string) ManagerOptions {
	return ManagerOptions{Directory: directory, CoreMigrations: CoreMigrations(), RuntimeMigrations: RuntimeMigrations(), CacheMigrations: CacheMigrations()}
}
