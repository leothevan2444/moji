# Regenerates GraphQL files
.PHONY: generate
generate: generate-stash-client generate-stashbox-client
generate:

.PHONY: generate-stash-client
generate-stash-client:
	go run github.com/Yamashou/gqlgenc generate --configdir graphql/stash

.PHONY: generate-stashbox-client
generate-stashbox-client:
	go run github.com/Yamashou/gqlgenc generate --configdir graphql/stashbox

# Cleans up generated GraphQL files
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