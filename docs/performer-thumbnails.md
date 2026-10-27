# Performer list thumbnails

The performer list requests `/api/images/{key}?variant=avatar`. The default image
URL still returns the original for performer details and other image consumers.
No GraphQL contract or existing image registration changes are required.

The avatar derivative fits within 192 × 320 pixels, keeps its aspect ratio and
does not upscale small images. Opaque images use JPEG quality 82; transparent
images use PNG. JPEG, PNG, GIF (first frame) and WebP sources can be resized.
AVIF retains the original proxy response because the current Go decoder set does
not support it.

Derivatives use a versioned `-avatar-v1` filename alongside the original, sharing
the existing disk budget, retention, clear-cache action and conditional HTTP
caching. Unknown variants are rejected. Disabling disk caching also disables
derivative persistence. Concurrent requests for the same derivative share one
generation; different derivatives share original fetching and at most two source
images are decoded at once. Sources above 32 million pixels are rejected before
full decoding.

Validation:

```sh
GOCACHE=/tmp/moji-go-cache go test ./internal/imagecache
GOCACHE=/tmp/moji-go-cache go test -race ./internal/imagecache
npm --prefix web test -- src/utils/performerUtils.test.ts src/pages/PerformersPage.test.tsx
web/scripts/test-visual-docker.sh performer-avatars.spec.ts
```

The browser regression uses synthetic data to verify that the list requests a
thumbnail and performer details request the original, on desktop and mobile.
