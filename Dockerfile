# syntax=docker/dockerfile:1
FROM node:22-bookworm-slim AS web
ARG VERSION
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN test -n "$VERSION" && MOJI_BUILD_VERSION="$VERSION" npm run build

FROM golang:1.26-bookworm AS backend
WORKDIR /src
ENV CGO_ENABLED=0 GOTOOLCHAIN=local
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY pkg/ ./pkg/
COPY version.go ./
ARG VERSION
# Make, Compose and CI resolve VERSION from Git before sending the build context.
RUN test -n "$VERSION" \
    && go build -trimpath -ldflags "-s -w -X github.com/leothevan2444/moji.releaseVersion=$VERSION" -o /out/moji ./cmd/moji \
    && test "$(/out/moji --version)" = "moji $VERSION"

FROM debian:bookworm-slim AS runtime
ARG VERSION
ARG VCS_REF=unknown
LABEL org.opencontainers.image.title="Moji" \
      org.opencontainers.image.description="Moji with web UI, FFmpeg and ffprobe" \
      org.opencontainers.image.version="$VERSION" \
      org.opencontainers.image.revision="$VCS_REF"
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates curl ffmpeg tini tzdata \
    && rm -rf /var/lib/apt/lists/* \
    && ffmpeg -version >/dev/null \
    && ffprobe -version >/dev/null \
    && groupadd --gid 1000 moji \
    && useradd --uid 1000 --gid moji --home-dir /data --no-create-home moji \
    && mkdir -p /data /downloads /library \
    && chown moji:moji /data /downloads /library
COPY --from=backend /out/moji /usr/local/bin/moji
COPY --from=web /src/web/dist/ /opt/moji/web/dist/
COPY docker/config.example.yaml /opt/moji/config.example.yaml
ENV MOJI_CONFIG=/data/config.yaml \
    MOJI_WEB_DIST=/opt/moji/web/dist \
    TMPDIR=/data
WORKDIR /data
USER 1000:1000
EXPOSE 10000
HEALTHCHECK --interval=30s --timeout=5s --start-period=30s --retries=3 \
    CMD curl --fail --silent --show-error http://127.0.0.1:10000/healthz || exit 1
ENTRYPOINT ["/usr/bin/tini", "--", "/usr/local/bin/moji"]
CMD ["-addr", ":10000"]
