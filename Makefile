# Shared command settings
GO ?= go
GOCACHE ?= /tmp/moji-go-cache
GO_RUN = GOCACHE=$(GOCACHE) $(GO) run
override VERSION := $(strip $(shell cat VERSION))
IMAGE ?= moji

# Generate files
.PHONY: generate
generate: generate-graphql

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
	npm --prefix web run codegen

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
	GOCACHE=$(GOCACHE) $(GO) build -trimpath ./cmd/moji

.PHONY: moji-run
moji-run:
	go run ./cmd/moji

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

.PHONY: test-media
test-media:
	@command -v ffmpeg >/dev/null || (echo "ffmpeg is required" >&2; exit 1)
	@command -v ffprobe >/dev/null || (echo "ffprobe is required" >&2; exit 1)
	GOCACHE=$(GOCACHE) $(GO) test ./internal/mediafiles

.PHONY: docker-build
docker-build:
	docker build --build-arg VERSION=$(VERSION) --build-arg VCS_REF=$(shell git rev-parse HEAD) -t $(IMAGE):$(VERSION) .

.PHONY: docker-test
docker-test:
	./docker/smoke-test.sh $(IMAGE):$(VERSION)
