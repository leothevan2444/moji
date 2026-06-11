# Shared command settings
GO ?= go
GOCACHE ?= /tmp/moji-go-cache
GO_RUN = GOCACHE=$(GOCACHE) $(GO) run

# Generate files
.PHONY: generate
generate: generate-graphql generate-sql

# Generates GraphQL files
.PHONY: generate-graphql
generate-graphql: generate-stash-client generate-stashbox-client generate-moji-graphql

.PHONY: generate-stash-client
generate-stash-client:
	$(GO_RUN) github.com/Yamashou/gqlgenc generate --configdir graphql/stash

.PHONY: generate-stashbox-client
generate-stashbox-client:
	$(GO_RUN) github.com/Yamashou/gqlgenc generate --configdir graphql/stashbox

.PHONY: generate-moji-graphql
generate-moji-graphql:
	@if command -v gqlgen >/dev/null 2>&1; then \
		GOCACHE=$(GOCACHE) gqlgen generate --config graphql/moji/gqlgen.yml; \
	else \
		$(GO_RUN) github.com/99designs/gqlgen generate --config graphql/moji/gqlgen.yml; \
	fi

# Generates SQL files
.PHONY: generate-sql
generate-sql: generate-r18dev

.PHONY: generate-r18dev
generate-r18dev:
	$(GO_RUN) github.com/sqlc-dev/sqlc/cmd/sqlc generate -f ./pkg/r18dev/pg/sqlc.yaml

# Cleans up generated files
.PHONY: clean-generated
clean-generated:
	rm -rf ./pkg/stash/graphql
	rm -rf ./pkg/stashbox/graphql
	rm -rf ./internal/graphqlapi/generated
	rm -rf ./internal/graphqlapi/model

# Builds the moji binary
.PHONY: moji
moji:
	go build ./cmd/moji

.PHONY: web-install
web-install:
	npm --prefix web install

.PHONY: web-dev
web-dev:
	npm --prefix web run dev

.PHONY: web-build
web-build:
	npm --prefix web run build

.PHONY: web-codegen
web-codegen:
	npm --prefix web run codegen

.PHONY: build
build: moji

.PHONY: clean
clean:
	rm moji
