# Generate files
.PHONY: generate
generate: generate-graphql generate-sql

# Generates GraphQL files
.PHONY: generate-graphql
generate-graphql: generate-stash-client generate-stashbox-client

.PHONY: generate-stash-client
generate-stash-client:
	go run github.com/Yamashou/gqlgenc generate --configdir graphql/stash

.PHONY: generate-stashbox-client
generate-stashbox-client:
	go run github.com/Yamashou/gqlgenc generate --configdir graphql/stashbox

# Generates SQL files
.PHONY: generate-sql
generate-sql: generate-r18dev

.PHONY: generate-r18dev
generate-r18dev:
	go run github.com/sqlc-dev/sqlc/cmd/sqlc generate -f ./pkg/r18dev/pg/sqlc.yaml

# Cleans up generated files
.PHONY: clean-generated
clean-generated:
	rm -rf ./pkg/stash/graphql
	rm -rf ./pkg/stashbox/graphql

# Builds the moji binary
.PHONY: moji
moji:
	go build ./cmd/moji

.PHONY: build
build: moji

.PHONY: clean
clean:
	rm moji